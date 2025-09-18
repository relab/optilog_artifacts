package bftsmart.optilog.monitors;

import bftsmart.optilog.SensorApp;
import bftsmart.optilog.sensors.LatencyMeasurement;
import bftsmart.optilog.sensors.SuspicionMeasurement;
import bftsmart.optilog.sensors.SuspicionSensor;
import bftsmart.optilog.sensors.SuspicionType;
import bftsmart.reconfiguration.ServerViewController;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collections;
import java.util.List;
import java.util.LinkedList;
import java.util.*;

import bftsmart.optilog.Monitor;


public class SuspicionMonitor implements Monitor {


    private static SuspicionMonitor instance;
    private final Logger logger = LoggerFactory.getLogger(this.getClass());
    private ServerViewController controller;

    private final List<SuspicionMeasurement> suspicions = new LinkedList<>();

    private SuspicionGraph suspicionGraph;

    // For cleaning up old suspicions perodically:
    private int lastConsensusRemovedOldSuspicions = 0;
    private int removedOldSuspicions = 0;

    private SuspicionMonitor(ServerViewController svc) {
        this.controller = svc;
        this.suspicionGraph = new SuspicionGraph(controller);
    }


    public static SuspicionMonitor getInstance(ServerViewController svc) {
        if (instance == null) {
            instance = new SuspicionMonitor(svc);
        }
        return instance;
    }

    @Override
    public synchronized void notify(int reporter, byte[] measurement, int consensusInstance) {

        SuspicionMeasurement suspicion = SuspicionMeasurement.fromBytes(measurement, reporter);
        synchronized (suspicions) {
            suspicions.add(suspicion);
        }
        if (suspicion.getSuspect() == controller.getStaticConf().getProcessId()) {
            logger.debug(">>SUSPICION Received: I was suspected by process: " + suspicion.getSuspect());
        }

        // suspicionGraph.handleSuspicion(reporter, suspicion);

        if (suspicion.getType() == SuspicionType.SLOW) {
            SensorApp.getInstance(controller).getSuspicionSensor().returnSuspicionIfFalselyAccused(suspicion, reporter, consensusInstance, suspicion.getProtocolMessageType());
        }
        // Proof of Concept for testing:
        if (controller.getStaticConf().getProcessId() == 1) { // For more comprehensive logs, let only first process output to LOG a Warning
            suspicionGraph.printGraphAscii();
        } else {
            logger.trace(">> OptiLog > SuspicionMonitor: New suspicion recorded: {}", suspicion);
        }

    }

    public synchronized void notify(int consensusInstance) {
        // Todo: Here we "clean up" the suspicion graph, by weakening and removing "old" suspicions
        // Todo:  Replace later by a more refined implementation
       // if ((consensusInstance - lastConsensusRemovedOldSuspicions) / 100 > removedOldSuspicions) {
       //     suspicionGraph.removeSuspicions(0.1 * (consensusInstance - lastConsensusRemovedOldSuspicions));
       //     removedOldSuspicions++;
       //     lastConsensusRemovedOldSuspicions = consensusInstance;
       // }
    }

    public synchronized boolean graphContainsSuspicion(int reporter, int suspect) {
        return suspicionGraph.existsEdge(reporter, suspect);
    }

    public synchronized Set<Integer> computeCandidateSet() {
        buildSuspicionGraph();
        return suspicionGraph.candidateSet();
    }

    public void buildSuspicionGraph() {
        List<SuspicionMeasurement> filteredSuspicions = new LinkedList<>();
        synchronized (suspicions) {
            Collections.sort(suspicions);
            Map<Integer, List<SuspicionMeasurement>> consensusInstancesToSuspicions = new HashMap<>();

            for (SuspicionMeasurement s : suspicions) {
                int consensusID = s.getConsensusID();  // or s.consensusID if it's accessible
                consensusInstancesToSuspicions.computeIfAbsent(consensusID, k -> new ArrayList<>()).add(s);
            }

            // boolean lastRoundWasDelayed=false; TODO
            for (Map.Entry<Integer, List<SuspicionMeasurement>> consensus : consensusInstancesToSuspicions.entrySet()) {
                Integer consensusID = consensus.getKey();
                List<SuspicionMeasurement> suspicionsWithinInstance = consensus.getValue();

                Collections.sort(suspicionsWithinInstance);
                Map<Integer, List<SuspicionMeasurement>> phasesToSuspicions = new HashMap<>();
                for (SuspicionMeasurement s : suspicionsWithinInstance) {
                    char phase = s.getProtocolMessageType();
                    phasesToSuspicions.computeIfAbsent(SuspicionMeasurement.protocolOrder(phase), k -> new ArrayList<>()).add(s);
                }
                boolean done = false;
                for (Map.Entry<Integer, List<SuspicionMeasurement>> phase : phasesToSuspicions.entrySet()) {
                    if (!done) {
                        Integer phaseID = phase.getKey();
                        List<SuspicionMeasurement> suspicionsWithinPhase = phase.getValue();
                        if (!suspicionsWithinPhase.isEmpty()) {
                            filteredSuspicions.addAll(suspicionsWithinPhase);
                            done = true;
                        }
                    }
                }
            }
        }
        suspicionGraph.populate(filteredSuspicions);
    }

    public List<SuspicionMeasurement> getSuspicions() {
        return suspicions;
    }

    /**
     * Bounds memory consumption by garbage collecting old suspicions that have been populated into the suspicion graph
     * Important: This method does NOT remove suspicions from the suspicion graph
     */
    public int garbageCollect(int index) {
        synchronized (suspicions) {
            Collections.sort(suspicions);
            int cutIndex = 0;
            for (SuspicionMeasurement s : suspicions) {
                if (s.getConsensusID() >= index) {
                    break;
                }
                cutIndex++;
            }
            suspicions.subList(0, cutIndex).clear();
            return cutIndex;
        }
    }

}
