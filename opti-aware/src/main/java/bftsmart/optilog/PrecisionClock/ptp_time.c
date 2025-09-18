#include "bftsmart_optilog_PrecisionClock_PTPClock.h"
#include <time.h>
#include <jni.h>
#include <fcntl.h>
#include <unistd.h>
#include <linux/ptp_clock.h>
#include <sys/ioctl.h>
#include <stdio.h>

#define PTP_DEVICE "/dev/ptp0"  // Default PTP device

JNIEXPORT jlong JNICALL Java_bftsmart_optilog_PrecisionClock_PTPClock_getPTPTimestamp(JNIEnv *env, jclass clazz) {
    struct timespec ts;
    int fd = open(PTP_DEVICE, O_RDONLY);

    if (fd >= 0) {
        struct ptp_sys_offset offset;
        offset.n_samples = 1; // Request a single timestamp

        // Attempt to read PTP timestamp
        if (ioctl(fd, PTP_SYS_OFFSET, &offset) == 0) {
            close(fd);
            return (jlong) offset.ts[1].sec * 1000000000L + offset.ts[1].nsec; // Convert to nanoseconds
        }

        close(fd);
    }

    // Fallback to system clock if PTP is unavailable, indicate this by returning -1
    return -1; // Error case
}
