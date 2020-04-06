package main

import "../u"

// Clusters represents both Clusters and Bunches in the algorithm
// !!WARNING!! TODO: Right now, clusters loaded from file do not have 'parent' filled
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
				Zones:       make(map[int64]map[*dijkstraNode]bool),
				MinDistance: 0,
			}
			clusterFrontier.Zones[0] = map[*dijkstraNode]bool{&source: true}

			wClusterGraph.runDijkstra(nodes, &clusterFrontier, 1)

			// Create cluster for w
			(*c)[w.Asn] = make(map[int]*dijkstraNode)
			for nd := range wClusterGraph {
				if wClusterGraph[nd].distance < (*prevRound)[nd].distance {
					(*c)[w.Asn][nd] = wClusterGraph[nd]
				}
			}
		}
	}
}
