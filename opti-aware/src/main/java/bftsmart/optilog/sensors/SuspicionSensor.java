package bftsmart.optilog.sensors;

import bftsmart.aware.decisions.Simulator;
import bftsmart.consensus.messages.ConsensusMessage;
import bftsmart.optilog.PrecisionClock.PTPClock;
import bftsmart.optilog.SensorApp;
import bftsmart.reconfiguration.ServerViewController;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Arrays;
import java.util.HashMap;


/**
 * This class allows to store and receive a replica's own  monitoring information. Note that all measurement data in here
 * is viewed by the perspective of what a single replica has measured recently by itself without a guarantee to be synchronized yet
 *
 * @author cb
 */
public class SuspicionSensor {

    private int window;
    private ServerViewController controller;

    private final Logger logger = LoggerFactory.getLogger(this.getClass());


    private long deltaRound = 0;        // delay until leader should start disseminating a proposal
                                        // in ms
    private static final long MEASUREMENT_ACCURACY = 4_000_000L;  // up to a delay of 4 ms, no supsicion should be raised
    private long consensusLatencyExpectation = 0;
    private Simulator.MessageDelays messageDelays;
    private HashMap<Integer, Long> proposalSentTimes;
    private HashMap<Integer, Long> proposalReceivedTimes;


    private SensorApp sensorapp;

    private boolean initialized = false;

    private long current_round = 0;
    private long current_round_proposal_sent_time = -1L;

    private boolean propose_delayed = false;
    private boolean write_delayed = false;
    private int lastDelayedRound = -1;

    /**
     * Creates a new instance of a latency monitor
     *
     * @param controller server view controller
     */
    public SuspicionSensor(ServerViewController controller, SensorApp sensorAppInstance) {
        this.window = controller.getStaticConf().getMonitoringWindow();
        this.controller = controller;

        proposalSentTimes = new HashMap<>();
        proposalReceivedTimes = new HashMap<>();

        sensorapp = sensorAppInstance;
        // init();
    }

    public synchronized boolean checkProposal(ConsensusMessage proposal) {
        if (!initialized || proposal == null || 0 >= proposal.getNumber() || messageDelays == null) {
            return true;
        }
        this.current_round = proposal.getNumber();
        this.current_round_proposal_sent_time = proposal.getSentTimestamp();
        this.propose_delayed = false;
        this.write_delayed = false;

        int consensusNumber = proposal.getNumber();

        long sentTime = proposal.getSentTimestamp();
        proposalSentTimes.put(consensusNumber, sentTime);

        //long receivedTime = PTPClock.precisionTimestamp();
        //proposalReceivedTimes.put(consensusNumber, receivedTime);

        long lastSentTime = -1;
        if (proposalSentTimes.get(consensusNumber - 1) != null) {
            lastSentTime = proposalSentTimes.get(consensusNumber - 1);
        } else {
            return true;
        }
        //logger.info("--Checking proposal for consensus " + proposal.getNumber());


        long roundTime = sentTime - lastSentTime;
        long currentTimeMillis = System.currentTimeMillis();
        boolean delayed = isDelayedProposal(sentTime, currentTimeMillis, roundTime, consensusNumber);
        //logger.info("--Round time: " + roundTime + "propose delayed: " + delayed);
        /* Uses that code to test functionality
        if (consensusNumber % 10 == 0) {
            StringBuilder s = new StringBuilder();
            for (Integer i : proposalSentTimes.keySet()) {
                s.append("(").append(i).append(",").append(proposalSentTimes.get(i)).append(")");
            }
            logger.info("Hashmap {}", s.toString());
            long now = PTPClock.precisionTimestamp();
            logger.info("PrecisionClock now() {}", now);
            logger.info("Delayed by {} ms, is it delayed? {}, RoundTime: {} ms, Expectation: {} ms, SentTime: {}, LastSentTime: {}",
                    (roundTime-deltaRound)/1000000, delayed, roundTime/1000000, this.consensusLatencyExpectation/1000000, sentTime, lastSentTime);
        }*/

        if (delayed && (consensusNumber % controller.getStaticConf().getCalculationInterval()) != controller.getStaticConf().getCalculationDelay() + 1) { // consensus message is delayed and did not follow-up a reconfiguration
            SuspicionMeasurement suspicion = new SuspicionMeasurement(proposal.getSender(), SuspicionType.SLOW, 'P', consensusNumber);
            sensorapp.publishSuspicion(suspicion);
        }
        return !delayed;
    }

    private boolean isDelayedProposal(long proposal_sent_time, long currentTime, long roundTime, int consensusRound) {
        long timeSinceLastRequestArrived = controller.tomLayer.clientsManager.getLastRequestArrivalTime();
        long timeDiff = currentTime - timeSinceLastRequestArrived; // Time since the last client request was received (in ms)

        long timeNow = PTPClock.precisionTimestamp();
        long observation = timeNow - proposal_sent_time;
        long expectation = messageDelays.proposedTime;


        // Raise a suspicion if a proposal arrives late..
        double delta = controller.getStaticConf().getSuspicionDelta();
        if (expectation*delta < observation - MEASUREMENT_ACCURACY) {
            this.propose_delayed = true;
            this.lastDelayedRound = consensusRound;
            logger.info("OptiLog >> SuspicionSensor >> Delayed proposal for consensus {}, I expected {} ns but I observed {} ns", consensusRound, expectation, observation);
            return true;
        }
        // Raise a suspicion if the observed round delay is  higher than the estimation (times some delta)
        // Dont raise a suspicion if no client request arrived within the last roundTime or last round was delayed already
        boolean delay = roundTime-MEASUREMENT_ACCURACY > deltaRound && timeDiff*1000000 < roundTime-MEASUREMENT_ACCURACY && consensusRound - 1 != lastDelayedRound;
        if (delay) {
            logger.info("OptiLog >> SuspicionSensor >> Delayed proposal from previous round before {}, roundtTime: {} ns ; deltaRound: {} ns", consensusRound, roundTime, deltaRound);
        }
        return delay;
    }

    public synchronized boolean checkVote(ConsensusMessage vote) {
        if (messageDelays == null) {
            return true;
        }
        int consensus = vote.getNumber();
        if (consensus != current_round) {
            return true;
        }
        if (propose_delayed) {
            return true;
        }
        long currentTime = PTPClock.precisionTimestamp(); // Todo time stamp a message upon receiving, not here...

        long observation = currentTime - current_round_proposal_sent_time;
        long expectation;

        if (vote.getPaxosVerboseType().equals("WRITE")) {
            expectation = messageDelays.writeArrivalTimes[vote.getSender()];
        } else {
            if (vote.getPaxosVerboseType().equals("ACCEPT")) {
                if (write_delayed) {
                    return true;
                }
                expectation = messageDelays.acceptArrivalTimes[vote.getSender()];
            } else {
                expectation = -1L;
                logger.error("Something went wrong when trying to check a vote");
                return true;
            }
        }

        boolean delayed = false;
        double delta = controller.getStaticConf().getSuspicionDelta();
        if (expectation*delta < observation-MEASUREMENT_ACCURACY) {
            delayed = true;
            if (vote.getPaxosVerboseType().equals("WRITE")) {
                write_delayed = true;
            }
        }

        if (delayed && (consensus % controller.getStaticConf().getCalculationInterval()) != controller.getStaticConf().getCalculationDelay() + 1) {
            // consensus message is delayed and did not follow-up a reconfiguration
            char protocolMessageType = vote.getPaxosVerboseType().equals("WRITE") ? 'W' : 'A';
            SuspicionMeasurement suspicion = new SuspicionMeasurement(vote.getSender(), SuspicionType.SLOW, protocolMessageType, consensus);
            //if (controller.)
            logger.info("OptiLog >> SuspicionSensor >> Delayed vote, expected arrival to be {} ns but I observed {} ns for consensus {}", expectation, observation, consensus);

            sensorapp.publishSuspicion(suspicion);
        }
        // Measure time milis now
        // compute time the leader proposed using
        // relative time = time now - leader propos time

        // IS the current message to late? Compare it with expectation in Message Delays..

        // Todo implement
        return !delayed;
    }

    public synchronized void notify(int consensus, int leader) {
        if (controller != null && initialized && consensus >= 100) { // todo remove 100 here, used for testing
            this.messageDelays = Simulator.predictMessageDelays(controller, leader);
            if (consensus % 100 == 0) {
                logger.info("[DEBUG] OptiLog >> SuspicionSensor: These are the current Message Delay Times:");
                logger.info("Proposal expectation: " + messageDelays.proposedTime/1_000_000);
                long[] writeTimes = Arrays.stream(messageDelays.writeArrivalTimes).map(l -> l / 1_000_000).toArray();
                logger.info("Write expectation: " + Arrays.toString(writeTimes));
                long[] acceptTimes = Arrays.stream(messageDelays.acceptArrivalTimes).map(l -> l / 1_000_000).toArray();
                logger.info("Accept expectation: " + Arrays.toString(acceptTimes));
            }
        }

    }

    public synchronized void returnSuspicionIfFalselyAccused(SuspicionMeasurement suspicion, int reporter, int consensusID, char messageTyp) {
        int me = controller.getStaticConf().getProcessId();
        if (initialized && suspicion.getSuspect() == me && reporter != me && suspicion.getType() == SuspicionType.SLOW ) { // I was suspected
            SuspicionMeasurement clarification = new SuspicionMeasurement(reporter, SuspicionType.FALSE, messageTyp, consensusID);
            sensorapp.publishSuspicion(clarification);
        }
    }

    public synchronized void setDeltaRound(long consensusLatencyExpectation) {
        this.consensusLatencyExpectation = consensusLatencyExpectation;
        this.deltaRound = (long) (consensusLatencyExpectation * controller.getStaticConf().getSuspicionDelta());
        initialized = true;
    }

    public synchronized void clear() {
        initialized = false;
    }

}