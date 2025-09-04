package main

type uniqueTree struct {
	eval func([]int)
	tree []int
	perm []int
	used []bool
	bf   int
}

func UniqueTreesStruct(tree []int, branchFactor int, eval func([]int)) {
	ut := &uniqueTree{
		bf:   branchFactor,
		tree: tree,
		eval: eval,
		perm: make([]int, len(tree)),
		used: make([]bool, len(tree)),
	}
	ut.evalUniqueTree2(0)
}

func (ut *uniqueTree) evalUniqueTree2(pos int) {
	if pos >= len(ut.tree) {
		ut.eval(ut.perm)
		return
	}

	// Find the start of the current subtree
	subtreeStart := pos - (pos % ut.bf)
	for i := range len(ut.tree) {
		if !ut.used[i] {
			// Ensure that the subtree is sorted by not allowing a smaller number
			// to be placed after a larger one within the same subtree
			if pos > subtreeStart && ut.perm[pos-1] > ut.tree[i] {
				continue
			}
			ut.used[i] = true
			ut.perm[pos] = ut.tree[i]
			ut.evalUniqueTree2(pos + 1)
			ut.used[i] = false
		}
	}
}
