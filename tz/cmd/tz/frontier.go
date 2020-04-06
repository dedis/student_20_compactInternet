package main

import (
	"fmt"
	"strings"

	"../u"
)

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

func (f *Frontier) expandFromNode(nodes *map[int]*Node, dijkstraGraph *DijkstraGraph, n *dijkstraNode) int {
	var discoveredNodes int = 0

	for _, neighbor := range (*nodes)[n.reference].Links {
		d, exists := (*dijkstraGraph)[neighbor]
		updatedDistance := n.distance + edgeWeight
		if exists {
			// Relax edge if needed
			if d.distance > updatedDistance {
				f.deleteFromFrontier(d)
				d.distance = updatedDistance
				d.parent = n.parent
				d.nextHop = (*nodes)[n.reference]
				f.addToFrontier(d)
			}
		} else {
			tempNode := dijkstraNode{
				reference: neighbor,
				distance:  updatedDistance,
				parent:    n.parent,
				nextHop:   (*nodes)[n.reference],
			}
			(*dijkstraGraph)[neighbor] = &tempNode
			f.addToFrontier(&tempNode)
			discoveredNodes++
		}
	}

	return discoveredNodes
}
