### OptiTree Simulation Experiments

Optitree simulation experiments are described in Figure 9 and Figure 12. 
These experiments are run on the a laptop with 16 GB RAM and 12 cores running Ubuntu 22.04 operating system.

#### Build OptiTree binary

Run the following command to build optitree binary

```sh
go mod download
cd optitree; go build .
```
#### Running Experiments

To conduct the OptiTree simulation experiments, follow these steps:

1. **Install Python 3**

    Ensure Python 3 is installed on your system:

    ```sh
    sudo apt install python3
    ```

2. **Run Reconfiguration Simulations (Figure 9)**

    Execute the following command in the `optitree` directory to run the reconfiguration simulations:

    ```sh
    python3 fig9_run.py
    ```

3. **Run Simulated Annealing Simulations (Figure 12)**

    Execute the following command in the `optitree` directory to run the simulated annealing simulations:

    ```sh
    python3 fig12_run.py
    ```

4. **Plot Results**

    Use the latency values from the script outputs to generate the corresponding graphs.
