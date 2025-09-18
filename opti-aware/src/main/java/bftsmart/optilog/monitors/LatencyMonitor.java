package bftsmart.optilog.monitors;

import bftsmart.optilog.sensors.LatencyMeasurement;
import bftsmart.optilog.Monitor;
import bftsmart.optilog.sensors.LatencySensor;
import bftsmart.reconfiguration.ServerViewController;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.HashMap;

/**
 * Singelton pattern. Only one instance of Monitor should be used
 *
 * @author cb
 */
public class LatencyMonitor implements Monitor {

    public static final long MISSING_VALUE = 1000000000000000L; // Long does not have an infinity value, but this
    // value is very large for a latency, roughly
    // 10.000 seconds and will be used

    // Singelton
    private static LatencyMonitor instance;

    private ServerViewController svc;

    // Stores and computes latency monitoring information
    private LatencySensor proposeLatencySensor;
    private LatencySensor writeLatencySensor;

    // Was only used for debugging: TODO: Remove later
    // The latencies the current process measures from its own perspective
    // private Long[] freshestProposeLatencies;
    // private Long[] freshestWriteLatencies;

    // The measured latency matrices which have been disseminated with total order
    // They are the same in all replicas for a defined consensus id, after all TOMMessages within this consensus
    // have been processed.
    private Long[][] m_propose;
    private Long[][] m_write;

    private HashMap<Integer, Integer> notifications;

    private final Logger logger = LoggerFactory.getLogger(this.getClass());


    private LatencyMonitor(ServerViewController svc) {

        this.svc = svc;

        // Todo; if system size changes, we need to handle this
        int n = svc.getCurrentViewN();

        // Initialize
        this.writeLatencySensor = new LatencySensor(svc);
        this.proposeLatencySensor = new LatencySensor(svc);

        notifications = new HashMap<>();

        clear(n);
        /*
        // Periodically compute point-to-point latencies
        Timer timer = new Timer();
        timer.scheduleAtFixedRate(new TimerTask() {
            @Override
            public void run() {
                // Computes the most recent point-to-point latency using the last 1000 (monitoring window) measurements
                // from consensus rounds
                freshestWriteLatencies = writeLatencySensor.create_L("WRITE");
                freshestProposeLatencies = proposeLatencySensor.create_L("PROPOSE");
            }
        }, MONITORING_DELAY, MONITORING_PERIOD);
        */

    }

    /**
     * Use this method to get the monitor
     *
     * @param svc server view controller
     * @return the monitoring instance
     */
    public static LatencyMonitor getInstance(ServerViewController svc) {
        if (LatencyMonitor.instance == null) {
            LatencyMonitor.instance = new LatencyMonitor(svc);
        }
        return LatencyMonitor.instance;
    }

    @Override
    public synchronized void notify(int sender, byte[] measurement, int consensusID) {
        int n = svc.getCurrentViewN();

        LatencyMeasurement li = LatencyMeasurement.fromBytes(measurement);

        m_write[sender] = li.writeLatencies;
        m_propose[sender] = li.proposeLatencies;

        notifications.put(sender, consensusID);

        // printM("Updated", m_write, consensusID, n); // Todo outcomment later
    }

    /**
     * Assume communication link delays are symmetric and use the maximum
     *
     * @param m latency matrix
     * @return sanitized latency matrix
     */
    public synchronized Long[][] sanitize(Long[][] m) {
        int n = svc.getCurrentViewN();
        Long[][] m_ast = new Long[n][n];
        for (int i = 0; i < n; i++) {
            for (int j = 0; j < n; j++) {
                m_ast[i][j] = Math.max(m[i][j], m[j][i]);
            }
        }
        return m_ast;
    }

    private static void printM(String description, Long[][] matrix, int consensusID, int n) {
        String result = "";
        result += ("--------------- " + description + " ---------------------\n");
        result += ("Sever Latency Matrix for consensus ID " + consensusID + "\n");
        result += ("----------------------------------------------------------\n");
        result += ("       0       1       2        3        4        ....    \n");
        result += ("----------------------------------------------------------\n");
        for (int i = 0; i < n; i++) {
            result = result + i + " | ";
            for (int j = 0; j < n; j++) {

                double latency = Math.round((double) matrix[i][j] / 1000.00); // round to precision of micro seconds
                latency = latency / 1000.00; // convert to milliseconds
                if (latency >= 0.00 & latency < 1.0E9)
                    result += ("  " + latency + "  ");
                else
                    result += ("  silent  ");
            }
            result += "\n";
        }
        System.out.println(result);
    }

    public synchronized Long[][] getM_propose() {
        return m_propose;
    }

    public synchronized Long[][] getM_write() {
        return m_write;
    }

    public synchronized void clear(int n) {
        this.m_propose = new Long[n][n];
        this.m_write = new Long[n][n];
        for (int i = 0; i < n; i++) {
            for (int j = 0; j < n; j++) {
                m_write[i][j] = MISSING_VALUE;
                m_propose[i][j] = MISSING_VALUE;
            }
        }
        notifications = new HashMap<>();
    }

    public synchronized boolean isInitialized() {
        return notifications.keySet().size() >= (svc.getCurrentViewN() - svc.getCurrentViewF());
    }

}
