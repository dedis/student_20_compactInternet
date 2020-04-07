package audit

import (
	"fmt"
	"math/rand"
	"time"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

func randomNode(a AbstractGraph) *Node {
	nodes := (a).GetNodes()

	stop := rand.Int() % len(*nodes)

	for k := range *nodes {
		if stop == 0 {
			return (*nodes)[k]
		}
		stop--
	}

	return nil
}

func stretchRound(baseline AbstractGraph, audited AbstractGraph, batches int, contribution chan float64) {
	origs := make([]int, 0, batches)
	dests := make([]int, 0, batches)
	for b := 0; b < batches; b++ {
		// Choose endpoints (baseline.Nodes == audited.Nodes)
		origs = append(origs, randomNode(baseline).Asn)
		dest := randomNode(baseline)
		dests = append(dests, dest.Asn)

		// Only destinations must be declared
		(baseline).SetDestinations(map[int]bool{dest.Asn: true})
		(audited).SetDestinations(map[int]bool{dest.Asn: true})
	}

	// Evolve graphs (if needed)
	(baseline).Evolve()
	(audited).Evolve()

	var acc float64

	// Get predictions
	for b := 0; b < batches; b++ {
		basePath, _ := (baseline).GetRoute(origs[b], dests[b])
		auditPath, _ := (audited).GetRoute(origs[b], dests[b])

		if basePath == nil || auditPath == nil {
			// TODO: Insert useful reaction
			fmt.Println("Unable to compare routes from " + u.Str(origs[b]) + " to " + u.Str(dests[b]))
			continue
		}

		// TODO: Also consider link types

		sampleStretch := float64(len(auditPath)-1) / float64(len(basePath)-1)

		fmt.Printf("\tRoute %d	from %d to %d	obtained %d vs %d	stretch %f\n", b, origs[b], dests[b], len(auditPath)-1, len(basePath)-1, sampleStretch)
		acc += sampleStretch
	}

	contribution <- acc
}

// MeasureStretch measures the average path stretch over random paths
// batches : number of routes added per round
// rounds  : number of rounds
func MeasureStretch(baseline AbstractGraph, audited AbstractGraph, rounds int, batches int) float64 {

	rand.Seed(time.Now().UnixNano())

	stretch := 0.0

	contributions := make(chan float64, rounds)

	for i := 0; i < rounds; i++ {
		baselineCopy := baseline.Copy()
		auditedCopy := audited.Copy()
		go stretchRound(baselineCopy, auditedCopy, batches, contributions)
	}

	for i := 0; i < rounds; i++ {
		stretch += <-contributions
	}

	return stretch / float64(rounds*batches)
}
