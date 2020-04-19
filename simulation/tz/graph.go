package tz

import (
	"fmt"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

// TODO: Debug only, add an attribute in node struct
const edgeWeight int64 = 1

// TODO: Change that to proper INT64 MAX
const int64Max int64 = 1000000

// Graph represents the AS graph
type Graph struct {
	Nodes     map[int]*Node
	K         int
	Landmarks Landmarks
	Witnesses map[int]*DijkstraGraph
	Bunches   Clusters
}

// InitGraph returns a fresh graph
func InitGraph() Graph {
	return Graph{
		Nodes:     make(map[int]*Node),
		Landmarks: make(Landmarks),
		Witnesses: make(map[int]*DijkstraGraph),
		Bunches:   make(Clusters),
	}
}

// GetNodes returns a pointer to the map of Nodes
func (g *Graph) GetNodes() *map[int]*Node {
	return &g.Nodes
}

func unsupportedOperation(callee string) {
	panic(callee + "() is not supported by TZ algorithm")
}

func (g *Graph) kIsValid() {
	if g.K < 1 {
		panic("Illegal k value for graph")
	}
}

// ElectLandmarks chooses the samples A_i (0 <= i < k) of available nodes
func (g *Graph) ElectLandmarks(selectionStrategy int) {
	if g.K < 1 {
		panic("The number of landmark sets must be >= 1, got " + u.Str(g.K))
	}

	switch selectionStrategy {
	case RandomStrategy:
		g.randomStrategy()
	case SplineStrategy:
		g.splineStrategy()
	case HarmonicStrategy:
		g.harmonicStrategy()
	case ImmunityStrategy:
		g.immunityStrategy()
	}
}

// Copy returns a duplicate of Landmarks
func (l *Landmarks) Copy() *Landmarks {
	copyLandmarks := make(Landmarks)

	for r, ld := range *l {
		copyLandmarks[r] = make(map[*Node]bool)
		for n := range ld {
			copyLandmarks[r][n] = true
		}
	}

	return &copyLandmarks
}

func (g *Graph) calculateWitnessForRound(round int) *DijkstraGraph {
	dijkstraGraph := make(DijkstraGraph)

	frontier := Frontier{
		Zones:       make(map[int64]map[int]*dijkstraNode),
		MinDistance: 0,
	}
	frontier.Zones[0] = make(map[int]*dijkstraNode)

	var frontierPopulation int = 0

	for entryPoint := range g.Landmarks[round] {
		// Initialize entry points
		tempNode := dijkstraNode{
			reference: entryPoint.Asn,
			distance:  0,
			parent:    entryPoint,
			nextHop:   entryPoint,
		}
		dijkstraGraph[entryPoint.Asn] = &tempNode
		frontier.Zones[0][tempNode.reference] = &tempNode
		frontierPopulation++
	}

	dijkstraGraph.runDijkstra(&g.Nodes, &frontier, frontierPopulation)

	return &dijkstraGraph
}

// Preprocess fills the data needed to answer queries
func (g *Graph) Preprocess() {

	g.kIsValid()

	// All the nodes in A_(k-1) belong to every bunch, since the distance to A_k is +inf
	infDijkstraGraph := make(DijkstraGraph)
	for v := range g.Nodes {
		tempDijkstra := dijkstraNode{
			reference: g.Nodes[v].Asn,
			distance:  int64Max,
			parent:    g.Nodes[v],
			nextHop:   g.Nodes[v],
		}
		infDijkstraGraph[g.Nodes[v].Asn] = &tempDijkstra
	}

	g.Witnesses[g.K] = &infDijkstraGraph

	clusters := make(Clusters)

	for i := g.K - 1; i >= 0; i-- {

		fmt.Printf("Starting round... %d\n", i)

		clusters.calculateClustersForRound(&g.Nodes, i, &g.Landmarks, g.Witnesses[i+1])
		g.Witnesses[i] = g.calculateWitnessForRound(i)

		g.enforceAsteriskRule(i)
	}

	for asn := range g.Nodes {
		g.Bunches[asn] = make(map[int]*dijkstraNode)

		for q := range clusters {
			if cl, ok := clusters[q][asn]; ok {
				g.Bunches[asn][q] = cl
			}
		}
	}
}

// Enforce Asterisk rule (same witness if same distance)
func (g *Graph) enforceAsteriskRule(round int) {
	for asn := range *g.Witnesses[round] {
		prevDijkstraNode, exists := (*g.Witnesses[round+1])[asn]
		if exists && (*g.Witnesses[round])[asn].distance == prevDijkstraNode.distance {
			(*g.Witnesses[round])[asn].parent = prevDijkstraNode.parent
			(*g.Witnesses[round])[asn].nextHop = prevDijkstraNode.nextHop
		}
	}
}

func (g *Graph) expandPath(a int, w int, b int, round int) []int {
	hopsAtoW := make([]int, 0, 4)

	// From a to w (landmark)
	prev := -1
	for cursor := a; cursor != w; cursor = (*g.Witnesses[round])[cursor].nextHop.Asn {
		if prev == cursor {
			panic("Wrong direction taken in path reconstruction")
			//return hops
		}
		prev = cursor
		hopsAtoW = append(hopsAtoW, cursor)
	}

	hopsAtoW = append(hopsAtoW, w)

	// From b to w
	hopsBtoW := make([]int, 0, 4)
	for ; b != w; b = g.Bunches[b][w].nextHop.Asn {
		hopsBtoW = append(hopsBtoW, b)
	}

	hopsAtoW, hopsBtoW = trimPrefix(hopsAtoW, hopsBtoW)

	// Choose which part to reverse according to the round parity
	var radix *[]int
	var toReverse *[]int
	if round%2 == 0 {
		radix = &hopsAtoW
		toReverse = &hopsBtoW
	} else {
		radix = &hopsBtoW
		toReverse = &hopsAtoW
	}

	// Reverse that part of the path
	for idx := len(*toReverse) - 1; idx >= 0; idx-- {
		(*radix) = append((*radix), (*toReverse)[idx])
	}

	return *radix
}

func (g *Graph) printPath(a int, w int, b int, round int) {
	hops := g.expandPath(a, w, b, round)

	for id := 0; id < len(hops)-1; id++ {
		fmt.Printf("%d > ", hops[id])
	}

	fmt.Printf("%d\n", hops[len(hops)-1])
}

// ApproximatePath compute an approximation of the path from 'from' to 'to'
// It returns (an estimation of) the distance and a path
func (g *Graph) ApproximatePath(from int, to int) (int64, []int) {

	var w int = from
	var i int = 0

	for {

		if w2to, ok := g.Bunches[to][w]; ok {
			from2w := (*g.Witnesses[i])[from].distance

			return from2w + w2to.distance, g.expandPath(from, w, to, i)
		}

		// TODO: Debug check
		if i == g.K {
			panic("Calculated a wrong distance approximation")
		}

		i++
		temp := to
		to = from
		from = temp
		w = (*g.Witnesses[i])[from].parent.Asn

		//sh.Overwrite(fmt.Sprintf("Using level %s%d%s landmarks: from:%d, to:%d, neighbor:%d\n", shell.Red, i, shell.Clear, from, to, w))
	}
}

// Copy returns a duplicate of the Graph
func (g *Graph) Copy() AbstractGraph {
	// TODO: Think about that. Up to now there is no need to copy anything
	copyGraph := Graph{
		Nodes:     make(map[int]*Node),
		K:         g.K,
		Landmarks: nil,
		Witnesses: make(map[int]*DijkstraGraph),
		Bunches:   make(Clusters),
	}

	for k, v := range g.Nodes {
		copyGraph.Nodes[k] = v
	}

	copyGraph.Landmarks = *g.Landmarks.Copy()

	for w, v := range g.Witnesses {
		copyGraph.Witnesses[w] = v.Copy()
	}

	copyGraph.Bunches = *g.Bunches.Copy()

	return &copyGraph
}

// Performs the union of two sets
func union(acculator map[int]bool, toAdd map[int]bool) map[int]bool {
	if acculator == nil {
		panic("Cannot perform union on nil accumulator")
	}
	if toAdd != nil {
		for e := range toAdd {
			acculator[e] = true
		}
	}
	return acculator
}

// RemoveEdge deletes an edge from the graph and update the
// relevant data structures
// returns true if the deletion was successful
// returns the number of nodes impacted by the update
// (false, 0) : the deletion could not be performed
// (false, >0): the deletion was performed but the graph is NO MORE 1 connected component
func (g *Graph) RemoveEdge(aAsn int, bAsn int) (bool, int) {

	a, aOk := g.Nodes[aAsn]
	b, bOk := g.Nodes[bAsn]

	if !(aOk && bOk) {
		return false, 0
	}

	if len(a.Links) <= 1 || len(b.Links) <= 1 {
		return false, 0
	}

	if !(a.DeleteLink(b) && b.DeleteLink(a)) {
		panic("Link deletion unsuccessful! Corrupted graph")
	}

	// TODO: Here, take into account the messages sent all the way back
	// to the landmarks (??)

	impactedArea := make(map[int]bool)

	// Fix Witnesses
	for round := g.K - 1; round >= 0; round-- {
		fixWitFromA := g.fixWitnessByRound(a, b, round)
		fixWitFromB := g.fixWitnessByRound(b, a, round)

		impactedArea = union(impactedArea, fixWitFromA)
		impactedArea = union(impactedArea, fixWitFromB)

		// Enforce asterisk rule only when witnesses are coherent
		g.enforceAsteriskRule(round)
	}

	fixBunFromA := g.fixBunches(a, b)
	fixBunFromB := g.fixBunches(b, a)

	impactedArea = union(impactedArea, fixBunFromA)
	impactedArea = union(impactedArea, fixBunFromB)

	// Check that the graph is still connected
	for ia := range impactedArea {
		if len(g.Bunches[ia]) < len(g.Landmarks[g.K-1]) {
			return false, len(impactedArea)
		}
	}

	return true, len(impactedArea)
}

// Remove from the bunch of 'target' the set of routes to 'unavailable' passing through 'nextHop'
// returns the set of invalidated destinations
// TODO: Reduce scope of argument + generalize function to multiple level of landmarks
func (g *Graph) purgeFromBunch(targetAsn int, unavailable map[int]*Node, nextHopAsn int) map[int]*Node {
	toInvalidate := make(map[int]*Node)

	// Collect destinations to invalidate
	for e, d := range g.Bunches[targetAsn] {
		if d.nextHop.Asn == nextHopAsn {
			if _, isUnreachable := unavailable[e]; isUnreachable {
				toInvalidate[e] = g.Nodes[e]
			}
		}
	}

	// Update the bunch
	for e := range toInvalidate {
		delete(g.Bunches[targetAsn], e)
	}

	return toInvalidate
}

// fixBunches restores the correctness of bunches
// returns the set of asn touched by the update
func (g *Graph) fixBunches(endpoint *Node, brokenLink *Node) map[int]bool {

	unavailable := make(map[int]*Node)

	// Fill unavailable
	for dest, dij := range g.Bunches[endpoint.Asn] {
		if dij.nextHop.Asn == brokenLink.Asn {
			unavailable[dest] = g.Nodes[dest]
		}
	}

	brokenTopLevel := g.Landmarks.filterByLevel(unavailable, g.K-1)

	dijkstraByLandmark := make(map[int]*DijkstraGraph)
	frontierByLandmark := make(map[int]*Frontier)
	populationByLandmark := make(map[int]int)
	toUpdateByLandmark := make(map[int]*map[int]*Node)
	for tl := range brokenTopLevel {
		frontierByLandmark[tl] = &Frontier{
			Zones:       make(map[int64]map[int]*dijkstraNode),
			MinDistance: int64Max,
		}
		populationByLandmark[tl] = 0

		tempGraph := make(DijkstraGraph)
		dijkstraByLandmark[tl] = &tempGraph

		tempUpdate := make(map[int]*Node)
		tempUpdate[endpoint.Asn] = endpoint
		toUpdateByLandmark[tl] = &tempUpdate
	}

	// For each asn, addedInRound stores the invalidated destinations
	addedInRound := make(map[int]map[int]*Node)
	addedInRound[endpoint.Asn] = unavailable

	g.purgeFromBunch(endpoint.Asn, unavailable, brokenLink.Asn)

	for len(addedInRound) > 0 {
		nextAdded := make(map[int]map[int]*Node)
		for a, deletedFromA := range addedInRound {
			for _, n := range g.Nodes[a].Links {
				revokedDests := g.purgeFromBunch(n, deletedFromA, a)

				// Check if some destinations were revoked
				if len(revokedDests) > 0 {
					nextAdded[n] = revokedDests
				}

				neededAtN := g.Landmarks.filterByLevel(revokedDests, g.K-1)
				for toUp := range neededAtN {
					(*toUpdateByLandmark[toUp])[n] = g.Nodes[n]
				}

				// Check if some of the missing top-level landmarks are found
				topLevelNeeded := g.Landmarks.filterByLevel(deletedFromA, g.K-1)
				for tl := range topLevelNeeded {
					if _, isPresent := g.Bunches[n][tl]; isPresent {
						// Node n has a valid path to tl
						tempDij := g.Bunches[n][tl].Copy()
						(*dijkstraByLandmark[tl])[n] = tempDij
						if frontierByLandmark[tl].addToFrontier(tempDij) {
							populationByLandmark[tl]++
						}
						(*toUpdateByLandmark[tl])[n] = g.Nodes[n]
					}
				}
			}
		}
		addedInRound = nextAdded
	}

	// Audit
	impactedAsn := make(map[int]bool)
	for tl := range brokenTopLevel {
		for e := range *toUpdateByLandmark[tl] {
			impactedAsn[e] = true
		}
	}

	// Execute Dijkstra for each top-level landmark
	for tl := range brokenTopLevel {
		dijkstraByLandmark[tl].runDijkstra(toUpdateByLandmark[tl], frontierByLandmark[tl], populationByLandmark[tl])

		for nd, toLandmark := range *dijkstraByLandmark[tl] {
			g.Bunches[nd][toLandmark.parent.Asn] = toLandmark
		}
	}

	return impactedAsn
}

// Restore the correctness of witnesses for a given round
// return the set of asn needed to complete the operation
func (g *Graph) fixWitnessByRound(endpoint *Node, brokenLink *Node, round int) map[int]bool {

	// Check if the witness was reached through the broken link
	if (*g.Witnesses[round])[endpoint.Asn].nextHop.Asn != brokenLink.Asn {
		return nil
	}

	toUpdateZone := make(map[int]*Node)
	toUpdateZone[endpoint.Asn] = endpoint
	(*g.Witnesses[round])[endpoint.Asn].distance = int64Max

	var addedInRound map[int]bool
	addedInRound = make(map[int]bool)

	addedInRound[endpoint.Asn] = true

	frontier := Frontier{
		Zones:       make(map[int64]map[int]*dijkstraNode),
		MinDistance: int64Max,
	}

	frontierPopulation := 0

	// Find the Nodes that must be updated
	for len(addedInRound) > 0 {
		nextAdded := make(map[int]bool)
		for a := range addedInRound {
			for _, n := range g.Nodes[a].Links {
				toUpdateZone[n] = g.Nodes[n]
				witness := (*g.Witnesses[round])[n]
				if witness.nextHop.Asn == a {
					nextAdded[n] = true
					(*g.Witnesses[round])[n].distance = int64Max
				} else if witness.distance < int64Max {
					dijNode := (*g.Witnesses[round])[n]

					// The node could already be in the frontier
					if frontier.addToFrontier(dijNode) {
						frontierPopulation++
					}
				}
			}
		}
		addedInRound = nextAdded
	}

	// Audit
	impactedAsn := make(map[int]bool)
	for asn := range toUpdateZone {
		impactedAsn[asn] = true
	}

	g.Witnesses[round].runDijkstra(&toUpdateZone, &frontier, frontierPopulation)

	return impactedAsn
}

// Evolve brings the graph to a stable state
func (g *Graph) Evolve() int {
	// TODO: Here too, unsupportedOperation
	return 0
}

// SetDestinations updates the speakers according to the set of chosen destinations
func (g *Graph) SetDestinations(dest map[int]bool) {
	// TODO: This is called by default by auditor
	//unsupportedOperation("SetDestinations")
}

func (g *Graph) DeleteDestination(dest int) {
	// TODO: This is called by default by auditor
	// unsupportedOperation("DeleteDestination")
}

// GetRoute returns a path (if it exists) from an origin to a destination along with the types of links used
// The first array is 1 ELEMENT LONGER than the second
func (g *Graph) GetRoute(originAsn int, destinationAsn int) ([]*Node, []int) {

	_, okOrigin := g.Nodes[originAsn]
	_, okDestination := g.Nodes[destinationAsn]

	if !(okOrigin && okDestination) {
		return nil, nil
	}

	// TODO: Add support for link type
	_, hops := g.ApproximatePath(originAsn, destinationAsn)

	nodeHops := make([]*Node, 0, len(hops))
	nodeTypes := make([]int, 0, len(hops)-1)
	for idx, h := range hops {
		nodeHops = append(nodeHops, g.Nodes[h])
		if idx > 0 {
			nodeTypes = append(nodeTypes, g.Nodes[hops[idx-1]].GetNeighborType(g.Nodes[h]))
		}
	}

	return nodeHops, nodeTypes
}
