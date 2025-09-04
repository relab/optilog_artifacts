package kauri

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/relab/hotstuff"
)

type OptiTree struct {
	mg1 map[hotstuff.ID]map[hotstuff.ID]int
	id  hotstuff.ID
}

func NewOptiTree(id hotstuff.ID) *OptiTree {
	return &OptiTree{
		mg1: make(map[hotstuff.ID]map[hotstuff.ID]int, 0),
		id:  id,
	}
}

func nodeDegree(node hotstuff.ID, graph map[hotstuff.ID]map[hotstuff.ID]int) int {
	if _, ok := graph[node]; ok {
		return len(graph[node])
	}
	return 0
}

func zeroDegreeNodes(graph map[hotstuff.ID]map[hotstuff.ID]int) []hotstuff.ID {
	zeroDegreeNodes := make([]hotstuff.ID, 0, len(graph))
	for node, neighbors := range graph {
		if len(neighbors) == 0 {
			zeroDegreeNodes = append(zeroDegreeNodes, node)
		}
	}
	return zeroDegreeNodes
}

func isEdgeAcceptable(node hotstuff.ID, neighbor hotstuff.ID, graph map[hotstuff.ID]map[hotstuff.ID]int) bool {
	return nodeDegree(node, graph) != 1 && nodeDegree(neighbor, graph) != 1
}

func (t *OptiTree) checkForBetterEdge(node hotstuff.ID, neighbor hotstuff.ID, suspicions map[hotstuff.ID]map[hotstuff.ID]int) {
	nodeEdge := hotstuff.ID(0)
	neighborEdge := hotstuff.ID(0)
	for nodeNeighbor := range getNeighbors(node, suspicions) {
		if nodeDegree(nodeNeighbor, suspicions) == 1 {
			nodeEdge = nodeNeighbor
		}
	}

	for nodeNeighbor := range getNeighbors(neighbor, suspicions) {
		if nodeDegree(nodeNeighbor, suspicions) == 1 {
			neighborEdge = nodeNeighbor
		}
	}
	if nodeEdge != 0 && neighborEdge != 0 {
		delete(t.mg1[node], neighbor)
		t.mg1[node][nodeEdge] = 1
		t.mg1[neighbor][neighborEdge] = 1
		t.mg1[nodeEdge][node] = 1
		t.mg1[neighborEdge][neighbor] = 1
	}
}

func getNeighbors(node hotstuff.ID, suspicions map[hotstuff.ID]map[hotstuff.ID]int) map[hotstuff.ID]int {
	return suspicions[node]
}

func (t *OptiTree) computeMG1(suspicions map[hotstuff.ID]map[hotstuff.ID]int) {
	for node := range suspicions {
		t.mg1[node] = make(map[hotstuff.ID]int)
	}
	for node, neighbors := range suspicions {
		for neighbor := range neighbors {
			if isEdgeAcceptable(node, neighbor, t.mg1) {
				if t.mg1[node] == nil {
					t.mg1[node] = make(map[hotstuff.ID]int)
				}
				t.mg1[node][neighbor] = 1
				t.mg1[neighbor][node] = 1
			}
		}
	}
	for node, neighbors := range t.mg1 {
		for neighbor := range neighbors {
			t.checkForBetterEdge(node, neighbor, suspicions)
		}
	}
}

// Perform the set difference operation on two sets A-B
func setMinus(a, b []hotstuff.ID) []hotstuff.ID {
	result := make([]hotstuff.ID, 0, len(a))
	for _, node := range a {
		if !slices.Contains(b, node) {
			result = append(result, node)
		}
	}
	return result
}

// Check all edges(va,vb) in mg1  if there are any vertices in v0 that (va,v0) and (v0,vb) are in suspicions
func (t *OptiTree) computeV2(suspicions map[hotstuff.ID]map[hotstuff.ID]int, v0 []hotstuff.ID) []hotstuff.ID {
	v2 := make([]hotstuff.ID, 0)
	for node, neighbors := range t.mg1 {
		for neighbor := range neighbors {
			for _, v := range v0 {
				_, ok1 := suspicions[node][v]
				_, ok2 := suspicions[v][neighbor]
				if ok1 && ok2 {
					v2 = append(v2, v)
				}
			}
		}
	}
	return v2
}

func (ot *OptiTree) getTotalInternalNodesSet(suspicions map[hotstuff.ID]map[hotstuff.ID]int) []hotstuff.ID {
	if len(suspicions) != 0 {
		ot.computeMG1(suspicions)
		v0 := zeroDegreeNodes(ot.mg1)
		v2 := ot.computeV2(suspicions, v0)
		return setMinus(v0, v2)
	}
	return nil
}

func (ot *OptiTree) GetInternalSet(suspicions map[hotstuff.ID]map[hotstuff.ID]int,
	latencyMatrix Latencies,
) []hotstuff.ID {
	totalSet := ot.getTotalInternalNodesSet(suspicions)
	if len(totalSet) == 0 {
		return nil
	}
	if pos := slices.Index(totalSet, ot.id); pos != -1 {
		slices.SortFunc(totalSet, func(i, j hotstuff.ID) int {
			return int(latencyMatrix[ot.id][i] - latencyMatrix[ot.id][j])
		})
		temp := totalSet[0]
		totalSet[0] = totalSet[pos]
		totalSet[pos] = temp
	}
	return totalSet[:MaxChild+1]
}

func (ot *OptiTree) GetTree(suspicions map[hotstuff.ID]map[hotstuff.ID]int,
	latencyMatrix Latencies,
	configuration []hotstuff.ID,
) map[hotstuff.ID]int {
	internalSet := ot.GetInternalSet(suspicions, latencyMatrix)
	if len(internalSet) == 0 {
		internalSet = configuration
	}
	treePos := make(map[hotstuff.ID]int)
	pos := 0
	for _, id := range internalSet {
		treePos[id] = pos
		pos++
	}
	leafNodes := setMinus(configuration, internalSet)
	for index, id := range internalSet {
		if index == 0 {
			continue
		}

		for count := 0; count < MaxChild; count++ {
			if len(leafNodes) == 0 {
				break
			}
			leafNode := findNearestReplica(id, leafNodes, latencyMatrix)
			treePos[leafNode] = pos
			pos++
			for j, temp := range leafNodes {
				if temp == leafNode {
					leafNodes = append(leafNodes[:j], leafNodes[j+1:]...)
					break
				}
			}
		}
	}
	return treePos
}

func findNearestReplica(id hotstuff.ID, leafNodes []hotstuff.ID, latencyMatrix Latencies) hotstuff.ID {
	minLatency := Latency(0)
	nearestReplica := hotstuff.ID(0)
	for _, leafNode := range leafNodes {
		if minLatency == 0 {
			minLatency = latencyMatrix[id][leafNode]
			nearestReplica = leafNode
		} else if latencyMatrix[id][leafNode] < minLatency {
			minLatency = latencyMatrix[id][leafNode]
			nearestReplica = leafNode
		}
	}
	return nearestReplica
}

func (ot *OptiTree) SimulatedAnnealing(tree map[hotstuff.ID]int, duration time.Duration, costFunctionThreshold int,
	latencyMatrix Latencies,
) map[hotstuff.ID]int {
	// Simulated Annealing parameters
	temp := 25000.0
	coolingRate := 0.0055
	threshold := 0.5
	timerChan := make(chan bool)
	predictX := qcLatency(costFunctionThreshold, tree, latencyMatrix)
	go func() {
		timer := time.NewTimer(duration)
		<-timer.C
		timerChan <- true
	}()
	isDone := false
	for temp > threshold && !isDone {
		select {
		case <-timerChan:
			isDone = true
		default:
			// Create new solution
			newSolution := mutate(tree)
			predictY := qcLatency(costFunctionThreshold, tree, latencyMatrix)
			if predictY < predictX {
				tree = newSolution
				predictX = predictY
			} else {
				random := rand.Float64()
				if math.Exp(-(float64((predictY - predictX)) / (temp))) > random {
					tree = newSolution
					predictX = predictY
				}
			}
			// Cool system down
			temp *= 1 - coolingRate
		}
	}
	return tree
}

// mutate only leaf nodes.
func mutate(tree map[hotstuff.ID]int) map[hotstuff.ID]int {
	newTree := make(map[hotstuff.ID]int)
	for k, v := range tree {
		newTree[k] = v
	}
	changeNode1Pos := rand.IntN(len(tree))
	changeNode2Pos := rand.IntN(len(tree))

	for changeNode1Pos != 0 || changeNode2Pos != 0 {
		changeNode1Pos = rand.IntN(len(tree))
		changeNode2Pos = rand.IntN(len(tree))
	}
	id1 := hotstuff.ID(0)
	id2 := hotstuff.ID(0)
	for id, pos := range newTree {
		if changeNode1Pos == pos {
			id1 = id
		}
		if changeNode2Pos == pos {
			id2 = id
		}
	}
	newTree[id1], newTree[id2] = newTree[id2], newTree[id1]
	return newTree
}

// qcLatency returns the latency to obtain a quorum certificate (QC) from the give tree.
func qcLatency(quorumSize int, tree map[hotstuff.ID]int, latencyMatrix Latencies) Latency {
	all := make([]hotstuff.ID, len(tree))
	for id, pos := range tree {
		all[pos] = id
	}
	root, internalNodes := all[0], all[1:MaxChild+1]
	cInternalNodes := make([]hotstuff.ID, len(internalNodes))
	// aggregationLatency at internal nodes
	aggregationLatency := make(map[hotstuff.ID]Latency)
	for index, internal := range internalNodes {
		aggregationLatency[internal] = 0
		for j := 1; j <= MaxChild; j++ {
			leafIndex := MaxChild*(index+1) + j
			aggregationLatency[internal] = max(aggregationLatency[internal], latencyMatrix[internal][all[leafIndex]]+latencyMatrix[all[leafIndex]][internal])
		}
		aggregationLatency[internal] += latencyMatrix[internal][root] + latencyMatrix[root][internal]
	}

	copy(internalNodes, cInternalNodes)

	slices.SortFunc(cInternalNodes, func(i, j hotstuff.ID) int {
		return int(aggregationLatency[i] - aggregationLatency[j])
	})
	// Collect QC latency
	votes := 0
	rootAggregated := Latency(0)
	for _, internal := range cInternalNodes {
		if votes >= quorumSize {
			return rootAggregated
		}
		rootAggregated = max(rootAggregated, aggregationLatency[internal])
		votes += MaxChild + 1
	}
	return rootAggregated
}
