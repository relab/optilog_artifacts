### HotStuff Framework Experiments

The experiments described in Figure 10 use the [HotStuff repository](https://github.com/relab/hotstuff). These experiments are designed to run on a cluster, with each node hosting one or more replicas.
We used Ubuntu 22.04 as the operating system on the nodes in the Cluster.

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

#### Building the HotStuff Binary

On the controller node, build the HotStuff binary with:
```sh
make install
make
```

#### Running the Experiment


To run the experiment described in Figure 10, adjust the `--cue` parameter based on the desired cluster size. Remove the `--modules` parameter if running the standard HotStuff protocol. For HotStuff with round-robin leader rotation, use `--leader-rotation round-robin`.

**Example command for running the experiment with 73 nodes:**

```sh
./hotstuff run --cue config_73.cue --ssh-config ssh_config --leader-rotation tree-leader --tree-delta 1ms --client-timeout 150s --duration 120s --metrics throughput,consensus-latency,latency-vector --measurement-interval 1s --output output_data --max-concurrent 3000 --view-timeout 1s --modules kauri
```

#### Processing Experiment Results

After the experiment completes, results are stored in the `output_data` directory. Use the `throughput_avg.py` and `latency_avg.py` scripts on the output data to compute throughput and latency metrics for the experiment.
