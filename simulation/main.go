package main

import (
	tz "dedis.epfl.ch/tz"

	. "dedis.epfl.ch/core"
)

func main() {

	tzGraph := tz.InitGraph()
	tzGraph.K = 3

	tz.LoadFromCsv(&tzGraph, "./data/test.csv")

	tzGraph.Landmarks[0] = map[*Node]bool{
		tzGraph.Nodes[1]: true,
		tzGraph.Nodes[2]: true,
		tzGraph.Nodes[3]: true,
		tzGraph.Nodes[4]: true,
		tzGraph.Nodes[5]: true,
		tzGraph.Nodes[6]: true,
		tzGraph.Nodes[7]: true,
	}
	tzGraph.Landmarks[1] = map[*Node]bool{
		tzGraph.Nodes[4]: true,
		tzGraph.Nodes[6]: true,
		tzGraph.Nodes[7]: true,
	}
	tzGraph.Landmarks[2] = map[*Node]bool{
		tzGraph.Nodes[6]: true,
	}
	tzGraph.Landmarks[3] = map[*Node]bool{}

	tzGraph.Preprocess()

	tzGraph.PrintRoute(5, 4)

	tzGraph.RemoveEdge(4, 3)

	tzGraph.PrintRoute(5, 4)

	tzGraph.RemoveEdge(5, 1)

	tzGraph.PrintRoute(5, 4)

	/*
		IDEAS:
		  - If lower level landmark receives top-level update messages, it also broadcast its presence
		  - If landmarks have topologically-dependent label (e.g. IP prefix) routing without handshake
		    is possible in most cases (also assuming they store clusters)
	*/

	/*
		// BGP graph
		bgpGraph := bgp.InitGraph()

		bgp.LoadFromCsv(&bgpGraph, "./data/202003-edges.csv")

		// TZ graph
		tzGraph := tz.InitGraph()
		tzGraph.K = 3

		tz.LoadFromCsv(&tzGraph, "./data/202003-edges.csv")

		rand.Seed(time.Now().UnixNano())

		// tzGraph.ElectLandmarks(tz.ImmunityStrategy)
		// tzGraph.Preprocess()

		// tz.WriteWitnessesToCsv("./data/202003-immunity-witnesses.csv", &tzGraph.Witnesses)
		// tz.WriteToCsv("./data/202003-immunity-bunches.csv", &map[int]core.Serializable{0: &tzGraph.Bunches})

		tzGraph.LoadWitnessesFromCsv("./data/202003-harmonic(orig)-witnesses.csv")
		tzGraph.LoadBunchesFromCsv("./data/202003-harmonic(orig)-bunches.csv")

		avgStretch, maxStretch := audit.MeasureStretch(&bgpGraph, &tzGraph, 4, 100)
		// Measure stretch
		fmt.Printf("Average stretch: %f		Maximum stretch: %f\n", avgStretch, maxStretch)

		bgp.SetupShell()
		tz.SetupShell()

		for tzGraph.ExecCommand() {
		}
	*/

}
