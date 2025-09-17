### OptiLog Tolerance Experiments

This directory contains the artifacts to conduct the experiements described in Figure 11. 
These experiments measure the effect of faulty replicas on the configuration latency on a 21 node configuration.
We conducted experiments under 1 to 4 faults in the configuration with varying tolerance limit from 1.1 to 1.4. 


#### Prerequisites

Before proceeding, ensure that you have completed all setup steps outlined in [`fig10_config/README.md`](../fig10_config/README.md).
This includes installing dependencies, configuring your environment, and preparing the required input files.
Use config_21.cue to run this set of experiments. 


#### Building the HotStuff Binary

1. To introduce faulty nodes in the experiment, modify the `FaultyNodes` variable in `hotstuff/hotstuff.go`. For the 21-node configuration, the intermediate nodes `[5, 6, 7, 1]` should be included in `FaultyNodes` sequentially to conduct experiments with 1 to 4 faulty nodes.

2. Adjust the `WriggleRoomValue` parameter in the configuration file to values ranging from 1.1 to 1.4, according to the requirements of each experimental run.

2. On the controller node, compile the HotStuff binary using the following commands:
```sh
cd hotstuff
make install
make
```

#### Running the Experiment

To execute the experiment described in Figure 11, change the values in the files mentioned above and recompile to build the new binary.
We used Ubuntu 22.04 as the operating system on the nodes in the Cluster.

**Example command for a 21-node cluster:**

```sh
./hotstuff run --cue config_21.cue --ssh-config ssh_config --leader-rotation tree-leader --tree-delta 1ms --client-timeout 150s --duration 120s --metrics throughput,consensus-latency --measurement-interval 1s --output output_data --max-concurrent 3000 --view-timeout 1s --modules kauri
```

#### Processing Experiment Results

Upon completion, experiment results will be available in the `output_data` directory. Use the `throughput_avg.py` and `latency_avg.py` scripts on the output data to compute throughput and latency metrics for the experiment.

To run this script install the following dependencies

```sh
sudo apt install python3
sudo apt install python3-pip    
pip3 install python-dateutil
```


Run script to generate the results, (first 5 secs of experiment data is not considered)

```sh
python3 throughput_avg.py output_data 5
python3 latency_avg.py output_data 5
```

These scripts calculate the average throughput and latency with 90% confidence interval bounds (upper and lower). 

To generate complete results as shown in Figure 11:
1. Execute this experiment with different wriggle room values.
2. Use pgfplots to visualize the collected data

Sample datasets from these experiments are available for download [here](wriggle_data.tgz).
