To build the HotStuff binary, run

```sh
make install 
make
```

Use the following command to setup password-less access from the controller node to all other nodes.

```sh
ssh -o UpdateHostKeys=yes -o PreferredAuthentications=publickey -o StrictHostKeyChecking=no bbchain$i echo "hello bbchain$i"
```

Command to run experiment(fig 10), change the cue parameter based on the size, remove modules parameter if running hotstuff, use round-robin for leader-rotation parameter for hotstuff rr.

An example command to run the experiement with 73 nodes

```sh
./hotstuff run --cue config_73.cue --ssh-config ssh_config --leader-rotation tree-leader --tree-delta 1ms --client-timeout 150s --duration 120s --metrics throughput,consensus-latency,latency-vector --measurement-interval 1s --output output_data --max-concurrent 3000  --view-timeout 1s  --modules kauri  
```

After completion of the experiment, the results are stored in output_data directory, use throughput_avg.py and latency_avg.py scripts on the output data to generate the throughput and latency for the experiment.
