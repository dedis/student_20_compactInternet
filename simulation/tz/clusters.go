package tz

import (
	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

// Clusters represents both Clusters and Bunches in the algorithm
type Clusters map[int]map[int]*dijkstraNode

// Serialize implements the interface Serializable for *Clusters
func (c *Clusters) Serialize(index int) [][]string {
	rows := make([][]string, 0, len(*c))

	for key, value := range *c {
		for asn, nd := range value {
			rows = append(rows, []string{u.Str(key), u.Str(asn), u.Str64(nd.distance), u.Str(nd.nextHop.Asn)})
		}
	}

	return rows
}

func (c *Clusters) calculateClustersForRound(nodes *map[int]*Node, k int, l *Landmarks, prevRound *DijkstraGraph) {
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
				Zones:       make(map[int64]map[int]*dijkstraNode),
				MinDistance: 0,
			}
			clusterFrontier.Zones[0] = map[int]*dijkstraNode{source.reference: &source}

			wClusterGraph.runDijkstra(nodes, &clusterFrontier, 1)

			// Create cluster for w
			(*c)[w.Asn] = make(map[int]*dijkstraNode)

			// First, add all the leaves (and some other nodes) of the tree
			for nd := range wClusterGraph {
				if wClusterGraph[nd].distance < (*prevRound)[nd].distance {
					(*c)[w.Asn][nd] = wClusterGraph[nd]
				}
			}

			// Include missing nodes (to form a spanning tree)
			inPathNodes := make(map[int]*dijkstraNode)
			for _, dijNode := range (*c)[w.Asn] {
				cursor := dijNode
				for cursor.distance > 0 {
					if _, inPath := (*c)[w.Asn][cursor.reference]; !inPath {
						inPathNodes[cursor.reference] = cursor
					}
					cursor = wClusterGraph[cursor.nextHop.Asn]
				}
			}

			for ip, dij := range inPathNodes {
				(*c)[w.Asn][ip] = dij
			}
		}
	}
}

// Copy returns a duplicate of Clusters
func (c *Clusters) Copy(nodes *map[int]*Node) *Clusters {
	clusterCopy := make(Clusters)

	for k, cl := range *c {
		clusterCopy[k] = make(map[int]*dijkstraNode)
		for ci, dn := range cl {
			clusterCopy[k][ci] = dn.Copy(nodes)
		}
	}

	return &clusterCopy
}
