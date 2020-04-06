package main

import (
	"fmt"

	"../u"
)

// DijkstraGraph contains nearest landmark information
type DijkstraGraph map[int]*dijkstraNode

type dijkstraNode struct {
	reference int
	distance  int64
	parent    *Node
	nextHop   *Node
}

func (d *dijkstraNode) String() string {
	return "<" + u.Str(d.reference) + "= " + u.Str(d.parent.Asn) + "..." + u.Str(d.nextHop.Asn) + "->" + "(" + u.Str64(d.distance) + ")>"
}

// Serialize produces a representation of the DijkstraGraph suitable to be saved to file
func (d *DijkstraGraph) Serialize(index int) [][]string {
	rows := make([][]string, 0, len(*d))

	for key, val := range *d {
		// TODO: Debug
		if key != val.reference {
			fmt.Printf("For key %d, dijkstraNode (ref %d) contained: %s\n", key, val.reference, val.String())
			panic("Debug check failed")
		}
		rows = append(rows, []string{u.Str(index), u.Str(key), u.Str64(val.distance), u.Str(val.parent.Asn), u.Str(val.nextHop.Asn)})
	}

	return rows
}

func (d *DijkstraGraph) runDijkstra(nodes *map[int]*Node, frontier *Frontier, frontierPopulation int) {
	for frontierPopulation > 0 {
		expandFrom := frontier.getFromClosest()
		frontierPopulation--
		frontierPopulation += frontier.expandFromNode(nodes, d, expandFrom)
	}
}
