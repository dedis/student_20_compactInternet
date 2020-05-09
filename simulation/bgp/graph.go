package bgp

import (
	"fmt"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

// Graph represents the AS graph
type Graph struct {
	Nodes     map[int]*Node
	Speakers  map[int]*Speaker
	unstable  map[*Node]bool
	remaining int
}

func InitGraph() Graph {
	return Graph{
		Nodes:     make(map[int]*Node),
		Speakers:  make(map[int]*Speaker),
		unstable:  make(map[*Node]bool),
		remaining: 0,
	}
}

// GetNodes returns a pointer to the map of Nodes
func (g *Graph) GetNodes() *map[int]*Node {
	return &g.Nodes
}

// CountLinks returns the total number of links found in the graph
func (g *Graph) CountLinks() int {
	counter := 0

	for _, n := range g.Nodes {
		counter += len(n.Links)
	}

	return counter / 2
}

// SetDestinations updates the speakers according to the set of chosen destinations
func (g *Graph) SetDestinations(dest map[int]bool) {
	for asn := range dest {
		if g.Speakers[asn].hasRoute(g.Nodes[asn]) < 0 {
			g.Speakers[asn].Fresh = append(g.Speakers[asn].Fresh, true)
			g.Speakers[asn].Destinations = append(g.Speakers[asn].Destinations, g.Nodes[asn])
			g.Speakers[asn].NextHop = append(g.Speakers[asn].NextHop, g.Nodes[asn])
			g.Speakers[asn].Length = append(g.Speakers[asn].Length, 0)
			g.setUnstable(g.Nodes[asn])
		}
	}
}

// DeleteDestination removes the destination from every speaker
func (g *Graph) DeleteDestination(dest int) {
	for n := range g.Nodes {
		g.Speakers[n].deleteRoute(g.Nodes[dest])
	}
}

func (g *Graph) setUnstable(node *Node) {
	_, pr := g.unstable[node]
	if !pr {
		g.remaining++
		g.unstable[node] = true
	}
}

func (g *Graph) setStable(node *Node) {
	_, pr := g.unstable[node]
	if pr {
		g.remaining--
		delete(g.unstable, node)
	}
}

// Activate evolves the status of a speaker
func (g *Graph) Activate(nodeIndex int) int {

	nd := g.Nodes[nodeIndex]
	sp := g.Speakers[nodeIndex]

	g.setStable(nd)

	var messagesSent int

	for i := 0; i < len(sp.Fresh); i++ {
		if sp.Fresh[i] {
			// Advertise change to neighbors
			for _, link := range nd.Links {

				// Check that it's not this neighbor that has advertised this route to me
				if !sp.heardFrom(i, g.Nodes[link]) {

					if nd.CanTellAbout(sp.NextHop[i], g.Nodes[link]) {
						hasBecomeUnstable := g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						if hasBecomeUnstable {
							g.setUnstable(g.Nodes[link])
						}
						messagesSent++
					}
				}

			}
			sp.Fresh[i] = false
		}
	}

	return messagesSent
}

func (g *Graph) validateAsn(asn int) bool {
	_, ok := g.Nodes[asn]
	return ok
}

// GetRoute returns a path (if it exists) from an origin to a destination along with the types of links used
// The first array is 1 ELEMENT LONGER than the second
func (g *Graph) GetRoute(originAsn int, destinationAsn int) ([]*Node, []int) {
	if !g.validateAsn(originAsn) || !g.validateAsn(destinationAsn) {
		fmt.Println("UNKNOWN ROUTE")
		return nil, nil
	}
	route := make([]*Node, 0, 5)
	linkTypes := make([]int, 0, 5)

	cursorAsn := originAsn
	for cursorAsn != destinationAsn {
		routeNum := g.Speakers[cursorAsn].hasRoute(g.Nodes[destinationAsn])
		if routeNum < 0 {
			return nil, nil
		}
		route = append(route, g.Nodes[cursorAsn])
		hopType := g.Speakers[cursorAsn].getNextHopType(g.Nodes[cursorAsn], routeNum)
		linkTypes = append(linkTypes, hopType)
		cursorAsn = g.Speakers[cursorAsn].NextHop[routeNum].Asn
	}

	// Add destination AS node
	return append(route, g.Nodes[destinationAsn]), linkTypes
}

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

	// TODO: Modify this
	return true, 0
}

func (g *Graph) printSpeakerStatus(asn int) {
	if !g.validateAsn(asn) {
		fmt.Println("INVALID AS number")
	} else {
		fmt.Println("Speaker #" + u.Str(g.Nodes[asn].Asn))
		fmt.Println("	neighbors: " + u.Str(len(g.Nodes[asn].Links)))
		fmt.Println("	ROUTES: ")
		fmt.Println(g.Speakers[asn].String(g.Nodes[asn]))
	}
}

// Copy returns a new Graph
func (g *Graph) Copy() AbstractGraph {
	// TODO: Think if deeper copy is needed
	copyGraph := Graph{
		Nodes:     make(map[int]*Node),
		Speakers:  make(map[int]*Speaker),
		unstable:  make(map[*Node]bool),
		remaining: g.remaining, // Just an int
	}

	for k, v := range g.Nodes {
		copyGraph.Nodes[k] = v.Copy()
	}

	// Deep copy of Speakers
	for k, v := range g.Speakers {
		copyGraph.Speakers[k] = v.Copy()
	}

	for k := range g.unstable {
		copyGraph.unstable[copyGraph.Nodes[k.Asn]] = true
	}

	return &copyGraph
}
