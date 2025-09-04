package main

import (
	"math/rand"

	"gonum.org/v1/gonum/stat"
)

func changeCluster(baseTree []int, cluster, bf int) TreeConfig {
	if cluster == 0 {
		return baseTree
	}
	startIndex := cluster * (bf + 1)
	return append(baseTree[startIndex:], baseTree[:startIndex]...)
}

func (l Latencies) KauriFaultLatency(params treeParams) result {
	baseTree := params.baseTree
	rand.Shuffle(len(baseTree), func(i, j int) {
		baseTree[i], baseTree[j] = baseTree[j], baseTree[i]
	})
	clusters := len(baseTree) / (params.bf + 1)
	for i := 0; i < clusters; i++ {
		latencies := make([]float64, 0, params.iterations)
		otherTreeLatencies := make([]float64, 0, params.iterations)
		for range params.iterations {
			newTree := changeCluster(baseTree, i, params.bf)
			treeLatency := l.qcLatency(params.scf, params.bf, newTree.AsNodes(), true)
			otherTreeLatency := l.qcLatency(params.scf+params.scd, params.bf, newTree.AsNodes(), true)
			latencies = append(latencies, float64(treeLatency))
			otherTreeLatencies = append(otherTreeLatencies, float64(otherTreeLatency))
		}
		mean, st1 := stat.MeanStdDev(latencies, nil)
		stat.MeanStdDev(otherTreeLatencies, nil)
		printf(" reconfigurations %d, mean latency %v, standard deviation %v\n", i+1, mean, st1)
	}
	return result{}
}

func (l Latencies) KauriSALatency(params treeParams) result {

	baseTree := make([]int, params.nNodes)
	copy(baseTree, params.baseTree)
	rand.Shuffle(len(baseTree), func(i, j int) {
		baseTree[i], baseTree[j] = baseTree[j], baseTree[i]
	})
	clusters := len(baseTree) / (params.bf + 1)
	clusterSize := params.bf + 1
	for i := 0; i < clusters; i++ {
		latencies := make([]float64, 0, params.iterations)
		baseTree := changeCluster(baseTree, i, params.bf)
		params.faultIndex = params.nNodes - (i * clusterSize)
		copy(params.baseTree, baseTree)
		params.faults = i * clusterSize
		for range params.iterations {
			res := l.SimulatedAnnealing(params)
			latencies = append(latencies, float64(res.latency))
		}
		mean, st1 := stat.MeanStdDev(latencies, nil)
		printf("reconfigurations %d mean latency  %v standard deviation %v \n", i+1, mean, st1)
	}
	return result{}
}
