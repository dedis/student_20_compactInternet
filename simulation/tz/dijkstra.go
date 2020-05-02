package tz

import (
	"fmt"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
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

func (d *dijkstraNode) Copy(nodes *map[int]*Node) *dijkstraNode {
	dijNodeCopy := dijkstraNode{
		reference: d.reference,
		distance:  d.distance,
		parent:    (*nodes)[d.parent.Asn],
		nextHop:   (*nodes)[d.nextHop.Asn],
	}

	return &dijNodeCopy
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
		frontierPopulation += frontier.expandFromNode(nodes, d, expandFrom, false)
		// TODO: Remove it, when frontierPopulation is integrated in Frontier struct
		// frontier.checkFrontierConsistency(frontierPopulation)
	}

	// Discover non GR-reachable nodes and run vanilla Dijkstra
	// TODO: Maybe refactor
	notCoveredNodes := 0
	nonGRneighborhood := make(map[int]*Node)
	for asn, nd := range *nodes {
		if _, reached := (*d)[asn]; !reached {
			nonGRneighborhood[asn] = (*nodes)[asn]
			for _, l := range nd.Links {
				// Must check that neighbor is in subgraph before considering it
				if neighborNode, neighborInSubgraph := (*nodes)[l]; neighborInSubgraph {
					nonGRneighborhood[l] = neighborNode
					// If it is reachable, add to frontier
					if _, isReachable := (*d)[l]; isReachable {
						if frontier.addToFrontier((*d)[l]) {
							frontierPopulation++
						}
					}
				}
			}
			notCoveredNodes++
		}
	}
	fmt.Println(notCoveredNodes)

	// frontier.checkFrontierConsistency(frontierPopulation)

	for frontierPopulation > 0 {
		expandFrom := frontier.getFromClosest()
		frontierPopulation--
		frontierPopulation += frontier.expandFromNode(&nonGRneighborhood, d, expandFrom, true)
		// frontier.checkFrontierConsistency(frontierPopulation)
	}

	notCoveredNodes++
}

// Copy returns a duplicate of DijkstraGraph
func (d *DijkstraGraph) Copy(nodes *map[int]*Node) *DijkstraGraph {
	dijkstraCopy := make(DijkstraGraph)

	for k, v := range *d {
		dijkstraCopy[k] = v.Copy(nodes)
	}

	return &dijkstraCopy
}
