package main

import (
	"fmt"
	"slices"
	"testing"
)

func TestQCLatency(t *testing.T) {
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		tree     []int
		wantTree []int
		bf       int
		wantLat  Latency
	}{
		{
			name:     "initial/stable",
			bf:       3,
			tree:     []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19}, // starting point (sorted with 0 as root)
			wantTree: []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19},
			wantLat:  602110,
		},
		{
			name:     "optimal/stable",
			bf:       3,
			tree:     []int{3, 1, 2, 0, 5, 7, 15, 8, 17, 19, 9, 12, 16}, // optimal according to FastKauriTrees
			wantTree: []int{3, 1, 2, 0, 5, 7, 15, 8, 17, 19, 9, 12, 16},
			wantLat:  191495,
		},
		{
			name:     "optimal/rearranged",
			bf:       3,
			tree:     []int{3, 0, 1, 2, 9, 12, 16, 5, 7, 15, 8, 17, 19}, // before reordering leaves
			wantTree: []int{3, 1, 2, 0, 5, 7, 15, 8, 17, 19, 9, 12, 16},
			wantLat:  191495,
		},
		{
			name:     "initial/rearranged",
			bf:       4,
			tree:     []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			wantTree: []int{0, 1, 4, 2, 3, 5, 6, 7, 8, 17, 18, 19, 20, 9, 10, 11, 12, 13, 14, 15, 16},
			wantLat:  613165,
		},
		{
			name:     "optimal/stable",
			bf:       4,
			tree:     []int{0, 1, 4, 2, 3, 5, 6, 7, 8, 17, 18, 19, 20, 9, 10, 11, 12, 13, 14, 15, 16},
			wantTree: []int{0, 1, 4, 2, 3, 5, 6, 7, 8, 17, 18, 19, 20, 9, 10, 11, 12, 13, 14, 15, 16},
			wantLat:  613165,
		},
	}
	for _, tt := range tests {
		qs := quorumSize(len(tt.tree))
		t.Run(fmt.Sprintf("%s/size=%d/bf=%d", tt.name, len(tt.tree), tt.bf), func(t *testing.T) {
			nodes := newNodes(tt.tree[0], tt.tree[1:])
			gotLat := latencies.qcLatency(qs, tt.bf, nodes)
			if gotLat != tt.wantLat {
				t.Errorf("qcLatency() = %d; want %d", gotLat, tt.wantLat)
			}
			gotTree := toTree(nodes)
			if !slices.Equal(tt.wantTree, gotTree) {
				t.Fail()
				t.Logf("origTree: %v", tt.tree)
				t.Logf(" gotTree: %v", gotTree)
				t.Logf("wantTree: %v", tt.wantTree)
			}
		})
	}
}

func TestQCOptimalTree(t *testing.T) {
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		tree     []int
		wantTree []int
		want     result
		bf       int
	}{
		{
			bf:       3,
			tree:     []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19},
			want:     result{latency: 191495, analyzedTrees: 4804800},
			wantTree: []int{3, 1, 2, 0, 5, 7, 15, 8, 17, 19, 9, 12, 16},
		},
	}
	for _, tt := range tests {
		params := NewTreeParams(tt.tree, tt.bf, 0, 0)
		t.Run(fmt.Sprintf("size=%d/bf=%d", len(tt.tree), tt.bf), func(t *testing.T) {
			gotOptimal := latencies.QCOptimalTreeChannel(params)
			if gotOptimal.latency != tt.want.latency {
				t.Errorf("QCOptimalTree() = %d; want %d", gotOptimal.latency, tt.want.latency)
			}
			if gotOptimal.analyzedTrees != tt.want.analyzedTrees {
				t.Errorf("QCOptimalTree() = %d; want %d", gotOptimal.analyzedTrees, tt.want.analyzedTrees)
			}
			gotTree := toTree(gotOptimal.nodes)
			if !slices.Equal(tt.wantTree, gotTree) {
				t.Fail()
				t.Logf("origTree: %v", tt.tree)
				t.Logf(" gotTree: %v", gotTree)
				t.Logf("wantTree: %v", tt.wantTree)
			}
		})
	}
}

// Run with: go test -v -run ^$ -bench BenchmarkQCOptimalTree -benchmem -count=5
func BenchmarkQCOptimalTree(b *testing.B) {
	benchmarking = true
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		b.Fatal(err)
	}
	tests := []struct {
		name string
		fn   func(params treeParams) result
		tree []int
		bf   int
	}{
		{
			name: "channel",
			fn:   latencies.QCOptimalTreeChannel,
			bf:   3,
			tree: []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19},
		},
		{
			name: "mutex",
			fn:   latencies.QCOptimalTreeMutex,
			bf:   3,
			tree: []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19},
		},
	}
	for _, tt := range tests {
		params := NewTreeParams(tt.tree, tt.bf, 0, 0)
		b.Run(fmt.Sprintf("name=%s/size=%v/bf=%d/trees=%d", tt.name, params.nNodes, tt.bf, params.nTrees), func(b *testing.B) {
			b.ResetTimer()
			b.StartTimer()
			for range b.N {
				tt.fn(params)
			}
			b.StopTimer()
			// Report the number of trees per second over all iterations
			b.ReportMetric(params.treesPerSecond(b), "trees/sec")
		})
	}
}

// Run with: go test -v -run ^$ -bench BenchmarkQCLatency -benchmem -count=5
func BenchmarkQCLatency(b *testing.B) {
	benchmarking = true
	latencies, err := loadLatencies(awsLatencyFile, "random")
	if err != nil {
		b.Fatal(err)
	}
	tests := []struct {
		name string
		tree []int
		bf   int
	}{
		{
			name: "initial/stable",
			bf:   3,
			tree: []int{0, 1, 2, 3, 5, 7, 8, 9, 12, 15, 16, 17, 19},
		},
		{
			name: "optimal/stable",
			bf:   3,
			tree: []int{3, 1, 2, 0, 5, 7, 15, 8, 17, 19, 9, 12, 16},
		},
		{
			name: "optimal/rearranged",
			bf:   3,
			tree: []int{3, 0, 1, 2, 9, 12, 16, 5, 7, 15, 8, 17, 19},
		},
		{
			name: "initial/rearranged",
			bf:   4,
			tree: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
		{
			name: "optimal/stable",
			bf:   4,
			tree: []int{0, 1, 4, 2, 3, 5, 6, 7, 8, 17, 18, 19, 20, 9, 10, 11, 12, 13, 14, 15, 16},
		},
	}
	for _, tt := range tests {
		qs := quorumSize(len(tt.tree))
		b.Run(fmt.Sprintf("%s/size=%d/bf=%d", tt.name, len(tt.tree), tt.bf), func(b *testing.B) {
			for range b.N {
				nodes := newNodes(tt.tree[0], tt.tree[1:])
				_ = latencies.qcLatency(qs, tt.bf, nodes)
			}
		})
	}
}
