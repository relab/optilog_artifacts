package main

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"gonum.org/v1/gonum/stat"
)

func ComputeBaseTree(treeSize, root, bf int, l Latencies) []int {
	tree := make([]int, treeSize)
	tree[0] = root
	taken := make([]bool, treeSize)
	for i := range treeSize {
		taken[i] = false
	}
	taken[root] = true
	nearest := l.GetKNearest(root, bf, remaining(taken))
	for i := range bf {
		tree[i+1] = nearest[i]
		taken[nearest[i]] = true
	}
	for i := range bf {
		nearest := l.GetKNearest(tree[i+1], bf, remaining(taken))
		for j := range len(nearest) {
			taken[nearest[j]] = true
			index := (i+1)*bf + j + 1
			tree[index] = nearest[j]
		}
	}
	return tree
}

// GetKNearest returns the k nearest nodes to the given node.
// remainingNodes is modified in place.
func (l Latencies) GetKNearest(node int, k int, remainingNodes []int) []int {
	slices.SortFunc(remainingNodes, func(i, j int) int {
		return l.distance(node, i, j)
	})
	if len(remainingNodes) > k {
		return remainingNodes[:k]
	}
	return remainingNodes
}

func remaining(taken []bool) []int {
	remaining := make([]int, 0)
	for i, value := range taken {
		if !value {
			remaining = append(remaining, i)
		}
	}
	return remaining
}

func evenFaults(n int) int {
	return ((n - 1) / 3) &^ 1
}

func (l Latencies) SimulatedAnnealingWithFaults(params treeParams) result {
	tFaults := evenFaults(params.nNodes)
	for faults, j := 0, 0; faults < tFaults; faults += 2 {
		treeLatencies := make([]float64, 0, params.iterations)
		otherTreeLatencies := make([]float64, 0, params.iterations)
		var result result
		for range params.iterations {
			result = l.SimulatedAnnealing(params)
			treeLatencies = append(treeLatencies, float64(result.latency))
			newQuorum := params.scf + (j * params.scd)
			newTree := result.GeTree()
			otherTreeLatencies = append(otherTreeLatencies, float64(l.qcLatency(newQuorum, params.bf, newTree.AsNodes(), true)))
		}
		mean, st1 := stat.MeanStdDev(treeLatencies, nil)
		stat.MeanStdDev(otherTreeLatencies, nil)
		printf("reconfigurations %d, mean latency is %v, sd is %v\n", j, mean, st1)
		tree := result.GeTree()
		newBaseTree := append(tree[2:], tree[0], tree[1])
		params.faultIndex = params.nNodes - faults
		params.baseTree = newBaseTree
		params.faults = faults
		j += 1
	}
	return result{}
}

func (l Latencies) SimulatedAnnealing(params treeParams) result {
	timer := time.NewTimer(params.timeout)
	tree := TreeConfig(params.baseTree)
	nodes := tree.AsNodes()
	quorumSize := quorumSize(params.nNodes)
	if params.scf > 0 {
		quorumSize = params.scf
	}
	if params.faults > 0 {
		quorumSize += params.faults
	}
	best := result{
		latency: Latency(10000000),
		nodes:   nodes,
	}
	for params.temp > params.threshold {
		select {
		case <-timer.C:
			timer.Stop()
			return best
		default:
			newSolution := mutate(tree, params.faultIndex)
			nodes = newSolution.AsNodes()
			latency := l.qcLatency(quorumSize, params.bf, nodes, (params.faultIndex > 0))
			if latency < best.latency {
				tree = newSolution
				best.latency = latency
				copy(best.nodes, nodes)
				//copy(best.tree, tree)
				// printf("new tree %v\n", nodes)
				// printf("quorum size %d, bf %d latency %v\n", quorumSize, params.bf, latency)
			} else {
				random := rand.Float64()
				if math.Exp(-(float64(latency-best.latency) / params.temp)) > random {
					tree = newSolution
					best.latency = latency
					copy(best.nodes, nodes)
					//copy(best.tree, tree)
					// printf("new tree %v\n", nodes)
				}
			}
			// Cool system down
			params.temp *= 1 - params.coolingRate
			best.analyzedTrees++
		}
	}
	return best
}

type TreeConfig []int

// IntN returns a random index in the range [1, len(tc)].
func (tc TreeConfig) IntN() int {
	return rand.IntN(len(tc))
}

func (tc TreeConfig) AsNodes() []node {
	nodes := make([]node, len(tc))
	for i, id := range tc {
		nodes[i] = node{id: id, votes: 1}
	}
	return nodes
}

// mutate the tree by swapping two nodes, such that the two nodes are swapped within the same
// group of nodes (above or below faultIdx). The faultIdx is the index of the first faulty node.
func mutate(tree TreeConfig, faultIdx int) TreeConfig {
	idx1, idx2 := tree.IntN(), tree.IntN()
	// If the two nodes (idx1 and idx2) aren't in the same group (above or below faultIdx), we try again.
	for idx1 == idx2 || ((idx1 >= faultIdx) != (idx2 >= faultIdx)) {
		idx2 = tree.IntN()
		idx1 = tree.IntN()
	}
	// swap the two nodes in the same group.
	newTree := slices.Clone(tree)
	newTree[idx1], newTree[idx2] = newTree[idx2], newTree[idx1]
	return newTree
}

func (l Latencies) SimulatedAnnealingPerformance(params treeParams) result {
	results := make([]float64, params.iterations)
	for i := range params.iterations {
		results[i] = float64(l.ParallelSimulatedAnnealing(params).latency)
	}
	mean, stdDev := stat.MeanStdDev(results, nil)
	return result{mean: mean, stdDev: stdDev}
}

func (l Latencies) ParallelSimulatedAnnealing(params treeParams) result {
	treeSize := params.nNodes
	saParams := simulatedAnnealingParams{
		temp:        params.temp,
		coolingRate: params.coolingRate,
		threshold:   params.threshold,
		timeout:     params.timeout,
	}
	results := make(chan result, treeSize)
	for root := range treeSize {
		go func() {
			baseTree := ComputeBaseTree(treeSize, root, params.bf, l)
			params := NewTreeParams(baseTree, params.bf, 100, params.faults, 0)
			params.SetSimulatedAnnealingParams(saParams)
			results <- l.SimulatedAnnealing(params)
		}()
	}
	optimal := result{latency: Latency(10000000)}
	for range treeSize {
		r := <-results
		if r.latency < optimal.latency {
			optimal.latency = r.latency
			optimal.nodes = r.nodes
		}
		optimal.analyzedTrees += r.analyzedTrees
	}
	return optimal
}
