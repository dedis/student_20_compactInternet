package audit

import (
	"fmt"
	"math"
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

// Check for a lax version of Gao-Rexford rules
func respectsNoValley(routeLinks []int) bool {
	goneDown := false

	for _, ln := range routeLinks {
		goneDown = goneDown || (ln == ToCustomer)

		if goneDown && (ln == ToProvider) {
			return false
		}
	}

	return true
}

type roundChannels struct {
	stretchContribution chan float64
	maxContribution     chan float64
	valleyContribution  chan int
}

func stretchRound(baseline AbstractGraph, audited AbstractGraph, batches int, channels roundChannels) {
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
	var localMax float64
	var localValley int

	// Get predictions
	for b := 0; b < batches; b++ {
		basePath, _ := (baseline).GetRoute(origs[b], dests[b])
		auditPath, auditLinks := (audited).GetRoute(origs[b], dests[b])

		if basePath == nil || auditPath == nil {
			// TODO: Insert useful reaction
			fmt.Println("Unable to compare routes from " + u.Str(origs[b]) + " to " + u.Str(dests[b]))
			continue
		}

		// TODO: Also consider link types
		if !respectsNoValley(auditLinks) {
			localValley++
		}

		var sampleStretch float64
		if len(basePath) == 1 {
			// Origin and destination coincide
			sampleStretch = float64(len(auditPath) - 1)
		} else {
			sampleStretch = float64(len(auditPath)-1) / float64(len(basePath)-1)
		}

		// Update localMax
		if localMax < sampleStretch {
			localMax = sampleStretch
		}

		fmt.Printf("\tRoute %d	from %d to %d	obtained %d vs %d	stretch %f\n", b, origs[b], dests[b], len(auditPath)-1, len(basePath)-1, sampleStretch)
		acc += sampleStretch
	}

	channels.stretchContribution <- acc
	channels.maxContribution <- localMax
	channels.valleyContribution <- localValley
}

// MeasureStretch measures the average path stretch over random paths
// batches : number of routes added per round
// rounds  : number of rounds
// return (averageStretch, maxStretch)
func MeasureStretch(baseline AbstractGraph, audited AbstractGraph, rounds int, batches int) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	stretch := 0.0
	max := 0.0
	valley := 0

	channels := roundChannels{
		stretchContribution: make(chan float64, rounds),
		maxContribution:     make(chan float64, rounds),
		valleyContribution:  make(chan int, rounds),
	}

	for i := 0; i < rounds; i++ {
		baselineCopy := baseline.Copy()
		auditedCopy := audited.Copy()
		go stretchRound(baselineCopy, auditedCopy, batches, channels)
	}

	for i := 0; i < rounds; i++ {
		stretch += <-channels.stretchContribution
		max = math.Max(max, <-channels.maxContribution)
		valley += <-channels.valleyContribution
	}

	fmt.Printf("%f%% of paths do not respec the no-valley rule\n", float64(valley)/float64(rounds*batches)*100)

	return stretch / float64(rounds*batches), max
}
