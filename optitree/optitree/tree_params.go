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
}

type treeParams struct {
	baseTree     []int
	bf           int
	nNodes       int
	nTrees       int64
	treesPerRoot int64
	cadence      int64
	temp         float64
	coolingRate  float64
	threshold    float64
	timeout      time.Duration
	iterations   int
	faults       int
}

func NewTreeParams(baseTree []int, bf, emitCadence int, faults int) treeParams {
	nTrees := NumTrees(TreeSize(bf), bf)
	treesPerRoot := nTrees / int64(len(baseTree))
	return treeParams{
		baseTree:     baseTree,
		bf:           bf,
		nNodes:       len(baseTree),
		nTrees:       nTrees,
		treesPerRoot: treesPerRoot,
		cadence:      EmitCadence(len(baseTree), emitCadence, nTrees),
		faults:       faults,
	}
}

func (s *treeParams) SetSimulatedAnnealingParams(params simulatedAnnealingParams) {
	s.temp = params.temp
	s.coolingRate = params.coolingRate
	s.threshold = params.threshold
	s.timeout = params.timeout
}

func (s treeParams) emitEvents() int64 {
	return s.nTrees / s.cadence
}

// remainingTime returns the number of trees per second and the estimated time to finish.
func (s treeParams) remaining(doneTrees int64, now time.Time) (remainingTrees int64, treesPerSecond float64, timeToFinish time.Duration) {
	treesPerSecond = float64(s.cadence) / time.Since(now).Seconds()
	remainingTrees = s.treesPerRoot - doneTrees
	return remainingTrees, treesPerSecond, time.Duration(float64(remainingTrees)/treesPerSecond) * time.Second
}

// logNow logs the progress of the analysis every cadence trees. It returns true if the progress was logged,
// so that the caller can update the time.
func (s treeParams) logNow(analyzedTrees int64, root int, now time.Time) bool {
	if analyzedTrees%s.cadence == 0 {
		remainingTrees, treesPerSecond, timeToFinish := s.remaining(analyzedTrees, now)
		printf("%2d: Analyzed %d trees, %d remains, %.0f trees/sec, estimated time to finish %v\n",
			root, analyzedTrees, remainingTrees, treesPerSecond, timeToFinish)
		return true
	}
	return false
}

func (s treeParams) treesPerSecond(b *testing.B) float64 {
	return float64(int64(b.N)*s.nTrees) / b.Elapsed().Seconds()
}

func printf(format string, args ...interface{}) {
	if benchmarking {
		return
	}
	fmt.Printf(format, args...)
}
