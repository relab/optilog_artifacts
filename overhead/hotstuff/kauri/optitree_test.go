package kauri

import (
	"fmt"
	"testing"
	"time"

	"github.com/relab/hotstuff"
)

type TestInternalNodesData struct {
	suspicions map[hotstuff.ID]map[hotstuff.ID]int
	expelled   []hotstuff.ID
}

func TestComputeMG1(t *testing.T) {
	suspicionTests := []TestInternalNodesData{
		{ // All nodes suspect one node
			suspicions: map[hotstuff.ID]map[hotstuff.ID]int{
				1: {
					2: 1,
					3: 1,
					4: 1,
					5: 1,
				},
				2: {
					1: 1,
				},
				3: {
					1: 1,
				},
				4: {
					1: 1,
				},
				5: {
					1: 1,
				},
			},
			expelled: []hotstuff.ID{1},
		},
		{ // All nodes suspect one node
			suspicions: map[hotstuff.ID]map[hotstuff.ID]int{
				1: {
					2: 1,
					3: 1,
				},
				2: {
					1: 1,
					4: 1,
				},
				3: {
					1: 1,
				},
				4: {
					1: 1,
				},
				5: {},
			},
			expelled: []hotstuff.ID{1, 2, 3, 4},
		},
		{ // All nodes suspect one node
			suspicions: map[hotstuff.ID]map[hotstuff.ID]int{
				1: {
					2: 1,
					3: 1,
					4: 1,
				},
				2: {
					1: 1,
					4: 1,
					3: 1,
				},
				3: {
					1: 1,
					2: 1,
				},
				4: {
					1: 1,
					2: 1,
				},
				5: {},
			},
			expelled: []hotstuff.ID{1, 2, 3, 4},
		},
	}

	tree := NewOptiTree(1)
	for _, test := range suspicionTests {
		internal := tree.getTotalInternalNodesSet(test.suspicions)
		t.Logf("Internal nodes: %v\n", internal)
	}
}

func TestGetTotalInternalNodesSet(t *testing.T) {
	tree := NewOptiTree(1)
	latencyMatrix := Latencies{
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 3, 4, 5, 6, 7},
		{0, 1, 0, 2, 3, 4, 5, 6, 7},
		{0, 1, 2, 0, 3, 4, 5, 6, 7},
		{0, 1, 2, 3, 0, 4, 5, 6, 7},
		{0, 1, 2, 3, 4, 0, 5, 6, 7},
		{0, 1, 2, 3, 4, 5, 0, 6, 7},
		{0, 1, 2, 3, 4, 5, 6, 0, 7},
		{0, 1, 2, 3, 4, 5, 6, 7, 0},
	}
	suspicions := map[hotstuff.ID]map[hotstuff.ID]int{
		1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {},
	}
	configuration := []hotstuff.ID{1, 2, 3, 4, 5, 6, 7}
	cleanLatencyMatrix(latencyMatrix)
	treePos := tree.GetTree(suspicions, latencyMatrix, configuration)
	t.Logf("Internal nodes: %v\n", treePos)

	ot1 := NewOptiTree(5)
	treePos = ot1.GetTree(suspicions, latencyMatrix, configuration)
	t.Logf("Internal nodes: %v\n", treePos)
}

func cleanLatencyMatrix(latencyMatrix Latencies) {
	for node, neighbors := range latencyMatrix {
		for neighbor, value := range neighbors {
			latencyMatrix[node][neighbor] = max(value, latencyMatrix[neighbor][node])
		}
	}
}

func TestSimulatedAnnealing(t *testing.T) {
	tree := NewOptiTree(1)
	latencyMatrix := Latencies{
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 2, 3, 4, 5, 6, 7},
		{0, 1, 0, 2, 3, 4, 5, 6, 7},
		{0, 1, 2, 0, 3, 4, 5, 6, 7},
		{0, 1, 2, 3, 0, 4, 5, 6, 7},
		{0, 1, 2, 3, 4, 0, 5, 6, 7},
		{0, 1, 2, 3, 4, 5, 0, 6, 7},
		{0, 1, 2, 3, 4, 5, 6, 0, 7},
		{0, 1, 2, 3, 4, 5, 6, 7, 0},
	}
	suspicions := make(map[hotstuff.ID]map[hotstuff.ID]int)
	configuration := []hotstuff.ID{1, 2, 3, 4, 5, 6, 7}
	cleanLatencyMatrix(latencyMatrix)
	treePos := tree.GetTree(suspicions, latencyMatrix, configuration)
	fmt.Printf("Tree %v\n", treePos)
	treePos = tree.SimulatedAnnealing(treePos, 2*time.Second, 5, latencyMatrix)
	fmt.Printf("Tree %v\n", treePos)
}

func TestLatencies(t *testing.T) {
	latencies, err := loadLatencies("latencies/wonderproxy.csv")
	if err != nil {
		t.Fatal(err)
	}
	latencies.Print([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, 3)
	latencies, err = loadLatencies("latencies/aws.csv")
	if err != nil {
		t.Fatal(err)
	}
	latencies.Print([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, 3)
}
