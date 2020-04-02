package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"../u"
)

// TODO: Debug only, add an attribute in node struct
const edgeWeight int64 = 1

// TODO: Change that to proper INT64 MAX
const int64Max int64 = 1000000

// Speaker represents a TZ speaker
type Speaker struct {
	Witnesses map[int]*Node
	Distances map[int]int64
	Bunches   map[*Node](map[*Node]int)

	// Old attributes
	Fresh        []bool
	Destinations []*Node
	NextHop      []*Node
	Length       []int
}

// Landmarks models the set of samples A_i (0 <= i < k)
type Landmarks map[int]map[*Node]bool

func (s *Speaker) String(n *Node) string {
	// TODO: Refine it
	var sb strings.Builder
	for idx, dest := range s.Destinations {
		sb.WriteString("	")
		nextHopType := s.getNextHopType(n, idx)

		sb.WriteString("(")
		sb.WriteString(u.Str(s.Length[idx]))
		sb.WriteString(") ")

		switch s.Length[idx] {
		case 0:
		case 1:
			if dest.Asn != s.NextHop[idx].Asn {
				panic("Destination != NextHop in length=1 paths")
			}
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
		case 2:
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
			sb.WriteString(u.Str(s.NextHop[idx].Asn))
			sb.WriteString(" > ")
		default:
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
			sb.WriteString(u.Str(s.NextHop[idx].Asn))
			sb.WriteString(" > ")
			sb.WriteString("...")
			sb.WriteString(" > ")
		}

		sb.WriteString(u.Str(dest.Asn))
		sb.WriteString("\n")
	}
	return sb.String()
}

// InitSpeaker initializes the Speaker associated with a certain Node
func InitSpeaker(node *Node) *Speaker {
	// TODO: Initialize new attributes too
	speaker := Speaker{Fresh: nil, Destinations: nil, NextHop: nil, Length: nil}
	return &speaker
}

// ElectLandmarks chooses the samples A_i (0 <= i < k) of available nodes
func (g *Graph) ElectLandmarks(k int) *Landmarks {
	if k < 1 {
		panic("The number of landmark sets must be >= 1, got " + u.Str(k))
	}

	var landmarks Landmarks = make(Landmarks)

	// Put all the nodes in A_0
	landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		landmarks[0][v] = true
	}

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1/float64(k))

	for i := 1; i < k; i++ {
		landmarks[i] = make(map[*Node]bool)
		for key := range landmarks[i-1] {
			extraction := rand.Float64()
			if extraction <= selProbability {
				landmarks[i][key] = true
			}
		}
	}

	landmarks[k] = nil

	return &landmarks
}

type dijkstraNode struct {
	reference int
	distance  int64
	parent    *Node
	nextHop   *Node
}

func (d *dijkstraNode) String() string {
	return "<" + u.Str(d.reference) + "= " + u.Str(d.parent.Asn) + "..." + u.Str(d.nextHop.Asn) + "->" + "(" + u.Str64(d.distance) + ")>"
}

// DijkstraGraph contains nearest landmark information
type DijkstraGraph map[int]*dijkstraNode

// Serialize produces a representation of the DijkstraGraph suitable to be saved to file
func (d *DijkstraGraph) Serialize(index int) [][]string {
	rows := make([][]string, 0, len(*d))

	for key, val := range *d {
		rows = append(rows, []string{u.Str(index), u.Str(key), u.Str64(val.distance), u.Str(val.parent.Asn), u.Str(val.nextHop.Asn)})
	}

	return rows
}

// Frontier makes easy to retrieve closest dijkstra nodes
type Frontier struct {
	Zones       map[int64]map[*dijkstraNode]bool
	MinDistance int64
}

func (f *Frontier) String() string {
	var sb strings.Builder
	sb.WriteString("Frontier (distance " + u.Str64(f.MinDistance) + "):\n")
	for c := range f.Zones[f.MinDistance] {
		sb.WriteString("	")
		sb.WriteString(u.Str(c.reference))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (f *Frontier) printFrontier() {
	fmt.Printf("Frontier at distance: %d\n", f.MinDistance)
	fmt.Println(f.Zones)
}

// Clusters is a set of sets of Asn
type Clusters map[int]map[int]int64

// Serialize implements the interface Serializable for *Clusters
func (c *Clusters) Serialize(index int) [][]string {
	rows := make([][]string, 0, len(*c))

	for key, value := range *c {
		for asn, dist := range value {
			rows = append(rows, []string{u.Str(key), u.Str(asn), u.Str64(dist)})
		}
	}

	return rows
}

func (f *Frontier) getSomeNode(distance int64) *dijkstraNode {
	for k := range f.Zones[distance] {
		return k
	}

	panic("The minimum distance Zone of the frontier was empty")
}

func (f *Frontier) getFromClosest() *dijkstraNode {

	elem := f.getSomeNode(f.MinDistance)

	f.deleteFromFrontier(elem)

	return elem

}

// If oldDistance is < 0, the node has just been discovered
func (f *Frontier) addToFrontier(n *dijkstraNode) {
	_, ok := f.Zones[n.distance]
	if !ok {
		f.Zones[n.distance] = make(map[*dijkstraNode]bool)
	}
	f.Zones[n.distance][n] = true

	if n.distance < f.MinDistance {
		f.MinDistance = n.distance
	}
}

func (f *Frontier) deleteFromFrontier(n *dijkstraNode) {
	delete(f.Zones[f.MinDistance], n)

	if len(f.Zones[f.MinDistance]) == 0 {
		delete(f.Zones, f.MinDistance)

		var newMin int64 = int64Max
		// Search new minDistance
		for k := range f.Zones {
			if k < newMin {
				newMin = k
			}
		}
		f.MinDistance = newMin
	}
}

func (f *Frontier) expandFromNode(g *Graph, dijkstraGraph *DijkstraGraph, n *dijkstraNode) int {
	var discoveredNodes int = 0

	for _, neighbor := range g.Nodes[n.reference].Links {
		d, exists := (*dijkstraGraph)[neighbor]
		updatedDistance := n.distance + edgeWeight
		if exists {
			// Relax edge if needed
			if d.distance > updatedDistance {
				f.deleteFromFrontier(d)
				d.distance = updatedDistance
				d.parent = n.parent
				d.nextHop = g.Nodes[n.reference]
				f.addToFrontier(d)
			}
		} else {
			tempNode := dijkstraNode{
				reference: neighbor,
				distance:  updatedDistance,
				parent:    n.parent,
				nextHop:   g.Nodes[n.reference],
			}
			(*dijkstraGraph)[neighbor] = &tempNode
			f.addToFrontier(&tempNode)
			discoveredNodes++
		}
	}

	return discoveredNodes
}

func (d *DijkstraGraph) runDijkstra(g *Graph, frontier *Frontier, frontierPopulation int) {
	for frontierPopulation > 0 {
		expandFrom := frontier.getFromClosest()
		frontierPopulation--
		frontierPopulation += frontier.expandFromNode(g, d, expandFrom)
	}
}

func (g *Graph) calculateWitnessForRound(round int, l *Landmarks) *DijkstraGraph {
	dijkstraGraph := make(DijkstraGraph)

	frontier := Frontier{
		Zones:       make(map[int64]map[*dijkstraNode]bool),
		MinDistance: 0,
	}
	frontier.Zones[0] = make(map[*dijkstraNode]bool)

	var frontierPopulation int = 0

	for entryPoint := range (*l)[round] {
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

	dijkstraGraph.runDijkstra(g, &frontier, frontierPopulation)

	return &dijkstraGraph
}

func (c *Clusters) calculateClustersForRound(g *Graph, k int, l *Landmarks, prevRound *DijkstraGraph) {
	for w := range (*l)[k] {
		if _, ok := (*l)[k+1][w]; !ok {
			// w is in the set difference A_(k)\A_(k+1)
			wClusterGraph := make(DijkstraGraph)

			// Initialize Dijkstra with the source
			source := dijkstraNode{
				reference: w.Asn,
				distance:  0,
				parent:    w,
				nextHop:   w,
			}
			wClusterGraph[w.Asn] = &source

			clusterFrontier := Frontier{
				Zones:       make(map[int64]map[*dijkstraNode]bool),
				MinDistance: 0,
			}
			clusterFrontier.Zones[0] = map[*dijkstraNode]bool{&source: true}

			wClusterGraph.runDijkstra(g, &clusterFrontier, 1)

			// Create cluster for w
			(*c)[w.Asn] = make(map[int]int64)
			for nd := range wClusterGraph {
				if wClusterGraph[nd].distance < (*prevRound)[nd].distance {
					(*c)[w.Asn][nd] = wClusterGraph[nd].distance
				}
			}
		}
	}
}

// CalculateWitnesses finds the closest member of landmarks to each vertex
func (g *Graph) CalculateWitnesses(k int, l *Landmarks) (*map[int]*DijkstraGraph, *Clusters) {

	witnessesByRound := make(map[int]*DijkstraGraph)

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

	witnessesByRound[k] = &infDijkstraGraph

	clusters := make(Clusters)

	for i := k - 1; i >= 0; i-- {

		fmt.Printf("Starting round... %d\n", i)

		clusters.calculateClustersForRound(g, i, l, witnessesByRound[i+1])
		witnessesByRound[i] = g.calculateWitnessForRound(i, l)

		// Enforce Asterisk rule
		for asn := range *witnessesByRound[i] {
			prevDijkstraNode, exists := (*witnessesByRound[i+1])[asn]
			if exists && (*witnessesByRound[i])[asn].distance == prevDijkstraNode.distance {
				(*witnessesByRound[i])[asn].parent = prevDijkstraNode.parent
			}
		}
	}

	bunches := make(Clusters)

	for asn := range g.Nodes {
		bunches[asn] = make(map[int]int64)

		for q := range clusters {
			if dist, ok := clusters[q][asn]; ok {
				bunches[asn][q] = dist
			}
		}
	}

	return &witnessesByRound, &bunches
}

func expandPath(a int, b int, round int, witnesses *map[int]*DijkstraGraph) []int {
	hops := make([]int, 0, 4)

	for cursor := a; cursor != b; cursor = (*(*witnesses)[round])[cursor].nextHop.Asn {
		hops = append(hops, cursor)
	}

	return hops
}

func printPath(from int, to int, w int, round int, witnesses *map[int]*DijkstraGraph) {
	hopsFromW := expandPath(from, w, round, witnesses)

	fmt.Println(hopsFromW)
	fmt.Println(u.Str(w) + " ... " + u.Str(to))
}

// ApproximateDistance compute an approximation of the distance from 'from' to 'to'
func (g *Graph) ApproximateDistance(k int, from int, to int, witnesses *map[int]*DijkstraGraph, bunches *Clusters) int64 {

	var w int = from
	var i int = 0

	for {

		if w2to, ok := (*bunches)[to][w]; ok {
			from2w := (*(*witnesses)[i])[from].distance
			printPath(from, to, w, i, witnesses)
			return from2w + w2to
		}

		// TODO: Debug check
		if i == k {
			panic("Calculated a wrong distance approximation")
		}

		i++
		temp := to
		to = from
		from = temp
		w = (*(*witnesses)[i])[from].parent.Asn

		fmt.Printf("%d, u:%d, v:%d, w:%d\n", i, from, to, w)
	}
}

// CODE FROM BEFORE

func (s *Speaker) hasRoute(dest *Node) int {
	for i, d := range s.Destinations {
		if d == dest {
			return i
		}
	}
	return -1
}

func (s *Speaker) heardFrom(destIndex int, neighbor *Node) bool {
	return s.NextHop[destIndex].Asn == neighbor.Asn
}

func (s *Speaker) addDestination(currNode *Node, dest *Node, nextHop *Node, length int) bool {
	s.Fresh = append(s.Fresh, true)
	s.Destinations = append(s.Destinations, dest)
	s.NextHop = append(s.NextHop, nextHop)
	s.Length = append(s.Length, length+1)
	return true
}

func (s *Speaker) updateDestination(currNode *Node, routeIndex int, dest *Node, nextHop *Node, length int) bool {
	// Compare with current path
	oldNextType := s.getNextHopType(currNode, routeIndex)
	newNextType := currNode.getNeighborType(nextHop)

	// Applies: customer < peer < provider (smaller is better)
	if (newNextType < oldNextType) ||
		(oldNextType == newNextType && length+1 < s.Length[routeIndex]) {
		// Higher preference or same preference, but the new route is shorter
		s.Fresh[routeIndex] = true
		s.Length[routeIndex] = length + 1
		s.NextHop[routeIndex] = nextHop
		return true
	}

	return false
}

// This function assumes, for performance reasons, that the node n does NOT appear in path
func (s *Speaker) getNextHopType(n *Node, destIndex int) int {
	if s.Length[destIndex] == 0 {
		// The node *n is the origin, return ToCustomer
		return ToCustomer
	}
	return n.getNeighborType(s.NextHop[destIndex])
}

func (s *Speaker) advertise(neighborNode *Node, destination *Node, nextHop *Node, length int) bool {
	routeNum := s.hasRoute(destination)
	if routeNum < 0 {
		// The neighbor speaker does not have the destination yet
		return s.addDestination(neighborNode, destination, nextHop, length)
	}
	// The neighbor has the destination, check which one is better
	return s.updateDestination(neighborNode, routeNum, destination, nextHop, length)
}
