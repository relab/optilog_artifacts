package kauri

import (
	"math/rand/v2"
	"slices"

	"github.com/relab/hotstuff"
)

type TreeConfig []hotstuff.ID

// IntN returns a random index in the range [1, len(tc)].
func (tc TreeConfig) IntN() int {
	return rand.IntN(len(tc))
}

type Suspicion int

type Suspicions [][]Suspicion

func (s Suspicions) suspected(a, b hotstuff.ID) bool {
	// a and b are 1-indexed
	return s[a-1][b-1] > 0
}

// degree returns the number of nodes that suspect node.
func (s Suspicions) degree(node hotstuff.ID) int {
	count := 0
	for _, suspicion := range s[node-1] { // node is 1-indexed
		if suspicion > 0 {
			count++
		}
	}
	return count
}

// zeroDegreeNodes returns a slice of nodes whose node degree 0 in the suspicion graph.
func (s Suspicions) zeroDegreeNodes() []hotstuff.ID {
	zeroDegreeNodes := make([]hotstuff.ID, 0, len(s))
	for i := range s {
		node := hotstuff.ID(i + 1)
		if s.degree(node) == 0 {
			zeroDegreeNodes = append(zeroDegreeNodes, node)
		}
	}
	return zeroDegreeNodes
}

// isEdgeAcceptable returns true if the edge between a and b is acceptable.
func (s Suspicions) isEdgeAcceptable(a, b hotstuff.ID) bool {
	// TODO(meling): does this function make sense?
	// It returns true:
	// - If a or b have degree 1, then the edge is not acceptable.
	// - If a and b have degree > 1, then the edge is acceptable.
	// - If a and b have degree 0, then the edge is acceptable.
	// - If a has degree 0 and b has degree > 1, then the edge is acceptable.
	return s.degree(a) != 1 && s.degree(b) != 1
}

// findDegreeOneEdge returns a neighbor of node that has degree 1.
// If no such neighbor exists, it returns 0.
func (s Suspicions) findDegreeOneEdge(node hotstuff.ID) hotstuff.ID {
	for i := range s[node-1] {
		neighbor := hotstuff.ID(i + 1)
		if s.degree(neighbor) == 1 {
			return neighbor
		}
	}
	return 0
}

// mutate swaps two elements in the tree.
func mutate2(tree TreeConfig) TreeConfig {
	idx1, idx2 := tree.IntN(), tree.IntN()
	// Ensure that idx1 != idx2
	for idx1 == idx2 {
		idx2 = tree.IntN()
	}
	newTree := slices.Clone(tree)
	newTree[idx1], newTree[idx2] = newTree[idx2], newTree[idx1]
	return newTree
}
