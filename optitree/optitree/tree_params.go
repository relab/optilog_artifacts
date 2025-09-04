package main

import (
	"fmt"
	"testing"
	"time"
)

type simulatedAnnealingParams struct {
	temp        float64
	coolingRate float64
	threshold   float64
	timeout     time.Duration
	scf         int
}

type treeParams struct {
	baseTree     []int
	bf           int
	nNodes       int
	nTrees       int
	treesPerRoot int
	cadence      int
	temp         float64
	coolingRate  float64
	threshold    float64
	timeout      time.Duration
	iterations   int
	faultIndex   int
	scf          int
	scd          int
	faults       int
}

func NewTreeParams(baseTree []int, bf, emitCadence, faults, scd int) treeParams {
	nTrees := NumTrees(TreeSize(bf), bf)
	treesPerRoot := nTrees / len(baseTree)
	return treeParams{
		baseTree:     baseTree,
		bf:           bf,
		nNodes:       len(baseTree),
		nTrees:       nTrees,
		treesPerRoot: treesPerRoot,
		cadence:      EmitCadence(len(baseTree), emitCadence, nTrees),
		scd:          scd,
		faults:       faults,
	}
}

func (s *treeParams) SetSimulatedAnnealingParams(params simulatedAnnealingParams) {
	s.temp = params.temp
	s.coolingRate = params.coolingRate
	s.threshold = params.threshold
	s.timeout = params.timeout
	s.scf = params.scf
}

func (s treeParams) emitEvents() int {
	return s.nTrees / s.cadence
}

// remainingTime returns the number of trees per second and the estimated time to finish.
func (s treeParams) remaining(doneTrees int, now time.Time) (remainingTrees int, treesPerSecond float64, timeToFinish time.Duration) {
	treesPerSecond = float64(s.cadence) / time.Since(now).Seconds()
	remainingTrees = s.treesPerRoot - doneTrees
	return remainingTrees, treesPerSecond, time.Duration(float64(remainingTrees)/treesPerSecond) * time.Second
}

// logNow logs the progress of the analysis every cadence trees. It returns true if the progress was logged,
// so that the caller can update the time.
func (s treeParams) logNow(analyzedTrees int, root int, now time.Time) bool {
	if analyzedTrees%s.cadence == 0 {
		remainingTrees, treesPerSecond, timeToFinish := s.remaining(analyzedTrees, now)
		printf("%2d: Analyzed %d trees, %d remains, %.0f trees/sec, estimated time to finish %v\n",
			root, analyzedTrees, remainingTrees, treesPerSecond, timeToFinish)
		return true
	}
	return false
}

func (s treeParams) treesPerSecond(b *testing.B) float64 {
	return float64((b.N)*s.nTrees) / b.Elapsed().Seconds()
}

func printf(format string, args ...interface{}) {
	if benchmarking {
		return
	}
	fmt.Printf(format, args...)
}
