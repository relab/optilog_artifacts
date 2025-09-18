Opti-AWARE 0.4 (Last updated 16/09/2025)
----------
This artifact is part of a [EuroSys](https://2026.eurosys.org/) Paper:

```
OptiLog: Assigning Roles in Byzantine Consensus. Hanish Gogada, Christian Berger, Leander Jehl, Hans P Reiser and Hein Meling. EuroSys 2026.
```


Specifically, this repository contains the source code for Opti-AWARE, an extension of AWARE and WHEAT/BFT-SMaRt that has OptiLog integrated.

*Important:
For evaluation purpose, we suggest to deploy Opti-AWARE in a wide-area network (WAN).*
In the following, we provide a description of how to (1) build, (2) configure, and (3) run this software.

----------

## Step 0: Environment Description

This tutorial is tested using the following operating system:

```
Description:	Ubuntu 24.04.3 LTS
Release:	24.04
Linux Kernel: 6.14.0-1012-aws
```

### Resource Use, Hardware Constraints, Expected Resource Use per Experiment
The test system specification for testing functionality is a ``t3.micro`` VM on AWS, but for performance evaluations we recommend ``c5.xlarge`` and above.

For *each* replica we recommend the following resources to be available on the VM:

Recommended system for performance measurements: More than 20 GB available disk space, 16 GB of RAM and 8 CPU cores.

Recommended system for testing functionality:  More than 8 GB available disk space, 1 GB of RAM and 1 CPU core.

Expected time for experiments: When conducting benchmarks in a WAN, we typically expect 20 min to run a single benchmark, and 60 min when using 3 repetitions.

Expected disk space for experiments: The log files typically are quite small, just a few hundred KBs to a few MBs. If you have configured ~8 GB available disk space you should not run into problems.

Note (optional):
If running inside a network simulator/emulator is envisioned, we advise to scale resource capabilities as needed to support the desired number of replicas on the host system. The time to conduct experiments will vary depending how many CPU cores are provided in total. Definetly use more RAM (> 128 GB) and more disk space (> 20 GB available), but in the end it depends on how many replicas you want to run (e.g. we calculate with ~4GB RAM per replica, but 8GB would be more safe).

### Software versions:

```
Java v. 11 (tested with openjdk version "11.0.28" 2025-07-15; build 11.0.28+6-post-Ubuntu-1ubuntu124.04.1)
Gradle v 7.1 (output of -version below)
------------------------------------------------------------
Gradle 7.1
------------------------------------------------------------
Build time:   2021-06-14 14:47:26 UTC
Revision:     989ccc9952b140ee6ab88870e8a12f1b2998369e
Kotlin:       1.4.31
Groovy:       3.0.7
Ant:          Apache Ant(TM) version 1.10.9 compiled on September 27 2020
JVM:          11.0.28 (Ubuntu 11.0.28+6-post-Ubuntu-1ubuntu124.04.1)
OS:           Linux 6.14.0-1012-aws amd64
------------------------------------------------------------
GCC: gcc (Ubuntu 13.3.0-6ubuntu2~24.04) 13.3.0
UNZIP: UnZip 6.00
GIT: git version 2.43.0
```

Note that these are the versions we tested with, newer version might also work but without guarantee.



## Step 1: Check for Updates (optional)

Check if the version you pulled from an online repo includes all recent changes form the most stable `main` branch from the [GitHub repository](https://github.com/bergerch/opti-aware). To check for updates it might be a good idea to  clone from the GitHub Repo we will maintain:


```
git clone https://github.com/bergerch/opti-aware.git
```


Now you should have the most recent changes. Next step is to build the software.

## Step 2: Install Dependencies / List of Dependencies

We use Java (v. 11) and Gradle (v 7.1), unzip and gcc for this project. If you have them already installed on your computer, then just skip this step otherwise you need to install them. For your convenience you may try the commands below:

### 2.1 Java
```
sudo apt install openjdk-11-jdk
```
### 2.2 Gradle
```
mkdir .gradle
cd .gradle
wget https\://services.gradle.org/distributions/gradle-7.1-bin.zip
unzip -d . gradle-7.1-bin.zip
```
### 2.3 GCC & UNZIP
```
sudo apt install gcc
```
```
sudo apt install unzip
```


## Step 3: Compile the Source Code

Change directory into this repository and execute the following commands:
For the first time, you can execute ``./gradlew`` (this should also install gradle) inside the repository.
```
./gradlew
```
Then, to compile all code, run the following commands (rebuild every time you make changes to ``src``) :
```
./gradlew installDist && ./gradlew build
```
This also prepares the required jar files and default configuration files to be available in the `build/install/library` directory.



## Step 4: Defining a System Configuration

To run any demonstration you first need to configure Opti-AWARE/BFT-SMaRt to define the protocol behavior and the location of each replica.

The servers must be specified in the configuration file (see `config/hosts.config`):

```
#server id, address and port (the ids from 0 to n-1 are the service replicas) 
0 127.0.0.1 11000 11001
1 127.0.0.1 11010 11011
2 127.0.0.1 11020 11021
3 127.0.0.1 11030 11031
4 127.0.0.1 11040 11041
5 127.0.0.1 11050 11051
6 127.0.0.1 11060 11061
```

**Important tip #1:** Always provide IP addresses instead of hostnames. If a machine running a replica is not correctly configured, BFT-SMaRt may fail to bind to the appropriate IP address and use the loopback address instead (127.0.0.1). This phenomenom may prevent clients and/or replicas from successfully establishing a connection among them.

**Important tip #2:** You may want to specify at least **n=4** replicas.


The system configurations also have to be specified (see `config/system.config`). A working configuration file is included but to understand the parameters we refer the interested reader to the paper.


## Step 5 (optional): Generating Public/Private Key Pairs

If you need to generate public/private keys for more replicas or clients, you can use the following command:

```
./runscripts/smartrun.sh bftsmart.tom.util.RSAKeyPairGenerator <id> <key size>
```

Keys are stored in the `config/keys` folder. The command above creates key pairs both for clients and replicas. 

**Important tip #3:** Alternatively, you can set the `system.communication.defaultkeys` to `true` in the `config/system.config` file to force all processes to use the same public/private keys pair and secret key. This is useful when deploying experiments and benchmarks, because it enables the programmer to avoid generating keys for all principals involved in the system. However, this must not be used in real deployments.


## Step 6: Deployment in a WAN

In this step you deploy the system in a WAN. You can launch several virtual machines in different regions. Note that every VM needs to have Java 11 installed to run the Java Bytecode. You will have to copy the build in
```
build/install/library
```
which also already includes configuration files and keys to every VM (both replicas and clients!).
For instance, we used AWS, 21 regions and `c5.xlarge` for each client and replica in every region. Regions are as outlined in:
```
data/aws/aws-servers.csv
```

Furthermore, note that firewall rules must be configured to allow TCP inbound and outbound traffic on the port range 11000 to 12000, or, the ports you defined yourself in Step 4.

**Important tip #4:** Never forget to delete the `config/currentView` file after you modify `config/hosts.config` or `config/system.config`. If `config/currentView` exists, BFT-SMaRt always fetches the group configuration from this file first. Otherwise, BFT-SMaRt fetches information from the other files and creates `config/currentView` from scratch. 



## Step 7: Running the Replicas


You can run a single instance of a *ThroughputLatencyServer* (a replica used to conduct benchmarks) using the following commands:

```
 cd build/install/library/
```
```
./smartrun.sh bftsmart.demo.microbenchmarks.ThroughputLatencyServer 0 10000 0 0 false nosig
```

Note that you passed the following parameters:

`ThroughputLatencyServer <processId> <measurement interval> <reply size> <state size> <context?> <nosig | default | ecdsa> [rwd | rw]`


**Important tip #5:** If you are getting timeout messages, it is possible that the application you are running takes too long to process the requests or the network delay is too high and PROPOSE messages from the leader does not arrive in time, so replicas may start the leader change protocol. To prevent that, try to increase the `system.totalordermulticast.timeout` parameter in `config/system.config`.


You need to repeat this procedure for all replicas on every VM, and increment the `<processId> ` for every replica. Make sure you use the correct `<processId>` as you defined with the `hosts.conf` in Step 4.


## Step 8: Running the Client(s)

**Important tip #6:** Clients requests should not be issued before all replicas have been properly initialized. Replicas are ready to process client requests when each one outputs `-- Ready to process operations` in the console.

Once all replicas are ready, the client can be launched as follows:

```
 cd build/install/library/
```
```
./smartrun.sh bftsmart.demo.microbenchmarks.ThroughputLatencyClient 1001 1 10000 0 0 false false nosig
```

`ThroughputLatencyClient <initial client id> <number of clients> <number of operations> <request size> <interval (ms)> <read only?> <verbose?> <nosig | default | ecdsa>`

**Important tip #7:** Always make sure that each client uses a unique ID. Otherwise, clients may not be able to complete their operations.

Note that these `bftsmart.benchmark` implementations should also automatically store the results to the file system.

**Important tip #8:** For parsing latency results from all clients from the different regions you may want to use a script after collecting the results. It may look similar to this:
```
cat bftSmartClient{0..20}/bftSmartClient*.java.*.stdout | grep "Average time for 1000 executions (-10%)" | sed 's/Average time for 1000 executions (-10%) = / /g' | sed 's/ \/\/  /, /g' | sed 's/us/ /g' > latencies.csv
```

## Step 9 (optional): Extract and Interpret Results

### ThroughputLatencyClient
The Client-side implementation can be used to measure the end-to-end latency.

#### What it measures
Client-side (end-to-end) latency per request: time from just before the client calls invoke{Ordered,Unordered} until the reply returns, measured in nanoseconds (System.nanoTime()).
Optional summary stats (if verbose=true) for the measured phase: average, standard deviation, and maximum latency; each printed in microseconds (ns/1000).

It can run in two modes:

```
read/write mode (readOnly=false) → invokeOrdered
```
#### What it prints to standard output:
Latency:
```
<clientId>\t<epochMillis>\t<latencyNanos>\n
```
where each line is the measured latency of a returned request-result cycle measured at the specific point in time at which the result was ready (at epochMillis)

After the benchmark completes it prints a summary to stdout:
```
<id> // Average time for <numOps/2> executions (-10%) = <avg_us> us
<id> // Standard desviation for <numOps/2> executions (-10%) = <sd_us> us
<id> // Average time for <numOps/2> executions (all samples) = <avg_us> us
<id> // Standard desviation for <numOps/2> executions (all samples) = <sd_us> us
<id> // Maximum time for <numOps/2> executions (all samples) = <max_us> us
```
The measurement unit is here microseconds.

#### What it writes to disk:

a tab seperated value file (similiar to a .csv but \t instead of comma as seperator), stored in the file system
```
latencies_<initialId>.txt
```
with the content of
```
<clientId>\t<epochMillis>\t<latencyNanos>\n
```
where each entry is the measured latency of a returned request-result cycle measured at the specific point in time at which the result was ready (at epochMillis)

#### Interpreting the numbers

Latency unit in the log: nanoseconds. Convert as needed:

microseconds = latencyNanos / 1_000

milliseconds = latencyNanos / 1_000_000

Averages/σ printed in verbose summary are in microseconds already.

## Step 10 (optional): Reproduce Results from the Paper

Evaluation results depend on the speed of communication links in the WAN. 
Interestingly, we observed that links may become faster (to some extent) over large time intervals (years) because large cloud providers like Amazon AWS improve their infrastructure. For this purpose we provide latency data that allow an interested person to mimic the network characteristics we used by relying on high-fidelity network emulation/simulation tools like [Kollaps](https://github.com/miguelammatos/Kollaps) and [Shadow](https://github.com/shadow/shadow). 
These latency data can be found in the directories:

For the AWS setup with 21 regions see:
```
data/aws/aws.csv
```

For the wonderproxy setup with 51 regions see:
```
data/wonderproxy/wonderproxy.csv
```

## Additional information and publications

If you are interested in learning more about BFT-SMaRt, you can read:

- The paper about its state machine protocol published in [EDCC'12](http://www.di.fc.ul.pt/~bessani/publications/edcc12-modsmart.pdf):
- The paper about its advanced state transfer protocol published in [Usenix'13](http://www.di.fc.ul.pt/~bessani/publications/usenix13-dsmr.pdf):
- The tool description published in [DSN'14](http://www.di.fc.ul.pt/~bessani/publications/dsn14-bftsmart.pdf):
- WHEAT is published in [SRDS'15](https://doi.org/10.1109/SRDS.2015.40)
- AWARE is published in [TDSC'20](https://doi.org/10.1109/TDSC.2020.3030605)

***Feel free to contact us if you have any questions!***


