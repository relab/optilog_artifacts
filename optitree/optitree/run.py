import subprocess


def run():
    for bf in range(4, 15):
        for fault in range(1, 6):
            output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-faults", str(fault), "-csv", "latencies/wonderproxy.csv",
                                    "-iter", "10", "-timer", "5s"], text=True)
            print("Completed simulated annealing for branch factor "+str(bf)+", faults " + str(fault)+' '+"\n")

run()
