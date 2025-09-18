if ! java -version 2>&1 | grep -q "11"; then
    echo "Installing OpenJDK 11 first..."
    sudo apt update && sudo apt install -y openjdk-11-jdk
else
    echo "Compiling.."
fi
gcc -shared -o libptp.so -fPIC ptp_time.c \
  -I/usr/lib/jvm/java-11-openjdk-amd64/include \
  -I/usr/lib/jvm/java-11-openjdk-amd64/include/linux \
  -lc
ls -lh libptp.so
echo "Moving libptp.so to /usr/lib/"
sudo mv libptp.so /usr/lib/
javac -h . PTPClock.java
java -cp ../../../ bftsmart.optilog.PrecisionClock.PTPClock

