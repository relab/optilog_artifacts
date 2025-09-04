package main

import (
	"fmt"
	"time"
)

// QCOptimalTreeChannel finds the tree with the lowest latency
// to collect votes from a quorum of nodes.
func (l Latencies) QCOptimalTreeChannel(params treeParams) result {
	treeSize := len(params.baseTree)
	qs := quorumSize(treeSize)
	results := make(chan result, treeSize)
	for _, root := range params.baseTree {
		go func() {
			// Find the best latency for this root
			best := result{
				latency: Latency(10000000),
				nodes:   make([]node, treeSize),
			}
			tree := newSubtree(root, params.baseTree)
			nodes := newNodes(root, tree)
			now := time.Now()

			UniqueTrees(tree, params.bf, func(tree []int) {
				resetNodes(nodes, tree)
				latency := l.qcLatency(qs, params.bf, nodes)
				if latency < best.latency {
					best.latency = latency
					copy(best.nodes, nodes)
					// printf("%2d: Tree %v has best latency %s\n", root, toTree(best.nodes), best.latency)
				}
				best.analyzedTrees++
				if params.logNow(best.analyzedTrees, root, now) {
					now = time.Now()
				}
			})
			results <- best
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

type result struct {
	nodes         []node
	latency       Latency
	analyzedTrees int64
	mean          float64
	stdDev        float64
}

func (r result) String() string {
	return fmt.Sprintf("tree: %v has latency: %s", toString(r.nodes), r.latency)
}
