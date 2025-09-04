package main

// UniqueTrees generates all unique permutations of a root's subtree
// such that each subtree group of size branchFactor is sorted.
// That is, each tree represents a unique tree (ignoring symmetric trees).
//
// The input tree must contain the nodes of the tree, excluding the root node.
// The function should be called once for each root node, typically in parallel.
// The tree slice must be sorted.
//
// The provided eval function is called with each generated tree.
// The eval function must not modify the tree slice provided to it.
func UniqueTrees(tree []int, branchFactor int, eval func([]int)) {
	perm := make([]int, len(tree))
	used := make([]bool, len(tree))
	evalUniqueTree(0, tree, perm, used, branchFactor, eval)
}

// evalUniqueTree is a recursive function that generates all unique permutations
// of the tree slice such that each subtree group of size branchFactor is sorted.
func evalUniqueTree(pos int, tree, perm []int, used []bool, bf int, eval func([]int)) {
	if pos >= len(tree) {
		eval(perm)
		return
	}

	// Find the start of the current subtree
	subtreeStart := pos - (pos % bf)
	for i := 0; i < len(tree); i++ {
		if !used[i] {
			// Ensure that the subtree is sorted by not allowing a smaller number
			// to be placed after a larger one within the same subtree
			if pos > subtreeStart && perm[pos-1] > tree[i] {
				continue
			}
			used[i] = true
			perm[pos] = tree[i]
			evalUniqueTree(pos+1, tree, perm, used, bf, eval)
			used[i] = false
		}
	}
}
