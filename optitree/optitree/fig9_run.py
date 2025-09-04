import subprocess
import math

iterations = 10

def config_size(bf):
    return int(math.pow(bf,2)+bf+1)

def max_faults(bf):
    size = config_size(bf)
    return int(size/3)

def run():
    for bf in range(12, 15):
        output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                                 "-analysis", "optitree", "-iter", str(iterations), "-scd", "2"], text=True)
        print("Completed fault analysis for branch factor "+str(bf)+", duration \n")

def over_provision():
    for bf in range(4, 15):
        size = config_size(bf)
        total_faults = max_faults(bf)
        for faults in range(1, total_faults):
            output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                                "-iter", str(iterations)], text=True)
            print("\nCompleted over provision analysis for branch factor "+str(bf)+" faults "+str(faults)+", duration 1s \n")

def optitree_reconfigurations(bf=14):
    #optitree reconfigurations
    bf = 14
    output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                               "-analysis", "optitree","-iter", str(iterations), "-scd", "2", "-scf", "141"], text=True)
    print("\nCompleted Optitree reconfiguration analysis for branch factor "+str(bf)+", duration 1s \n")

def kauri_sa_reconfigurations(bf=14):
    #Kauri-sa reconfigurations
    output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                               "-analysis", "kauri-sa","-iter", str(iterations)], text=True)
    print("\nCompleted kauri sa reconfiguration analysis for branch factor "+str(bf)+", duration 1s \n")

def kauri_reconfigurations(bf=14):
    #Kauri-sa reconfigurations
    output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                               "-analysis", "kauri","-iter", str(iterations)], text=True)
    print("\nCompleted kauri sa reconfiguration analysis for branch factor "+str(bf)+", duration 1s \n")

def reconfigurations():
    optitree_reconfigurations()
    kauri_sa_reconfigurations()
    kauri_reconfigurations()

def sa_timer():
    for bf in range(4, 15):
        duration = 0.25
        while duration <= 4:
            output = subprocess.run(["./optitree", "-opt", "sa", "-bf", str(bf), "-csv", "latencies/wonderproxy.csv",
                                "-iter", str(iterations), "-timer", str(duration)+"s"], text=True)
            print("\nCompleted SA analysis for branch factor "+str(bf)+", duration "+str(duration)+" s \n")
            duration *= 2
sa_timer()