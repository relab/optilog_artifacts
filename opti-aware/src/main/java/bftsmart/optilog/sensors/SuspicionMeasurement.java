package bftsmart.optilog.sensors;

import java.io.*;
import java.util.Objects;

import java.io.*;
import java.util.Objects;

public class SuspicionMeasurement implements Serializable, Comparable<SuspicionMeasurement> {
    private int suspect;        // Suspected process ID
    private SuspicionType type; // Either SLOW or FALSE
    private char protocolMessageType; // "P" = Propose, "W" = Write, "A"=Accept
    private int consensusID; // Round in which the suspicion was made

    private transient int reporter = -1;       // ID of replica to report this suspicion

    // Constructor
    public SuspicionMeasurement(int suspect, SuspicionType type, char protocolMessageType, int consensusID) {
        this.suspect = suspect;
        this.type = type;
        this.protocolMessageType = protocolMessageType;
        this.consensusID = consensusID;
    }

    // Getters
    public int getSuspect() {
        return suspect;
    }

    public SuspicionType getType() {
        return type;
    }

    public char getProtocolMessageType() {
        return protocolMessageType;
    }

    public int getConsensusID() {
        return consensusID;
    }

    // Setters
    public void setSuspect(int suspect) {
        this.suspect = suspect;
    }

    public void setType(SuspicionType type) {
        this.type = type;
    }

    public void setProtocolMessageType(char protocolMessageType) {
        this.protocolMessageType = protocolMessageType;
    }

    public void setConsensusID(int consensusID) {
        this.consensusID = consensusID;
    }

    public void setReporter(int reporter) {
        this.reporter = reporter;
    }

    public int getReporter() {
        return reporter;
    }

    // Convert SuspicionMeasurement to byte array using DataOutputStream
    public static byte[] toBytes(SuspicionMeasurement measurement) {
        try (ByteArrayOutputStream baos = new ByteArrayOutputStream();
             DataOutputStream dos = new DataOutputStream(baos)) {

            dos.writeInt(measurement.getSuspect()); // Write suspect (int)
            dos.writeInt(measurement.getType().getCode()); // Write type as int
            dos.writeChar(measurement.getProtocolMessageType());
            dos.writeInt(measurement.getConsensusID());

            return baos.toByteArray(); // Return serialized byte array

        } catch (IOException e) {
            throw new RuntimeException("Error serializing SuspicionMeasurement", e);
        }
    }

    // Convert byte array back to SuspicionMeasurement using DataInputStream
    public static SuspicionMeasurement fromBytes(byte[] measurement, int reporter) {
        try (ByteArrayInputStream bais = new ByteArrayInputStream(measurement);
             DataInputStream dis = new DataInputStream(bais)) {

            int suspect = dis.readInt(); // Read suspect
            int typeCode = dis.readInt(); // Read type as int
            char protocolMessageType = dis.readChar();
            int consensusID = dis.readInt();
            SuspicionType type = SuspicionType.fromCode(typeCode); // Convert int to enum
            SuspicionMeasurement sm = new SuspicionMeasurement(suspect, type, protocolMessageType, consensusID);
            sm.setReporter(reporter);
            return sm;

        } catch (IOException e) {
            throw new RuntimeException("Error deserializing SuspicionMeasurement", e);
        }
    }

    // toString method for debugging
    @Override
    public String toString() {
        return "SuspicionMeasurement{" +
                "suspect=" + suspect +
                "reporter=" + reporter +
                ", type=" + type +
                ", protocolMessageType=" + protocolMessageType +
                ", consensusID=" + consensusID +
                '}';
    }

    // equals and hashCode for object comparison
    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        SuspicionMeasurement that = (SuspicionMeasurement) o;
        return suspect == that.suspect
                && type == that.type
                && reporter == that.reporter
                && consensusID == that.consensusID
                && protocolMessageType == that.protocolMessageType;
    }

    @Override
    public int hashCode() {
        return Objects.hash(suspect, reporter, type, protocolMessageType, consensusID);
    }

    @Override
    public int compareTo(SuspicionMeasurement o) {
        int result = Integer.compare(this.consensusID, o.consensusID);
        if (result == 0) {
            // Custom order: P < W < A
            int thisOrder = protocolOrder(this.protocolMessageType);
            int otherOrder = protocolOrder(o.protocolMessageType);
            result = Integer.compare(thisOrder, otherOrder);
        }
        if (result == 0) {
            result = this.type.compareTo(o.type);
        }
        if (result == 0) {
            result = Integer.compare(this.suspect, o.suspect);
        }
        if (result == 0) {
            result = Integer.compare(this.reporter, o.reporter);
        }
        return result;
    }

    // Helper method
    public static int protocolOrder(char c) {
        switch (c) {
            case 'P':   // Propose
                return 0;
            case 'W':   // Write
                return 1;
            case 'A':   // Accept
                return 2;
            default:
                return 3; // Put unknown types last
        }
    }

    public static char protocolFromOrder(int order) {
        switch (order) {
            case 0:
                return 'P';  // Propose
            case 1:
                return 'W';  // Write
            case 2:
                return 'A';  // Accept
            default:
                return '?';  // Unknown / Invalid

        }
    }


    public static void main(String[] args) {
        // quick serialization test
        SuspicionMeasurement sm = new SuspicionMeasurement(0, SuspicionType.SLOW, 'P', 0);
        byte[] serialized = SuspicionMeasurement.toBytes(sm);
        System.out.println("serialized: " + serialized.length);
    }
}