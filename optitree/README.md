### OptiTree Simulation Experiments

Optitree simulation experiments are described in Figure 9 and Figure 12. 
These experiments are run on the a laptop with 16 GB RAM and 12 cores running Ubuntu 22.04 operating system.

#### Build OptiTree binary

Run the following command to build optitree binary

```sh
go mod download
cd optitree; go build .
```

#### Run Experiments 

1. Install Python3 to run the experiments

```sh
 sudo apt install python3
```
2. Use `fig9_run.py` in optitree directory to run reconfiguration simulations

```sh
python3 fig9_run.py
```

3. Use `fig12_run.py` in optitree directory to run simulated annealhing simulations

```sh
python3 fig12_run.py
```

4. Use the latency from the script output to plot the graphs
