package main

import (
	"math/big"

	"gonum.org/v1/gonum/stat/combin"
)

func NumTrees(n, k int) int {
	result := n
	for i := 0; i <= k; i++ {
		result *= combin.Binomial(n-1-i*k, k)
	}
	return result
}

func NumTrees2(n, k int) *big.Int {
	result := big.NewInt(int64(n))
	for i := 0; i <= k; i++ {
		temp := big.NewInt(1)
		result.Mul(result, temp.Binomial(int64(n-1-i*k), int64(k)))
	}
	return result
}

func TreeSize(bf int) int {
	return bf*bf + bf + 1
}

// EmitCadence returns the number of times to emit intermediate results.
// If cadence is 0, no intermediate results are emitted.
// If cadence is 1, one intermediate result is emitted from each subtree.
// If cadence is 2, two intermediate results are emitted from each subtree, and so on.
func EmitCadence(n, cadence int, nTrees int) int {
	if cadence == 0 {
		return nTrees
	}
	return nTrees / (n * cadence)
}
