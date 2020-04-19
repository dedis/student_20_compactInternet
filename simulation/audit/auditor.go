package audit

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

func randomNode(a AbstractGraph) *Node {
	nodes := a.GetNodes()

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

		withValleyFlag := 0
		// TODO: Also consider link types
		if !respectsNoValley(auditLinks) {
			localValley++
			withValleyFlag = 1
		}

		// TODO: Make that thread safe
		record(
			u.Str(len(basePath)-1),
			u.Str(len(auditPath)-1),
			u.Str(withValleyFlag),
		)

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

	stopRecording()
}

// TODO: Think about refactoring
type Recorder struct {
	active bool
	file   *os.File
	rec    *csv.Writer
}

var globalRecorder Recorder = Recorder{
	active: false,
	file:   nil,
	rec:    nil,
}

func InitRecorder(filename string) {
	var err error
	globalRecorder.file, err = os.Create(filename)
	if err != nil {
		panic("Could not create the output file for the auditor")
	}

	globalRecorder.rec = csv.NewWriter(globalRecorder.file)
	globalRecorder.active = true
}

func record(payload ...string) {
	if globalRecorder.active {
		globalRecorder.rec.Write(payload)
	}
}

func stopRecording() {
	if globalRecorder.active {
		globalRecorder.rec.Flush()
		defer globalRecorder.file.Close()
		globalRecorder.active = false
	}
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

// MeasureEdgeDeletionImpact measures the number of nodes that must be updated when a random link fails
// batches: 	 number of random link deletions
// returns (averageImpact, maxImpact)
func MeasureEdgeDeletionImpact(baseline AbstractGraph, audited AbstractGraph, batches int) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	var averageImpact float64
	var maxImpact float64

	for b := 0; b < batches; {

		// Choose a random node (with more than 1 link)
		endpoint := randomNode(audited)
		for len(endpoint.Links) < 2 {
			endpoint = randomNode(audited)
		}

		linksNum := len(endpoint.Links)

		// Choose a random link among the possible ones
		linkIdx := rand.Int() % linksNum
		otherAsn := endpoint.Links[linkIdx]

		success, impactedNodes := audited.RemoveEdge(endpoint.Asn, otherAsn)

		if success {
			// Consider the sample only if it's successful
			b++
			averageImpact += float64(impactedNodes)
			maxImpact = math.Max(maxImpact, float64(impactedNodes))

			otherEndpoint := (*audited.GetNodes())[otherAsn]

			record(
				u.Str(endpoint.Asn),
				u.Str(otherAsn),
				u.Str(len(endpoint.Links)+1),
				u.Str(len(otherEndpoint.Links)+1),
				u.Str(impactedNodes),
			)
		}
	}

	averageImpact /= float64(batches)

	stopRecording()

	return averageImpact, maxImpact
}

func MeasureDeletionStretch(baseline AbstractGraph, audited AbstractGraph, batches int) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	var averageStretchIncrease float64
	var maxStretchIncrease float64

	b := 0
	for b < batches {

		// Choose a random node (with more than 1 link)
		endpoint := randomNode(audited)
		for len(endpoint.Links) < 2 {
			endpoint = randomNode(audited)
		}

		linksNum := len(endpoint.Links)

		// Choose a random link among the possible ones
		linkIdx := rand.Int() % linksNum
		otherAsn := endpoint.Links[linkIdx]

		// TODO: Could process many destinations at a time

		// Measure path lengths before deletion
		baseline.SetDestinations(map[int]bool{otherAsn: true})
		baseline.Evolve()
		baselineBefore, _ := baseline.GetRoute(endpoint.Asn, otherAsn)

		// TODO: Should call evolve, ... even on audited
		auditedBefore, _ := audited.GetRoute(endpoint.Asn, otherAsn)

		baseline.DeleteDestination(otherAsn)
		baseline.Evolve()
		audited.DeleteDestination(otherAsn)
		audited.Evolve()

		baselineSuccess, _ := baseline.RemoveEdge(endpoint.Asn, otherAsn)
		success, impactedNum := audited.RemoveEdge(endpoint.Asn, otherAsn)

		if success {
			// Consider the sample only if it's successful
			b++

			if !baselineSuccess {
				panic("Difference in graphs")
			}

			baseline.SetDestinations(map[int]bool{otherAsn: true})
			baseline.Evolve()
			// TODO: Should do the same with audited

			baselineAfter, _ := baseline.GetRoute(endpoint.Asn, otherAsn)
			auditedAfter, _ := audited.GetRoute(endpoint.Asn, otherAsn)

			if baselineAfter == nil {
				panic("No no-valley path after deletion")
			}

			sampleIncrease := (float64(len(auditedAfter)) / float64(len(baselineAfter))) / (float64(len(auditedBefore)) / float64(len(baselineBefore)))

			averageStretchIncrease += sampleIncrease
			maxStretchIncrease = math.Max(maxStretchIncrease, sampleIncrease)

			/*
				fmt.Printf("BL before: %s		AD before: %s\n", baselineBefore, auditedBefore)
				fmt.Printf("BL after : %s		AD after : %s\n", baselineAfter, auditedAfter)
			*/

			record(
				u.Str(len(baselineBefore)),
				u.Str(len(auditedBefore)),
				u.Str(len(baselineAfter)),
				u.Str(len(auditedAfter)),
			)
		} else if !success && impactedNum > 0 {
			// The graph is no more a connected component
			// Must conclude the test (TODO: Make it restart from fresh graph)
			fmt.Printf("Test aborted after %d samples: detected >1 connected component\n", b)
			break
		}
	}

	averageStretchIncrease /= float64(b)

	stopRecording()

	return averageStretchIncrease, maxStretchIncrease
}
