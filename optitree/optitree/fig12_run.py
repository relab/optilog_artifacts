import subprocess

timers = ["250ms", "500ms", "1s", "2s", "4s"]
def run():
    for bf in range(4, 15):
        for timer in timers:
            output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                                    "-iter", "10", "-timer", timer], text=True)
        print("Completed simulated annealing for"+timer+" with branch factor "+str(bf)+' '+"\n")

run()
