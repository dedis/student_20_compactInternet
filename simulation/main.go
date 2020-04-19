package main

import (
	"fmt"
	"math/rand"
	"time"

	"dedis.epfl.ch/audit"
	"dedis.epfl.ch/bgp"
	"dedis.epfl.ch/tz"
	"dedis.epfl.ch/u"

	. "dedis.epfl.ch/core"
)

func restoreTZ(folder string, datasetName string, k int, landmarkStrategy int) tz.Graph {
	tzGraph := tz.InitGraph()
	tzGraph.K = k

	tz.LoadFromCsv(&tzGraph, folder+datasetName+".csv")

	tzGraph.LoadLandmarksFromCsv(folder + datasetName + "-landmarks-" + u.Str(landmarkStrategy) + ".csv")
	tzGraph.LoadWitnessesFromCsv(folder + datasetName + "-witnesses-" + u.Str(landmarkStrategy) + ".csv")
	tzGraph.LoadBunchesFromCsv(folder + datasetName + "-bunches-" + u.Str(landmarkStrategy) + ".csv")

	return tzGraph
}

// folder must end with a slash
func loadAndProcessTZ(folder string, datasetName string, k int, landmarkStrategy int) tz.Graph {
	tzGraph := tz.InitGraph()
	tzGraph.K = k

	tz.LoadFromCsv(&tzGraph, folder+datasetName+".csv")

	rand.Seed(time.Now().UnixNano())

	tzGraph.ElectLandmarks(landmarkStrategy)

	tzGraph.Preprocess()

	tz.WriteLandmarksToCsv(folder+datasetName+"-landmarks-"+u.Str(landmarkStrategy)+".csv", &tzGraph.Landmarks)
	tz.WriteWitnessesToCsv(folder+datasetName+"-witnesses-"+u.Str(landmarkStrategy)+".csv", &tzGraph.Witnesses)
	tz.WriteToCsv(folder+datasetName+"-bunches-"+u.Str(landmarkStrategy)+".csv", &map[int]Serializable{0: &tzGraph.Bunches})

	return tzGraph
}

func main() {

	tzGraph := restoreTZ("./data/", "202003-full-edges", 3, tz.HarmonicStrategy)
	//loadAndProcessTZ("./data/", "202003-full-edges", 3, tz.RandomStrategy)

	bgpGraph := bgp.InitGraph()

	bgp.LoadFromCsv(&bgpGraph, "./data/202003-full-edges.csv")

	// Measure stretch
	audit.InitRecorder("./data/full-stretch-2000.csv")
	avgStretch, maxStretch := audit.MeasureStretch(&bgpGraph, &tzGraph, 4, 500)
	fmt.Printf("Average stretch: %f		Maximum stretch: %f\n", avgStretch, maxStretch)

	// audit.InitRecorder("./data/full-impact-2000.csv")
	// avgImpact, maxImpact := audit.MeasureEdgeDeletionImpact(&bgpGraph, &tzGraph, 40)
	// fmt.Printf("Average impact: %f		Maximum impact: %f\n", avgImpact, maxImpact)
	/*
		audit.InitRecorder("./data/full-deletion-stretch-3000.csv")
		avgDelStretch, maxDelStretch := audit.MeasureDeletionStretch(&bgpGraph, &tzGraph, 3000)
		fmt.Printf("Average stretch increase: %f 	Max stretch increase: %f\n", avgDelStretch, maxDelStretch)
	*/
	/* if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
		fmt.Println("Failed to initiate shutdown:", err)
	} */

	tz.SetupShell()
	bgp.SetupShell()

	for tzGraph.ExecCommand() {
	}

	/*
		tzGraph.PrintRoute(5, 4)

		tzGraph.RemoveEdge(3, 4)
		tzGraph.PrintRoute(5, 4)

		tzGraph.RemoveEdge(5, 1)
		tzGraph.PrintRoute(5, 4)
	*/
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
