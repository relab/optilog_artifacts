package main

import (
	"testing"
	"time"
)

func TestComputeBaseTree(t *testing.T) {
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		t.Fatal(err)
	}
	tree := ComputeBaseTree(21, 0, 4, latencies)
	t.Logf("Tree: %v", tree)
	params := NewTreeParams(tree, 4, 100, 0)
	params.timeout = 2 * time.Second
	pTree := latencies.ParallelSimulatedAnnealing(params)
	t.Logf("PTree: %v", pTree)
}
