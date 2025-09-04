package main

import "fmt"

// OptimalTree finds the optimal tree for the given tree size and branch factor.
// This includes all latencies between the nodes in the tree.
func (l Latencies) OptimalTree(treeSize, branchFactor int) {
	type result struct {
		tree           []int
		latency        Latency
		numUniqueTrees int
	}
	results := make(chan result, treeSize)
	for root := 0; root < treeSize; root++ {
		go func() {
			bestLatency := Latency(10000000)
			var bestTree []int
			tree := generateBaseTree(treeSize, root)
			numUniqueTrees := 0
			UniqueTrees(tree, branchFactor, func(tree []int) {
				latency := l.treeLatency(root, branchFactor, tree)
				if latency < bestLatency {
					bestLatency = latency
					bestTree = append([]int{root}, tree...)
					// best latency so far for this root
					fmt.Printf("Tree %v has latency %d\n", bestTree, bestLatency)
				}
				numUniqueTrees++
				if numUniqueTrees%100_000_000 == 0 {
					tmpTree := append([]int{root}, tree...)
					fmt.Printf("Tree[%d]: %v\n", numUniqueTrees, tmpTree)
				}
			})
			results <- result{bestTree, bestLatency, numUniqueTrees}
		}()
	}
	optimalLatency := Latency(10000000)
	var optimalTree []int
	totalTrees := 0
	for range treeSize {
		r := <-results
		if r.latency < optimalLatency {
			optimalLatency = r.latency
			optimalTree = r.tree
		}
		totalTrees += r.numUniqueTrees
	}
	fmt.Printf("Optimal tree %v has latency: %d\n", optimalTree, optimalLatency)
	fmt.Printf("Total unique trees: %d\n", totalTrees)
	l.Print(optimalTree, branchFactor)
}
