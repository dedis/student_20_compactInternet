package tz

import (
	"fmt"
	"strings"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

// Frontier makes easy to retrieve closest dijkstra nodes
type Frontier struct {
	Zones       map[int64]map[int]*dijkstraNode
	MinDistance int64
}

func (f *Frontier) String() string {
	var sb strings.Builder
	sb.WriteString("Frontier (distance " + u.Str64(f.MinDistance) + "):\n")
	for _, c := range f.Zones[f.MinDistance] {
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

func (f *Frontier) getSomeNode(distance int64) *dijkstraNode {
	for _, k := range f.Zones[distance] {
		return k
	}

	panic(fmt.Sprintf("The minimum distance Zone (%d) of the frontier was empty", distance))
}

func (f *Frontier) getFromClosest() *dijkstraNode {

	elem := f.getSomeNode(f.MinDistance)

	f.deleteFromFrontier(elem)

	return elem

}

// AddToFrontier inserts a new dijkstraNode in the right Zone of the frontier
// returns true if the insertion was successful
// If oldDistance is < 0, the node has just been discovered
func (f *Frontier) addToFrontier(n *dijkstraNode) bool {

	if n.distance == int64Max {
		panic("Frontier cannot contain infinite distance nodes")
	}

	_, ok := f.Zones[n.distance]
	if !ok {
		f.Zones[n.distance] = make(map[int]*dijkstraNode)
	}

	if _, exists := f.Zones[n.distance][n.reference]; exists {
		return false
	}

	f.Zones[n.distance][n.reference] = n

	if n.distance < f.MinDistance {
		f.MinDistance = n.distance
	}

	if f.MinDistance == int64Max {
		panic("Frontier has been corrupted")
	}

	return true
}

// Removes an element from the frontier
// returns false if the element was not present
func (f *Frontier) deleteFromFrontier(n *dijkstraNode) bool {
	_, existed := f.Zones[f.MinDistance][n.reference]

	delete(f.Zones[f.MinDistance], n.reference)

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

	return existed
}

func (f *Frontier) expandFromNode(nodes *map[int]*Node, dijkstraGraph *DijkstraGraph, n *dijkstraNode) int {
	var discoveredNodes int = 0

	for _, neighbor := range (*nodes)[n.reference].Links {

		// Work also on induced subgraphs
		if _, inSubgraph := (*nodes)[neighbor]; inSubgraph {
			d, exists := (*dijkstraGraph)[neighbor]
			updatedDistance := n.distance + edgeWeight
			if exists {
				// Relax edge if needed
				if d.distance > updatedDistance {
					if f.deleteFromFrontier(d) {
						discoveredNodes--
					}
					d.distance = updatedDistance
					d.parent = n.parent
					d.nextHop = (*nodes)[n.reference]
					if f.addToFrontier(d) {
						discoveredNodes++
					}
				}
			} else {
				d = &dijkstraNode{
					reference: neighbor,
					distance:  updatedDistance,
					parent:    n.parent,
					nextHop:   (*nodes)[n.reference],
				}
				(*dijkstraGraph)[neighbor] = d
				if f.addToFrontier(d) {
					discoveredNodes++
				}
			}
		}
	}

	return discoveredNodes
}
