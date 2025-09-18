### OptiLog Overhead Experiments

This directory contains the necessary artifacts to conduct the OptiLog overhead experiments as presented in Figure 13.
The experiments are organized into four variants, each enabling a specific set of OptiLog sensors.
We used Ubuntu 22.04 as the operating system on the nodes in the cluster.

#### Prerequisites

Before proceeding, ensure that you have completed all setup steps outlined in [`fig10_config/README.md`](../fig10_config/README.md).
This includes installing dependencies, configuring your environment, and preparing the required input files.
Update the TOML configuration files by replacing occurrences of `bbchain` with the actual node hostnames in your cluster.

#### Building the HotStuff Binary

On the controller node, compile the HotStuff binary using the following commands:
```sh
cd hotstuff
make install
make
```

#### Running the Experiment

To execute the experiment described in Figure 13, modify the `--config` parameter to match your cluster size.

**Example command for a 20-node cluster:**
```sh
./hotstuff run --config 20.toml --ssh-config ssh_config --leader-rotation fixed --client-timeout 150s --duration 120s --metrics proposalBytes --measurement-interval 1s --output output_data --max-concurrent 3000 --view-timeout 1s --modules ranking
```

#### Processing Experiment Results

Upon completion, experiment results will be available in the `output_data` directory.
Use the `senddata.py` script on this output to compute the average proposal size.

To run this script, install python3.
```sh
sudo apt install python3
```

To generate the results from the experiment output:

```sh
python3 senddata.py output_data
```

This script generates the average proposal size in the experiment. 
Repeat the experiment for other configurations.
Each experiment consumes 1.5MB of disk space on the controller node and negligible disk space is taken on other nodes in the cluster at `/tmp/`.
Sample data from the experiments is [here](overhead_data.tgz).
