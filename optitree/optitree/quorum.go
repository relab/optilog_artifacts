package main

import "math"

// quorumSize returns the number of nodes needed to form a quorum
// for an arbitrary number of nodes.
func quorumSize(n int) int {
	f := (n - 1) / 3
	return int(math.Ceil(float64(n+f+1) / 2.0))
}

// quorumSizeSimple returns the number of nodes needed to form a quorum
// given that the number of nodes n is different from 3f+3.
func quorumSizeSimple(n int) int {
	f := (n - 1) / 3
	return n - f
}
