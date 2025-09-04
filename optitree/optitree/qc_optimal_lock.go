package main

import (
	"sync"
)

// QCOptimalTreeMutex finds the tree with the lowest latency
// to collect votes from a quorum of nodes.
func (l Latencies) QCOptimalTreeMutex(params treeParams) result {
	treeSize := len(params.baseTree)
	qs := quorumSize(treeSize)
	var mutex sync.Mutex
	optimal := result{
		latency: Latency(10000000),
		nodes:   make([]node, treeSize),
	}

	var wg sync.WaitGroup
	for _, root := range params.baseTree {
		wg.Add(1)
		go func(root int) {
			defer wg.Done()
			bestLatency := Latency(10000000)
			tree := newSubtree(root, params.baseTree)
			nodes := newNodes(root, tree)

			UniqueTrees(tree, params.bf, func(tree []int) {
				resetNodes(nodes, tree)
				latency := l.qcLatency(qs, params.bf, nodes)
				if latency < bestLatency {
					mutex.Lock()
					optLat := optimal.latency
					if latency < optLat {
						optimal.latency = latency
						copy(optimal.nodes, nodes)
						mutex.Unlock()

						bestLatency = latency
						printf("%2d: Tree %v has best latency %s\n", root, toTree(nodes), bestLatency)
					} else {
						mutex.Unlock()
						bestLatency = optLat
					}
				}
			})
		}(root)
	}
	wg.Wait()
	return optimal
}
