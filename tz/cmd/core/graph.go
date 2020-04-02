package main

import (
	"fmt"

	"../u"
)

// Graph represents the AS graph
type Graph struct {
	Nodes     map[int]*Node
	Speakers  map[int]*Speaker
	unstable  map[*Node]bool
	remaining int
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
			nextHopType := sp.getNextHopType(nd, i)
			// Advertise change to neighbors
			for k, link := range nd.Links {

				// Check that it's not this neighbor that has advertised this route to me
				if !sp.heardFrom(i, g.Nodes[link]) {
					var flag bool = false

					switch linkType := nd.Type[k]; {
					// CUSTOMER: Advertise all routes
					case linkType == ToCustomer:
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++

					// PEER: Advertise routes from customers and peers? (TODO?)
					case linkType == ToPeer && (nextHopType == ToCustomer || nextHopType == ToPeer):
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++

					// PROVIDER: Advertise routes from customers
					case linkType == ToProvider && nextHopType == ToCustomer:
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++
					}

					if flag {
						g.setUnstable(g.Nodes[link])
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
