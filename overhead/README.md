Overhead experiment is similar the experiment described in Figure 10, we need to change the code in consensus.go file to run different variations. 

Comment line 217 to run the overhead experiment with only latency vector enabled. 

Run the experiment with `proposalBytes` as one of the values to the metrics parameter.

Use `senddata.py` python script on the output data to know the overhead of the OptiLog on the proposal.