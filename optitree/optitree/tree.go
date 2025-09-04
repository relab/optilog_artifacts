package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type node struct {
	id           int
	votes        int
	disseminated Latency
	aggregated   Latency
	delivered    Latency
}

func (n node) String() string {
	return fmt.Sprintf("id %d: votes: %d, disseminated: %d, aggregated: %d, delivered: %d", n.id, n.votes, n.disseminated, n.aggregated, n.delivered)
}

func toString(nodes []node) string {
	var builder strings.Builder
	builder.WriteString("[]int{")
	for i, node := range nodes {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("%d", node.id))
	}
	builder.WriteString("}")
	return builder.String()
}

func toTree(nodes []node) []int {
	tree := make([]int, len(nodes))
	for i, node := range nodes {
		tree[i] = node.id
	}
	return tree
}

// basicTree generates a basic tree with the given size,
// where each node has an id from 0 to size-1.
func basicTree(size int) []int {
	tree := make([]int, size)
	for i := range size {
		tree[i] = i
	}
	return tree
}

// generateBaseTree generates a base tree for root of the given size.
func generateBaseTree(size, root int) []int {
	tree := make([]int, 0, size)
	for i := range size {
		if i == root {
			continue
		}
		tree = append(tree, i)
	}
	return tree
}

// newSubtree returns the subtree without the given root.
// This function will panic if the root is not found.
func newSubtree(root int, baseTree []int) []int {
	index := slices.Index(baseTree, root)
	newTree := slices.Clone(baseTree)
	return append(newTree[:index], newTree[index+1:]...)
}

func newNodes(root int, tree []int) []node {
	nodes := make([]node, len(tree)+1)
	nodes[0] = node{id: root, votes: 1}
	for i := range tree {
		nodes[i+1] = node{id: tree[i], votes: 1}
	}
	return nodes
}

// resetNodes resets provided the nodes for reuse avoiding reallocation.
func resetNodes(nodes []node, tree []int) {
	for i := range nodes {
		if i > 0 {
			// since the root is at index 0 (and doesn't change),
			// while the tree does not include the root
			nodes[i].id = tree[i-1]
		}
		nodes[i].votes = 1
		nodes[i].disseminated = 0
		nodes[i].aggregated = 0
		nodes[i].delivered = 0
	}
}

func parseTreeString(s string) ([]int, error) {
	tree := make([]int, 0)
	s = strings.TrimPrefix(s, "[]int")
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	for _, k := range strings.Split(s, ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}
		tree = append(tree, i)
	}
	return tree, nil
}
