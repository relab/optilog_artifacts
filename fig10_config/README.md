### HotStuff Framework Experiments

The experiments described in Figure 10 use the [HotStuff repository](https://github.com/relab/hotstuff). These experiments are designed to run on a cluster, with each node hosting one or more replicas.
We used Ubuntu 22.04 as the operating system on the nodes in the cluster.
These experiments utilize four distinct network configurations, with each configuration evaluated under five different protocol settings. For each experiment, the average throughput and latency are measured, and results are reported along with the 95% confidence interval.

#### Prerequisites

1. **Cluster Setup:**  
    Choose one node in your cluster to serve as the orchestrator/controller.

2. **Golang Installation:**  
    On the controller node, install Go version 1.23.4 by following the instructions at [https://go.dev/dl/](https://go.dev/dl/).

3. **Passwordless SSH Access:**  
    Set up passwordless SSH from the controller node to all other nodes. Replace `node_name` with the hostname of each target node:
    ```sh
    ssh -o UpdateHostKeys=yes -o PreferredAuthentications=publickey -o StrictHostKeyChecking=no node_name echo "hello node_name"
    ```
4. **Configure the SSH Configuration File:**  
    A sample SSH configuration file (`ssh_config`) is provided. Review and modify this file to accurately reflect the hostnames, user credentials, and connection parameters specific to your cluster environment.

5. **Prepare Configuration Files:**  
    Edit the sample configuration files (`config_21.cue`, `config_43.cue`, and `config_73.cue`) by replacing all instances of `bbchain` with the actual hostnames of the cluster nodes.

#### Building the HotStuff Binary

On the controller node, build the HotStuff binary with:
```sh
cd hotstuff
make install
make
```

#### Running the Experiment

To run the experiment described in Figure 10, adjust the `--cue` parameter based on the desired cluster size.
Remove the `--modules` parameter if running the standard HotStuff protocol. For HotStuff with round-robin leader rotation, use `--leader-rotation round-robin`.

**Example command for running the experiment with 73 nodes:**

```sh
./hotstuff run --cue config_73.cue --ssh-config ssh_config --leader-rotation tree-leader --tree-delta 1ms --client-timeout 150s --duration 120s --metrics throughput,consensus-latency,latency-vector --measurement-interval 1s --output output_data --max-concurrent 3000 --view-timeout 1s --modules kauri
```

#### Processing Experiment Results

After the experiment completes, results are stored in the `output_data` directory.
Use the `throughput_avg.py` and `latency_avg.py` scripts on the output data to compute throughput and latency metrics for the experiment.

Install the following dependencies

```sh
sudo apt install python3
sudo apt install python3-pip    
pip3 install python-dateutil
```

Run script to generate the results, (first 5 seconds of experiment data is not considered)

```sh
python3 throughput_avg.py output_data 5
python3 latency_avg.py output_data 5
```

These scripts calculate the average throughput and latency with 90% confidence interval bounds (upper and lower). 

To generate complete results as shown in Figure 10:
1. Execute this experiment across all network configurations
2. Test each configuration with all five protocol variants
3. Use pgfplots to visualize the collected data

Each experiment is run for 120 seconds and measurements for first 5 seconds are discarded.
This experiment consumes 1.5MB of disk space on the controller node and on other nodes in the cluster negligible disk space is taken at `/tmp/`.
Sample datasets from these experiments are available for download [here](fig_10_data.tgz).

