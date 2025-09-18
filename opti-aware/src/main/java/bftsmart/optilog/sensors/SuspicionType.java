package bftsmart.optilog.sensors;

public enum SuspicionType {
    SLOW(1),
    FALSE(2);

    private final int code;

    SuspicionType(int code) {
        this.code = code;
    }

    public int getCode() {
        return code;
    }

    public static SuspicionType fromCode(int code) {
        for (SuspicionType type : values()) {
            if (type.code == code) {
                return type;
            }
        }
        throw new IllegalArgumentException("Invalid code for SuspicionType: " + code);
    }
}
