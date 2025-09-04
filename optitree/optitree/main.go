package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/pkg/profile"
)

const (
	awsLatencyFile         = "latencies/aws.csv"
	wonderproxyLatencyFile = "latencies/wonderproxy.csv"
)

func main() {
	var (
		mode = flag.String("profile", "", "enable profiling mode, one of [cpu, mem, mutex, block, trace]")
		opt  = flag.String("opt", "channel", "optimization algorithm to run, one of [channel, mutex, sa]")
		bf   = flag.Int("bf", 3, "branch factor of the tree")
		sz   = flag.Int("size", 0, "size of the tree, if zero, bf is used to compute the tree size")
		tree = flag.String("tree", "", "starting tree [0,1,2,3,4,5,6]")
		emit = flag.Int("emit", 0, "progress emit cadence (0 for no progress output)")
		csv  = flag.String("csv", awsLatencyFile, "use latencies from csv file")

		cities = flag.String("cities", "random", "comma separated list of cities to use for latencies")
		iter   = flag.Int("iter", 1, "number of iterations for simulated annealing (for performance evaluation)")
		timer  = flag.Duration("timer", 1*time.Second, "timeout for simulated annealing")
		cool   = flag.Float64("cool", 0.00055, "cooling rate, if set to zero simulated annealing runs until timer expires")

		faultAnalysis = flag.String("analysis", "", "latency fault analysis, one of [optitree, kauri, kauri-sa]")
		scf           = flag.Int("scf", 0, "scoring function value")
		scd           = flag.Int("scd", 0, "scoring function value delta")
		faults        = flag.Int("faults", 0, "number of faulty nodes in the tree")
	)
	flag.Parse()

	const profilePath = "."
	switch *mode {
	case "cpu":
		defer profile.Start(profile.ProfilePath(profilePath), profile.CPUProfile).Stop()
	case "mem":
		defer profile.Start(profile.ProfilePath(profilePath), profile.MemProfile).Stop()
	case "mutex":
		defer profile.Start(profile.ProfilePath(profilePath), profile.MutexProfile).Stop()
	case "block":
		defer profile.Start(profile.ProfilePath(profilePath), profile.BlockProfile).Stop()
	case "trace":
		defer profile.Start(profile.ProfilePath(profilePath), profile.TraceProfile).Stop()
	default:
		// don't profile
	}

	size := TreeSize(*bf)
	if *sz > 0 {
		size = *sz
	}
	var latencies Latencies
	if *csv != "" {
		fmt.Println("Loading latencies from:", *csv)
		var err error
		latencies, err = loadLatencies(*csv, *cities)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		latencies = NewRand(size)
	}

	var startTree []int
	if *tree != "" {
		t, err := parseTreeString(*tree)
		if err != nil {
			log.Fatal(err)
		}
		startTree = t
	} else {
		startTree = basicTree(size)
	}
	if len(startTree) != size {
		log.Fatalf("Invalid tree size: %d, expected: %d", len(startTree), size)
	}

	if len(startTree) < *faults+quorumSize(len(startTree)) {
		log.Fatalf("Invalid number of faults: %d, should be less than %d", *faults, len(startTree)-quorumSize(len(startTree)))
	}
	params := NewTreeParams(startTree, *bf, *emit, *faults, *scd)
	fmt.Printf("Number of CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("Number of trees expected: %d\n", params.nTrees)
	fmt.Printf("Starting tree (size=%d, bf=%d): %v\n", size, *bf, startTree)
	fmt.Printf("Will emit every %d trees for a total of %d emit events\n", params.cadence, params.emitEvents())

	var optimize func(params treeParams) result
	switch *opt {
	case "channel":
		optimize = latencies.QCOptimalTreeChannel
	case "mutex":
		optimize = latencies.QCOptimalTreeMutex
	case "sa":
		if *scf == 0 {
			*scf = quorumSize(len(startTree))
		}
		params.SetSimulatedAnnealingParams(simulatedAnnealingParams{
			temp:        25000.0,
			coolingRate: *cool,
			threshold:   0.5,
			timeout:     *timer,
			scf:         *scf,
		})
		params.iterations = *iter
		switch *faultAnalysis {
		case "optitree":
			optimize = latencies.SimulatedAnnealingWithFaults
		case "kauri":
			optimize = latencies.KauriFaultLatency
		case "kauri-sa":
			optimize = latencies.KauriSALatency
		default:
			if *iter > 1 {
				fmt.Printf("Running simulated annealing with %d iterations, timer duration: %s\n", *iter, *timer)
				optimize = latencies.SimulatedAnnealingPerformance
			} else {
				fmt.Print("Running single shot simulated annealing\n")
				optimize = latencies.ParallelSimulatedAnnealing
			}
		}
	default:
		log.Fatalf("Invalid optimization algorithm: %s", *opt)
	}

	now := time.Now()
	optimal := optimize(params)
	stop := time.Since(now)
	if optimal.mean > 0 {
		fmt.Printf("\nSimulated annealing performance: mean: %f, std dev: %f\n", optimal.mean, optimal.stdDev)
	} else {
		fmt.Printf("Total unique trees analyzed: %d\n\n", optimal.analyzedTrees)
		fmt.Println("Optimal tree found after:", stop)
		// fmt.Printf("\nThe QC optimal %s\n", optimal)
		// latencies.PrintNodes(optimal.nodes, *bf)
	}
}

// TODO clean up the cli flags: sa vs brute force, fault analysis, etc.
// TODO Clean up the run.py script with the new cli flags and make it replicated the paper results
// TODO make the output go to csv files
// TODO make the output from the default SA analysis generate a toml file for ingestion into the hotstuff framework

// TODO run with pgo
// TODO make separate function that does not use goroutines at all; and we can run them in parallel using SLURM.
// TODO integrate with iago to run on bbchain and unix gorina nodes
// TODO add random tree generation and cost calculation
// TODO cost of a tree when removing all faulty nodes
