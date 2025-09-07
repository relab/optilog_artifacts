package kauri

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/relab/hotstuff"
)

func TestIntN(t *testing.T) {
	tc := TreeConfig{1, 2, 3, 4, 5, 6, 7}
	for i := 0; i < 100; i++ {
		got := tc.IntN()
		if got < 0 {
			t.Errorf("IntN() = %d, want >= 0", got)
		}
		if got > len(tc) {
			t.Errorf("IntN() = %d, want <= %d", got, len(tc))
		}
	}
}

func TestMutate(t *testing.T) {
	tree := TreeConfig{1, 2, 3, 4, 5, 6, 7}
	for i := 0; i < 1000; i++ {
		mutatedTree := mutate2(tree)
		if len(mutatedTree) != len(tree) {
			t.Errorf("Expected mutatedTree to have the same length as tree")
		}
		if cmp.Equal(mutatedTree, tree) {
			t.Errorf("Expected mutatedTree to be different from tree")
		}
		tree = mutatedTree
	}
}

func TestNodeDegree(t *testing.T) {
	tests := []struct {
		name          string
		suspicions    Suspicions
		nodes         []hotstuff.ID
		wantDegrees   []int
		wantZeroNodes []hotstuff.ID
	}{
		{
			name: "All nodes suspect every other node",
			suspicions: Suspicions{
				{0, 1, 1, 1, 1},
				{1, 0, 1, 1, 1},
				{1, 1, 0, 1, 1},
				{1, 1, 1, 0, 1},
				{1, 1, 1, 1, 0},
			},
			nodes:         []hotstuff.ID{1, 2, 3, 4, 5},
			wantDegrees:   []int{4, 4, 4, 4, 4},
			wantZeroNodes: []hotstuff.ID{},
		},
		{
			name: "No nodes suspect any other node",
			suspicions: Suspicions{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
			},
			nodes:         []hotstuff.ID{1, 2, 3, 4, 5},
			wantDegrees:   []int{0, 0, 0, 0, 0},
			wantZeroNodes: []hotstuff.ID{1, 2, 3, 4, 5},
		},
		{
			name: "Node 1 suspects all other nodes",
			suspicions: Suspicions{
				{0, 1, 1, 1, 1},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
			},
			nodes:         []hotstuff.ID{1, 2, 3, 4, 5},
			wantDegrees:   []int{4, 1, 1, 1, 1},
			wantZeroNodes: []hotstuff.ID{},
		},
		{
			name: "Node suspicions: 1->{2,3}, 2->{1,4}, 3->{1}, 4->{1}, 5->{}",
			suspicions: Suspicions{
				{0, 1, 1, 0, 0}, // 1->{2,3}
				{1, 0, 0, 1, 0}, // 2->{1,4}
				{1, 0, 0, 0, 0}, // 3->{1}
				{1, 0, 0, 0, 0}, // 4->{1}
				{0, 0, 0, 0, 0}, // 5->{}
			},
			nodes:         []hotstuff.ID{1, 2, 3, 4, 5},
			wantDegrees:   []int{2, 2, 1, 1, 0},
			wantZeroNodes: []hotstuff.ID{5},
		},
		{
			name: "Node suspicions: 1->{2,3,4}, 2->{1,3,4}, 3->{1,2}, 4->{1,2}, 5->{}",
			suspicions: Suspicions{
				{0, 1, 1, 1, 0}, // 1->{2,3,4}
				{1, 0, 1, 1, 0}, // 2->{1,3,4}
				{1, 1, 0, 0, 0}, // 3->{1,2}
				{1, 1, 0, 0, 0}, // 4->{1,2}
				{0, 0, 0, 0, 0}, // 5->{}
			},
			nodes:         []hotstuff.ID{1, 2, 3, 4, 5},
			wantDegrees:   []int{3, 3, 2, 2, 0},
			wantZeroNodes: []hotstuff.ID{5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, node := range tt.nodes {
				if got := tt.suspicions.degree(node); got != tt.wantDegrees[i] {
					t.Errorf("degree(%d) = %d, want %d", node, got, tt.wantDegrees[i])
				}
			}
			if got := tt.suspicions.zeroDegreeNodes(); !cmp.Equal(got, tt.wantZeroNodes) {
				t.Errorf("zeroDegreeNodes() = %v, want %v", got, tt.wantZeroNodes)
			}
		})
	}
}

func TestIsEdgeAcceptable(t *testing.T) {
	tests := []struct {
		name                string
		suspicions          Suspicions
		nodes               []hotstuff.ID
		wantAcceptableEdges [][]bool
	}{
		{
			name: "All nodes suspect every other node",
			suspicions: Suspicions{
				{0, 1, 1, 1, 1},
				{1, 0, 1, 1, 1},
				{1, 1, 0, 1, 1},
				{1, 1, 1, 0, 1},
				{1, 1, 1, 1, 0},
			},
			nodes: []hotstuff.ID{1, 2, 3, 4, 5},
			wantAcceptableEdges: [][]bool{
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
			},
		},
		{
			name: "No nodes suspect any other node",
			suspicions: Suspicions{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
			},
			nodes: []hotstuff.ID{1, 2, 3, 4, 5},
			wantAcceptableEdges: [][]bool{
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
			},
		},
		{
			name: "Node 1 suspects all other nodes",
			suspicions: Suspicions{
				{0, 1, 1, 1, 1},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
				{1, 0, 0, 0, 0},
			},
			nodes: []hotstuff.ID{1, 2, 3, 4, 5},
			wantAcceptableEdges: [][]bool{
				{true, false, false, false, false},
				{false, false, false, false, false},
				{false, false, false, false, false},
				{false, false, false, false, false},
				{false, false, false, false, false},
			},
		},
		{
			name: "Node suspicions: 1->{2,3}, 2->{1,4}, 3->{1}, 4->{1}, 5->{}",
			suspicions: Suspicions{
				{0, 1, 1, 0, 0}, // 1->{2,3}
				{1, 0, 0, 1, 0}, // 2->{1,4}
				{1, 0, 0, 0, 0}, // 3->{1}
				{1, 0, 0, 0, 0}, // 4->{1}
				{0, 0, 0, 0, 0}, // 5->{}
			},
			nodes: []hotstuff.ID{1, 2, 3, 4, 5},
			wantAcceptableEdges: [][]bool{
				{true, true, false, false, true},
				{true, true, false, false, true},
				{false, false, false, false, false},
				{false, false, false, false, false},
				{true, true, false, false, true},
			},
		},
		{
			name: "Node suspicions: 1->{2,3,4}, 2->{1,3,4}, 3->{1,2}, 4->{1,2}, 5->{}",
			suspicions: Suspicions{
				{0, 1, 1, 1, 0}, // 1->{2,3,4}
				{1, 0, 1, 1, 0}, // 2->{1,3,4}
				{1, 1, 0, 0, 0}, // 3->{1,2}
				{1, 1, 0, 0, 0}, // 4->{1,2}
				{0, 0, 0, 0, 0}, // 5->{}
			},
			nodes: []hotstuff.ID{1, 2, 3, 4, 5},
			wantAcceptableEdges: [][]bool{
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
				{true, true, true, true, true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, a := range tt.nodes {
				for j, b := range tt.nodes {
					if got := tt.suspicions.isEdgeAcceptable(a, b); got != tt.wantAcceptableEdges[i][j] {
						t.Errorf("isEdgeAcceptable(%d, %d) = %t, want %t", a, b, got, tt.wantAcceptableEdges[i][j])
					}
				}
			}
		})
	}
}
