package kauri

import "github.com/relab/hotstuff"

type OptiTree3 struct {
	mg1 Suspicions
	id  hotstuff.ID
}

func NewOptiTree3(id hotstuff.ID, size int) *OptiTree3 {
	return &OptiTree3{
		mg1: make(Suspicions, size),
		id:  id,
	}
}

func (t *OptiTree3) checkForBetterEdge(node, neighbor hotstuff.ID, suspicions Suspicions) {
	nodeEdge := suspicions.findDegreeOneEdge(node)
	neighborEdge := suspicions.findDegreeOneEdge(neighbor)

	if nodeEdge != 0 && neighborEdge != 0 {
		t.updateMG1(node, nodeEdge)
		t.updateMG1(neighbor, neighborEdge)
	}
}

func (t *OptiTree3) updateMG1(a, b hotstuff.ID) {
	// a and b are 1-indexed
	if t.mg1[a-1] == nil {
		t.mg1[a-1] = make([]Suspicion, len(t.mg1))
	}
	if t.mg1[b-1] == nil {
		t.mg1[b-1] = make([]Suspicion, len(t.mg1))
	}
	t.mg1[a-1][b-1] = 1
	t.mg1[b-1][a-1] = 1
}

func (t *OptiTree3) computeMG1(suspicions Suspicions) {
	for i, neighbors := range suspicions {
		node := hotstuff.ID(i + 1)
		for j := range neighbors {
			neighbor := hotstuff.ID(j + 1)
			if t.mg1.isEdgeAcceptable(node, neighbor) {
				t.updateMG1(node, neighbor)
			}
		}
	}
	for i, neighbors := range t.mg1 {
		node := hotstuff.ID(i + 1)
		for j := range neighbors {
			neighbor := hotstuff.ID(j + 1)
			t.checkForBetterEdge(node, neighbor, suspicions)
		}
	}
}
