package bftsmart.optilog;


import bftsmart.consensus.Decision;
import bftsmart.tom.core.messages.TOMMessage;


public interface Monitor {

    // Updates the monitor with consistent measurements output from the consensus engine
    public void notify(int senderReplicaId, byte[] measurement, int consensusInstance);
}
