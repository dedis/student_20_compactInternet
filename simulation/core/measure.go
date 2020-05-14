package core

import (
	"fmt"
	"strings"
)

// TapeMeasure records the distance of an Asn from another node
// In a sense, it's a simplified version of a DijkstraGraph
type TapeMeasure map[int]int64

func (t *TapeMeasure) String() string {
	var sb strings.Builder

	for nd, dist := range *t {
		sb.WriteString(fmt.Sprintf("%d@%d-", nd, dist))
	}

	return sb.String()
}

// InitMeasure initializes a TapeMeasure keeping distance estimates from
// the node 'originAsn'
func InitMeasure(originAsn int) TapeMeasure {
	t := make(TapeMeasure)
	t[originAsn] = 0
	return t
}

func (t *TapeMeasure) createOrApproach(node int, newMeasure int64) {
	if oldMeasure, exists := (*t)[node]; !exists || newMeasure < oldMeasure {
		(*t)[node] = newMeasure
	}
}

// Extend estimates the distance from the origin to 'toAsn', based
// on the distance to the neighboring point 'fromAsn'
func (t *TapeMeasure) Extend(fromAsn int, toAsn int) {
	if fromMeasure, hasFrom := (*t)[fromAsn]; hasFrom {
		t.createOrApproach(toAsn, fromMeasure+EdgeWeight)
	} else {
		panic("Extending measure from unmeasured node")
	}
}

// Mean calculates the mean distance from the origin of the TapeMeasure
func (t *TapeMeasure) Mean() float64 {
	mean := 0.0

	for _, dist := range *t {
		mean += float64(dist)
	}

	mean /= float64(len(*t))

	return mean
}

// Combine returns a TapeMeasure that maintains the best available estimate
// to any given node
// WARNING: The function assume that the two measures have the same (link) origin
func Combine(measureA *TapeMeasure, measureB *TapeMeasure) *TapeMeasure {
	var baseMeasure, otherMeasure *TapeMeasure
	// Optimization, iterate on the smaller set of nodes
	if len(*measureA) >= len(*measureB) {
		baseMeasure = measureA
		otherMeasure = measureB
	} else {
		baseMeasure = measureB
		otherMeasure = measureA
	}

	for nd, est := range *otherMeasure {
		baseMeasure.createOrApproach(nd, est)
	}

	return baseMeasure
}
