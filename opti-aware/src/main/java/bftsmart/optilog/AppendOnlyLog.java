package bftsmart.optilog;

import bftsmart.consensus.Decision;
import bftsmart.optilog.monitors.LatencyMonitor;
import bftsmart.optilog.monitors.SuspicionMonitor;
import bftsmart.reconfiguration.ServerViewController;
import bftsmart.tom.core.messages.TOMMessage;
import bftsmart.tom.core.messages.TOMMessageType;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


public class AppendOnlyLog {

    private static AppendOnlyLog instance;

    private final Logger APPEND_TO_LOG = LoggerFactory.getLogger(this.getClass());

    private ServerViewController svc;

    public static AppendOnlyLog getInstance(ServerViewController svc) {
        if (instance == null) {
            instance = new AppendOnlyLog(svc);
        }
        return AppendOnlyLog.instance;
    }

    private AppendOnlyLog(ServerViewController controller) {
        svc = controller;
    }

    public void commit(Decision decision) {
        APPEND_TO_LOG.info("OptiLog >> AppendOnlyLog: Append to log: decision:" + decision.getConsensusId());
        for (TOMMessage tm : decision.getDeserializedValue()) {
            if (tm.getIsMonitoringMessage()) {
                APPEND_TO_LOG.trace("Consensus outputs monitoring message, " + tm.toString());
                switch (TOMMessageType.fromInt(tm.getIsMonitoringType())) {
                    case MEASUREMENT_LATENCY:
                        //APPEND_TO_LOG.info("Received from consensus: LATENCY monitoring message from " + tm.getSender()); // Todo outcomment later
                        LatencyMonitor.getInstance(svc)
                                .notify(tm.getSender(), tm.getContent(), decision.getConsensusId());
                        break;
                    case MEASUREMENT_SUSPICION:
                        SuspicionMonitor.getInstance(svc)
                                .notify(tm.getSender(), tm.getContent(), decision.getConsensusId());
                        break;
                    case MEASUREMENT_MISBEHAVIOR:
                        APPEND_TO_LOG.error("MISBEHAVIOR MONITOR NOT IMPLEMENTED");
                        // Todo Implement later
                        break;
                }
                //onReceiveMonitoringInformation(tm.getSender(), tm.getContent(), decision.getConsensusId());

            } else { // TOMMessage is type client command
                APPEND_TO_LOG.trace("Consensus output client command " + tm.getReqType().toInt());
            }
        }

    }

}
