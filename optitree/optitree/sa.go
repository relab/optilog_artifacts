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

func (l Latencies) SimulatedAnnealing(params treeParams) result {
	timer := time.NewTimer(params.timeout)
	tree := TreeConfig(params.baseTree)
	nodes := tree.AsNodes()
	quorumSize := quorumSize(params.nNodes)
	if params.faults > 0 {
		quorumSize += params.faults
	}
	best := result{
		latency: l.qcLatency(quorumSize, params.bf, nodes),
		nodes:   nodes,
	}
	for params.temp > params.threshold {
		select {
		case <-timer.C:
			timer.Stop()
			return best
		default:
			newSolution := mutate(tree)
			nodes = newSolution.AsNodes()
			latency := l.qcLatency(quorumSize, params.bf, nodes)
			if latency < best.latency {
				tree = newSolution
				best.latency = latency
				copy(best.nodes, nodes)
			} else {
				random := rand.Float64()
				if math.Exp(-(float64(latency-best.latency) / params.temp)) > random {
					tree = newSolution
					best.latency = latency
					copy(best.nodes, nodes)
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

func mutate(tree TreeConfig) TreeConfig {
	idx1, idx2 := tree.IntN(), tree.IntN()
	for idx1 == idx2 {
		idx2 = tree.IntN()
	}
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
	if params.timeout == 0 {
		params.timeout = time.Second
	}
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
			params := NewTreeParams(baseTree, params.bf, 100, params.faults)
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
