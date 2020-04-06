package main

import (
	"fmt"
	"math"
	"math/rand"

	"../shell"
	"../u"
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

func unsupportedOperation(callee string) {
	panic(callee + "() is not supported by TZ algorithm")
}

func (g *Graph) kIsValid() {
	if g.K < 1 {
		panic("Illegal k value for graph")
	}
}

// Landmarks models the set of samples A_i (0 <= i < k)
type Landmarks map[int]map[*Node]bool

// ElectLandmarks chooses the samples A_i (0 <= i < k) of available nodes
func (g *Graph) ElectLandmarks(k int) {
	if k < 1 {
		panic("The number of landmark sets must be >= 1, got " + u.Str(k))
	}

	// Put all the nodes in A_0
	g.Landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		g.Landmarks[0][v] = true
	}

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1/float64(k))

	for i := 1; i < k; i++ {
		g.Landmarks[i] = make(map[*Node]bool)
		for key := range g.Landmarks[i-1] {
			extraction := rand.Float64()
			if extraction <= selProbability {
				g.Landmarks[i][key] = true
			}
		}
	}

	g.Landmarks[k] = nil
}

func (g *Graph) calculateWitnessForRound(round int) *DijkstraGraph {
	dijkstraGraph := make(DijkstraGraph)

	frontier := Frontier{
		Zones:       make(map[int64]map[*dijkstraNode]bool),
		MinDistance: 0,
	}
	frontier.Zones[0] = make(map[*dijkstraNode]bool)

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
		frontier.Zones[0][&tempNode] = true
		frontierPopulation++
	}

	dijkstraGraph.runDijkstra(&g.Nodes, &frontier, frontierPopulation)

	return &dijkstraGraph
}

// Evolve preprocesses the graph
func (g *Graph) Evolve() {

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

		// Enforce Asterisk rule
		for asn := range *g.Witnesses[i] {
			prevDijkstraNode, exists := (*g.Witnesses[i+1])[asn]
			if exists && (*g.Witnesses[i])[asn].distance == prevDijkstraNode.distance {
				(*g.Witnesses[i])[asn].parent = prevDijkstraNode.parent
				(*g.Witnesses[i])[asn].nextHop = prevDijkstraNode.nextHop
			}
		}
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

		sh.Overwrite(fmt.Sprintf("Using level %s%d%s landmarks: from:%d, to:%d, neighbor:%d\n", shell.Red, i, shell.Clear, from, to, w))
	}
}

// SetDestinations updates the speakers according to the set of chosen destinations
func (g *Graph) SetDestinations(dest map[int]bool) {
	unsupportedOperation("SetDestinations")
}

// GetRoute returns a path (if it exists) from an origin to a destination along with the types of links used
// The first array is 1 ELEMENT LONGER than the second
func (g *Graph) GetRoute(originAsn int, destinationAsn int) ([]*Node, []int) {
	// TODO: Add support for link type
	_, hops := g.ApproximatePath(originAsn, destinationAsn)

	nodeHops := make([]*Node, 0, len(hops))
	nodeTypes := make([]int, 0, len(hops)-1)
	for _, h := range hops {
		nodeHops = append(nodeHops, g.Nodes[h])
		nodeTypes = append(nodeTypes, 100)
	}

	return nodeHops, nodeTypes
}
