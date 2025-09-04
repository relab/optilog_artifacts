package main

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestComputeBaseTree(t *testing.T) {
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		t.Fatal(err)
	}
	tree := ComputeBaseTree(21, 0, 4, latencies)
	t.Logf("Tree: %v", tree)
	params := NewTreeParams(tree, 4, 100, 0, 0)
	params.timeout = 2 * time.Second
	pTree := latencies.ParallelSimulatedAnnealing(params)
	t.Logf("PTree: %v", pTree)
}

func TestAsNodes(t *testing.T) {
	treeConfig := TreeConfig{1, 4, 5, 6, 7, 2, 3}
	nodes := treeConfig.AsNodes()
	if nodes[6].id != 3 {
		t.Error("expected 6")
	}
}

func TestGetTree(t *testing.T) {
	res := result{nodes: []node{{id: 1}, {id: 2}, {id: 3}, {id: 4}, {id: 5}, {id: 6}, {id: 7}}}
	tree := res.GeTree()
	if tree[0] != 1 {
		t.Error("expected 1")
	}
}

func TestFMutate(t *testing.T) {
	tree := TreeConfig{1, 4, 5, 6, 7, 2, 3}
	for i := 0; i < 100; i++ {
		tree = mutate(tree, 5)
	}
	newTree := mutate(tree, 5)
	if newTree[len(newTree)-1] != 2 && newTree[len(newTree)-1] != 3 {
		t.Error("expected last nodes to be 2 or 3 ")
	}
	if newTree[len(newTree)-2] != 2 && newTree[len(newTree)-2] != 3 {
		t.Error("expected last nodes to be 2 or 3 ")
	}
	tree = make([]int, 100)
	for i := 0; i < 90; i++ {
		tree[i] = i + 100
	}
	faulty := make(map[int]bool)
	for i := 0; i < 10; i++ {
		tree[i+90] = i
		faulty[i] = true
	}

	for i := 0; i < 100; i++ {
		tree = mutate(tree, 90)
	}
	newTree = mutate(tree, 90)
	for i := 90; i < 100; i++ {
		if newTree[i] > 10 {
			t.Error("last elements should be faulty and less than 10")
		}
	}
}

func TestMutate(t *testing.T) {
	tests := []struct {
		bf int
	}{
		{bf: 3},
		{bf: 4},
	}

	for _, tt := range tests {
		sz := TreeSize(tt.bf)
		for faultIdx := range sz + 1 {
			t.Run(fmt.Sprintf("bf=%d/size=%v/idx=%d", tt.bf, sz, faultIdx), func(t *testing.T) {
				tree := TreeConfig(basicTree(sz))
				a, b := tree[:faultIdx], tree[faultIdx:]
				// t.Logf("Before: %v, %v", a, b)
				mutatedTree := mutate(tree, faultIdx)
				p, q := mutatedTree[:faultIdx], mutatedTree[faultIdx:]
				// t.Logf("After : %v, %v", p, q)

				topEqual, botEqual := cmp.Equal(a, p), cmp.Equal(b, q)
				switch {
				case topEqual && botEqual:
					t.Error("expected different tree after mutation")
				case !topEqual && !botEqual:
					t.Error("expected mutation only in top or bottom part of tree, not both")
				}
				if !topEqual {
					slices.Sort(p)
					if diff := cmp.Diff(a, p); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				}
				if !botEqual {
					slices.Sort(q)
					if diff := cmp.Diff(b, q); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				}
			})
		}
	}
}

func TestChangeCluster(t *testing.T) {
	baseTree := TreeConfig{1, 4, 5, 6, 7, 2, 3}
	newTree := changeCluster(baseTree, 0, 2)
	if newTree[0] != 1 {
		t.Error("expected 1")
	}
	newTree = changeCluster(baseTree, 1, 2)
	if newTree[0] != 6 {
		t.Error("expected 6")
	}
	newTree = changeCluster(baseTree, 2, 2)
	if newTree[0] != 3 {
		t.Error("expected 3")
	}
}

func TestNumFaults(t *testing.T) {
	for n := range 101 {
		f := evenFaults(n)
		if f%2 != 0 {
			t.Error("expected even number of faults")
		}
	}
}
