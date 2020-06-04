package audit

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/tz"
	"dedis.epfl.ch/u"
)

// Returns a randomly chosen node of the AbstractGraph
func RandomNode(a AbstractGraph) *Node {
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

// RandomLink returns a randomly chosen link
// with the form of one of its endpoints and the index of the link to the other endpoint
func RandomLink(a AbstractGraph, linksNum int) (*Node, int) {
	nodes := a.GetNodes()

	// Must be doubled, since each edge appears twice
	linksNum *= 2

	stop := rand.Int() % linksNum

	for _, n := range *nodes {
		stop -= len(n.Links)
		if stop <= 0 {
			return n, len(n.Links) + stop - 1
		}
	}

	return nil, -1
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

// TODO: Refactor these 2 functions
func formatPath(path []*Node) string {
	var sbPath strings.Builder
	for _, n := range path {
		sbPath.WriteString(u.Str(n.Asn) + ">")
	}
	return sbPath.String()
}

func formatAsnPath(asnPath []int) string {
	var sbAsnPath strings.Builder
	for _, n := range asnPath {
		sbAsnPath.WriteString(u.Str(n) + ">")
	}
	return sbAsnPath.String()
}

func formatTypes(types []int) string {
	var sbType strings.Builder
	for _, t := range types {
		sbType.WriteString(u.Str(t) + ">")
	}
	return sbType.String()
}

func stretchRound(baseline AbstractGraph, audited AbstractGraph, batches int, disconnectedNodes map[int]bool, channels roundChannels) {
	origs := make([]int, 0, batches)
	dests := make([]int, 0, batches)

	for b := 0; b < batches; b++ {
		// Choose endpoints (baseline.Nodes == audited.Nodes)

		// Choose endpoint that are reachable
		// TODO: Maybe refactor this
		var or *Node
		var ds *Node
		for {
			or = RandomNode(baseline)
			ds = RandomNode(baseline)

			_, orDisconnected := disconnectedNodes[or.Asn]
			_, dsDisconnected := disconnectedNodes[ds.Asn]

			if !orDisconnected && !dsDisconnected {
				break
			}
		}

		origs = append(origs, or.Asn)
		dests = append(dests, ds.Asn)

		// Only destinations must be declared
		baseline.SetDestinations(map[int]bool{ds.Asn: true})
		audited.SetDestinations(map[int]bool{ds.Asn: true})
	}

	// Evolve graphs (if needed)
	baseline.Evolve()
	audited.Evolve()

	var acc float64
	var localMax float64
	var localValley int

	// Get predictions
	for b := 0; b < batches; b++ {
		basePath, baseLinks := (baseline).GetRoute(origs[b], dests[b])
		auditPath, auditLinks := (audited).GetRoute(origs[b], dests[b])

		if basePath == nil || auditPath == nil {
			// TODO: Insert useful reaction
			// println disabled to clean tmux logs
			//fmt.Println("Unable to compare routes from " + u.Str(origs[b]) + " to " + u.Str(dests[b]))
			continue
		}

		withValleyFlag := 0
		if !respectsNoValley(auditLinks) {
			localValley++
			withValleyFlag = 1
		}

		record(
			u.Str(len(basePath)-1),
			u.Str(len(auditPath)-1),
			u.Str(withValleyFlag),
			formatPath(basePath),
			formatTypes(baseLinks),
			formatPath(auditPath),
			formatTypes(auditLinks),
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

		// fmt.Printf("\tRoute %d	from %d to %d	obtained %d vs %d	stretch %f\n", b, origs[b], dests[b], len(auditPath)-1, len(basePath)-1, sampleStretch)
		acc += sampleStretch
	}

	// Delete destinations
	for _, d := range dests {
		baseline.DeleteDestination(d)
		audited.DeleteDestination(d)
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
		go stretchRound(baselineCopy, auditedCopy, batches, map[int]bool{}, channels)
	}

	for i := 0; i < rounds; i++ {
		stretch += <-channels.stretchContribution
		max = math.Max(max, <-channels.maxContribution)
		valley += <-channels.valleyContribution
	}

	fmt.Printf("%f%% of paths do not respec the no-valley rule\n", float64(valley)/float64(rounds*batches)*100)

	stopRecording()

	return stretch / float64(rounds*batches), max
}

func loadEdgeDeletionsFile(filename string) [][]int {

	csvFile, err := os.Open(filename)
	if err != nil {
		panic("Unable to load sequence of edges deletions")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	// deletionsList contains the endpoints of edges to delete
	// (there is no rule on endpoints ordering)
	deletionsList := make([][]int, 0, 64)

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		deletionsList = append(deletionsList, []int{u.Int(row[0]), u.Int(row[1])})
	}

	return deletionsList
}

// MeasureChosenEdgeDeletionImpact measures the number of nodes that must be updated when a sequence of edge
// deletions is executed
// deletionsFilename: 	 path to csv file containing the sequence of deletions
// returns (averageImpact, maxImpact)
func MeasureChosenEdgeDeletionImpact(audited AbstractGraph, deletionsFilename string) (float64, float64) {

	var averageImpact float64
	var maxImpact float64

	linksNum := audited.CountLinks()

	deletionsList := loadEdgeDeletionsFile(deletionsFilename)

	var b int

	for _, endpoints := range deletionsList {
		// Delete link from the graph
		success, impactedArea, impactedMeasure := audited.RemoveEdge(endpoints[0], endpoints[1])
		impactedNodes := len(impactedArea)
		linksNum--

		if success {
			// Consider the sample only if it's successful
			b++
			averageImpact += float64(impactedNodes)
			maxImpact = math.Max(maxImpact, float64(impactedNodes))

			endA := (*audited.GetNodes())[endpoints[0]]
			endB := (*audited.GetNodes())[endpoints[1]]

			record(
				u.Str(endpoints[0]),
				u.Str(endpoints[1]),
				u.Str(len(endA.Links)+1),
				u.Str(len(endB.Links)+1),
				u.Str(impactedNodes),
				impactedMeasure.String(),
			)
		}
	}

	averageImpact /= float64(b)

	stopRecording()

	return averageImpact, maxImpact
}

// MeasureEdgeDeletionImpact measures the number of nodes that must be updated when a random link fails
// batches: 	 number of random link deletions
// returns (averageImpact, maxImpact)
func MeasureEdgeDeletionImpact(baseline AbstractGraph, audited AbstractGraph, batches int) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	var averageImpact float64
	var maxImpact float64

	linksNum := audited.CountLinks()

	var b int

	for b < batches {

		// Choose a random link (from a node with more than 1 link)
		endpoint, linkIdx := RandomLink(audited, linksNum)
		for len(endpoint.Links) < 2 {
			endpoint, linkIdx = RandomLink(audited, linksNum)
		}

		otherAsn := endpoint.Links[linkIdx]

		// Delete link from the graph
		success, impactedArea, impactedMeasure := audited.RemoveEdge(endpoint.Asn, otherAsn)
		impactedNodes := len(impactedArea)
		linksNum--

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
				impactedMeasure.String(),
			)
		}
	}

	averageImpact /= float64(b)

	stopRecording()

	return averageImpact, maxImpact
}

// MeasureDeletionStretch computes the relative increase in stretch after link deletion
// the ONLY considered routes are the one between 2 neighboring nodes (in the original graph)
func MeasureDeletionStretch(baselineOriginal AbstractGraph, auditedOriginal AbstractGraph, batches int) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	// Conduct measurements on a copy of the graphs
	baseline := baselineOriginal.Copy()
	audited := auditedOriginal.Copy()

	var averageStretchIncrease float64
	var maxStretchIncrease float64

	linksNum := audited.CountLinks()

	b := 0
	for b < batches {

		// Choose a random link (with endpoint with more than 1 link)
		endpoint, linkIdx := RandomLink(audited, linksNum)
		for len(endpoint.Links) < 2 || len((*audited.GetNodes())[endpoint.Links[linkIdx]].Links) < 2 {
			endpoint, linkIdx = RandomLink(audited, linksNum)
		}

		otherAsn := endpoint.Links[linkIdx]

		// TODO: Could process many destinations at a time

		// Measure path lengths before deletion
		baseline.SetDestinations(map[int]bool{otherAsn: true})
		baseline.Evolve()
		baselineBefore, baselineTypesBefore := baseline.GetRoute(endpoint.Asn, otherAsn)

		audited.SetDestinations(map[int]bool{otherAsn: true})
		audited.Evolve()
		auditedBefore, auditedTypesBefore := audited.GetRoute(endpoint.Asn, otherAsn)
		linksNum--

		baseline.DeleteDestination(otherAsn)
		baseline.Evolve()
		audited.DeleteDestination(otherAsn)
		audited.Evolve()

		baselineSuccess, _, _ := baseline.RemoveEdge(endpoint.Asn, otherAsn)
		success, impactedArea, _ := audited.RemoveEdge(endpoint.Asn, otherAsn)

		impactedNum := len(impactedArea)

		if success {
			if !baselineSuccess {
				panic("Difference in graphs")
			}

			baseline.SetDestinations(map[int]bool{otherAsn: true})
			baseline.Evolve()
			// TODO: Should do the same with audited

			baselineAfter, baselineTypesAfter := baseline.GetRoute(endpoint.Asn, otherAsn)
			auditedAfter, auditedTypesAfter := audited.GetRoute(endpoint.Asn, otherAsn)

			if baselineAfter == nil {
				// After the deletion, there is no path respecting GR rules in the original graph (only paths with valleys)
				continue
			}

			// Consider the sample only if it's successful
			b++

			sampleIncrease := (float64(len(auditedAfter)) / float64(len(baselineAfter))) / (float64(len(auditedBefore)) / float64(len(baselineBefore)))

			averageStretchIncrease += sampleIncrease
			maxStretchIncrease = math.Max(maxStretchIncrease, sampleIncrease)

			record(
				u.Str(len(baselineBefore)),
				formatPath(baselineBefore),
				formatTypes(baselineTypesBefore),
				u.Str(len(auditedBefore)),
				formatPath(auditedBefore),
				formatTypes(auditedTypesBefore),
				u.Str(len(baselineAfter)),
				formatPath(baselineAfter),
				formatTypes(baselineTypesAfter),
				u.Str(len(auditedAfter)),
				formatPath(auditedAfter),
				formatTypes(auditedTypesAfter),
			)
		} else if !success && impactedNum > 0 {
			// Game over! The graph is no more a connected component
			// Start with fresh copies
			fmt.Printf("Starting from fresh graphs after %d samples (detected > 1 connected component)\n", b)
			baseline = baselineOriginal.Copy()
			audited = auditedOriginal.Copy()

			// Recount links
			linksNum = audited.CountLinks()
		}
	}

	averageStretchIncrease /= float64(b)

	stopRecording()

	return averageStretchIncrease, maxStretchIncrease
}

// MeasureLandmarkLevelAfterDeletion stores which level of landmarks is used
// to compute the path between to adjacent nodes, before and after the edge
// connecting them is deleted
// WARNING: Only works on tz.Graph
func MeasureLandmarkLevelAfterDeletion(baselineGraph AbstractGraph, auditedGraph *tz.Graph, samples int) {

	baseline := baselineGraph.Copy()
	audited := auditedGraph.CopyAsTz()

	rand.Seed(time.Now().UnixNano())

	linksNum := audited.CountLinks()

	var s int = 0

	for s < samples {

		// Choose a random link (with endpoint with more than 1 link)
		endpoint, linkIdx := RandomLink(audited, linksNum)
		for len(endpoint.Links) < 4 || len(audited.Nodes[endpoint.Links[linkIdx]].Links) < 4 {
			endpoint, linkIdx = RandomLink(audited, linksNum)
		}

		otherAsn := endpoint.Links[linkIdx]

		// TODO: Could process many destinations at a time

		// Measure path lengths before deletion
		baseline.SetDestinations(map[int]bool{otherAsn: true})
		baseline.Evolve()
		baselineBefore, baselineTypesBefore := baseline.GetRoute(endpoint.Asn, otherAsn)

		audited.SetDestinations(map[int]bool{otherAsn: true})
		audited.Evolve()
		levelBefore, auditedAsnBefore := audited.ApproximatePath(endpoint.Asn, otherAsn)
		linksNum--

		baseline.DeleteDestination(otherAsn)
		baseline.Evolve()
		audited.DeleteDestination(otherAsn)
		audited.Evolve()

		baselineSuccess, _, _ := baseline.RemoveEdge(endpoint.Asn, otherAsn)
		success, impactedArea, _ := audited.RemoveEdge(endpoint.Asn, otherAsn)

		impactedNum := len(impactedArea)

		if success {
			if !baselineSuccess {
				panic("Difference in graphs")
			}

			baseline.SetDestinations(map[int]bool{otherAsn: true})
			baseline.Evolve()
			// TODO: Should do the same with audited

			baselineAfter, baselineTypesAfter := baseline.GetRoute(endpoint.Asn, otherAsn)
			levelAfter, auditedAsnAfter := audited.ApproximatePath(endpoint.Asn, otherAsn)

			if baselineAfter == nil {
				// After the deletion, there is no path respecting GR rules in the original graph (only paths with valleys)
				continue
			}

			// Consider the sample only if it's successful
			s++

			record(
				u.Str(len(baselineBefore)),
				formatTypes(baselineTypesBefore),
				u.Str(levelBefore),
				formatAsnPath(auditedAsnBefore),
				u.Str(len(baselineAfter)),
				formatTypes(baselineTypesAfter),
				u.Str(levelAfter),
				formatAsnPath(auditedAsnAfter),
			)

		} else if !success && impactedNum > 0 {
			// Game over! The graph is no more a connected component
			// Start with fresh copies
			fmt.Printf("Starting from fresh graphs after %d samples (detected > 1 connected component)\n", s)
			baseline = baselineGraph.Copy()
			audited = auditedGraph.CopyAsTz()

			// Recount links
			linksNum = audited.CountLinks()
		}
	}

	stopRecording()
}

func deletionsRound(baseline AbstractGraph, audited AbstractGraph, round int, deletionProportion float64) bool {

	linksNum := audited.CountLinks()

	toDelete := int(float64(linksNum) * deletionProportion)

	fmt.Printf("Starting round #%d: deleting %d links\n", round, toDelete)

	for toDelete > 0 {
		// Choose a random link
		endpoint, linkIdx := RandomLink(audited, linksNum)
		otherAsn := endpoint.Links[linkIdx]
		// Here, 8 links are required at both endpoints to perform a link deletion
		for len(endpoint.Links) < 8 || len((*audited.GetNodes())[otherAsn].Links) < 8 {
			endpoint, linkIdx = RandomLink(audited, linksNum)
			otherAsn = endpoint.Links[linkIdx]
		}

		baselineSuccess, _, _ := baseline.RemoveEdge(endpoint.Asn, otherAsn)
		auditedSuccess, impactedArea, _ := audited.RemoveEdge(endpoint.Asn, otherAsn)

		impactedNum := len(impactedArea)

		if auditedSuccess {
			if !baselineSuccess {
				panic("Baseline and Audited graphs out of sync")
			}

			toDelete--
			linksNum--

		} else if !auditedSuccess && impactedNum > 0 {
			// Multiple connected components detected
			return false
		} else {
			// Something strange happened
			panic("Could not perform a legitimate deletion")
		}

	}

	return true
}

func chosenDeletionsRound(baseline AbstractGraph, audited AbstractGraph, deletionsList [][]int, slot int, rounds int) map[int]bool {

	startIdx := slot * len(deletionsList) / rounds
	var endIdx int

	if slot == rounds-1 {
		endIdx = len(deletionsList)
	} else {
		endIdx = (slot + 1) * len(deletionsList) / rounds
	}

	toDelete := endIdx - startIdx

	fmt.Printf("Starting round #%d: deleting %d links\n", slot, toDelete)

	disconnectedNodes := make(map[int]bool)

	for startIdx < endIdx {

		endA := deletionsList[startIdx][0]
		endB := deletionsList[startIdx][1]

		auditedSuccess, impactedArea, _ := audited.RemoveEdge(endA, endB)

		impactedNum := len(impactedArea)

		if auditedSuccess {
			baselineSuccess, _, _ := baseline.RemoveEdge(endA, endB)
			if !baselineSuccess {
				panic("Baseline and Audited graphs out of sync")
			}

		} else if !auditedSuccess && impactedNum > 0 {
			// Multiple connected components detected
			disconnectedNodes = u.Union(disconnectedNodes, impactedArea)
		} else {
			// Something strange happened
			auditedNodes := audited.GetNodes()
			fmt.Printf("Could not perform the deletion of %d (%d) -> %d (%d)\n", endA, len((*auditedNodes)[endA].Links), endB, len((*auditedNodes)[endB].Links))
		}

		startIdx++
	}

	return disconnectedNodes
}

// MeasureChosenDeletionsStretch computes the average and maximum increase in empirical stretch after having deleted
// a specific sequence of edges from the graph (distributed over 'rounds' rounds)
// If recording is active, for each round, the lengths and shapes of measured paths are saved to file
func MeasureChosenDeletionsStretch(baselineOriginal *AbstractGraph, auditedOriginal *AbstractGraph, rounds int, deletionsFilename string) (float64, float64) {

	// Conduct measurements on a copy of the graphs
	baseline := (*baselineOriginal).Copy()
	audited := (*auditedOriginal).Copy()

	var previousStretch float64
	var averageStretchIncrease float64
	var maxStretchIncrease float64

	deletionsList := loadEdgeDeletionsFile(deletionsFilename)

	perRoundSamples := 2000

	disconnectedNodes := make(map[int]bool)

	// 1 round is performed, since the round#0 is without deletions
	for r := 0; r <= rounds; r++ {

		record(
			u.Str(-r),
			u.Str(-r),
			u.Str(-r),
		)

		// Measure stretch
		stretchChannel := roundChannels{
			stretchContribution: make(chan float64, 1),
			maxContribution:     make(chan float64, 1),
			valleyContribution:  make(chan int, 1),
		}

		go stretchRound(baseline, audited, perRoundSamples, disconnectedNodes, stretchChannel)

		roundStretch := <-stretchChannel.stretchContribution / float64(perRoundSamples)
		roundStretchIncrease := roundStretch - previousStretch
		previousStretch = roundStretch

		averageStretchIncrease += roundStretchIncrease
		if roundStretchIncrease > maxStretchIncrease {
			maxStretchIncrease = roundStretchIncrease
		}

		fmt.Printf("	Measured %f increase in round stretch\n", roundStretchIncrease)

		if r != rounds {
			newlyDisconnected := chosenDeletionsRound(baseline, audited, deletionsList, r, rounds)
			disconnectedNodes = u.Union(disconnectedNodes, newlyDisconnected)
		}
	}

	averageStretchIncrease /= float64(rounds)

	stopRecording()

	(*baselineOriginal) = baseline
	(*auditedOriginal) = audited

	return averageStretchIncrease, maxStretchIncrease
}

// MeasureRandomDeletionsStretch computes the average and maximum increase in empirical stretch after having deleted
// a fraction 'deletionProportion' of edges from the graph (without creating multiple connected components)
// this operation is repeated ('rounds' - 1) times
// If recording is active, for each round, the lengths and shapes of measured paths are saved to file
func MeasureRandomDeletionsStretch(baselineOriginal *AbstractGraph, auditedOriginal *AbstractGraph, rounds int, deletionProportion float64) (float64, float64) {

	rand.Seed(time.Now().UnixNano())

	// Conduct measurements on a copy of the graphs
	baseline := (*baselineOriginal).Copy()
	audited := (*auditedOriginal).Copy()

	var previousStretch float64
	var averageStretchIncrease float64
	var maxStretchIncrease float64

	perRoundSamples := 1200

	for r := 0; r < rounds; r++ {

		// TODO: Hnadle this better
		// Mark the beginning of a round
		record(
			u.Str(-r),
			u.Str(-r),
			u.Str(-r),
		)

		// Measure stretch
		stretchChannel := roundChannels{
			stretchContribution: make(chan float64, 1),
			maxContribution:     make(chan float64, 1),
			valleyContribution:  make(chan int, 1),
		}

		go stretchRound(baseline, audited, perRoundSamples, map[int]bool{}, stretchChannel)

		roundStretch := <-stretchChannel.stretchContribution / float64(perRoundSamples)
		roundStretchIncrease := roundStretch - previousStretch
		previousStretch = roundStretch

		averageStretchIncrease += roundStretchIncrease
		if roundStretchIncrease > maxStretchIncrease {
			maxStretchIncrease = roundStretchIncrease
		}

		fmt.Printf("	Measured %f increase in round stretch\n", roundStretchIncrease)

		if r != rounds-1 {
			safeBaselineCopy := baseline.Copy()
			safeAuditedCopy := audited.Copy()

			for !deletionsRound(baseline, audited, r, deletionProportion) {
				// Try again
				fmt.Println("Obtained 2 connected components, retrying from safe copy ...")
				baseline = safeBaselineCopy.Copy()
				audited = safeAuditedCopy.Copy()
			}
		}
	}

	isRecording, logPath := GetOutputDir()
	if isRecording {
		GraphStructure(*audited.GetNodes()).WriteStructureToCsv(fmt.Sprintf("%smissing-edges-%dx%.3f.csv", logPath, rounds, deletionProportion))
	}

	averageStretchIncrease /= float64(rounds)

	stopRecording()

	(*baselineOriginal) = baseline
	(*auditedOriginal) = audited

	return averageStretchIncrease, maxStretchIncrease
}

// MeasureEndpointsDegrees returns the list of the degrees of endpoints
// of each edge in the graph
func MeasureEndpointsDegrees(graph AbstractGraph) {
	nodes := *graph.GetNodes()

	for _, n := range nodes {
		for _, l := range n.Links {
			record(
				u.Str(len(n.Links)),
				u.Str(len(nodes[l].Links)),
			)
		}
	}

	stopRecording()
}
