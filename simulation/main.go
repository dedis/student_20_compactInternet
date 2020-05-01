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

func restoreTZ(folder string, graphName string, structuresName string, k int, landmarkStrategy int) tz.Graph {
	tzGraph := tz.InitGraph()
	tzGraph.K = k

	tz.LoadFromCsv(&tzGraph, folder+graphName+".csv")

	tzGraph.LoadLandmarksFromCsv(folder + structuresName + "-landmarks-" + u.Str(landmarkStrategy) + ".csv")
	tzGraph.LoadWitnessesFromCsv(folder + structuresName + "-witnesses-" + u.Str(landmarkStrategy) + ".csv")
	tzGraph.LoadBunchesFromCsv(folder + structuresName + "-bunches-" + u.Str(landmarkStrategy) + ".csv")

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

	grFiltered := "-GR"

	tz.WriteLandmarksToCsv(folder+datasetName+grFiltered+"-landmarks-"+u.Str(landmarkStrategy)+".csv", &tzGraph.Landmarks)
	tz.WriteWitnessesToCsv(folder+datasetName+grFiltered+"-witnesses-"+u.Str(landmarkStrategy)+".csv", &tzGraph.Witnesses)
	tz.WriteToCsv(folder+datasetName+grFiltered+"-bunches-"+u.Str(landmarkStrategy)+".csv", &map[int]Serializable{0: &tzGraph.Bunches})

	return tzGraph
}

// folder must end with a slash
func loadAndProcessWithLandmarksTZ(folder string, datasetName string, k int, landmarkStrategy int, landmarkFile string) tz.Graph {
	tzGraph := tz.InitGraph()
	tzGraph.K = k

	tz.LoadFromCsv(&tzGraph, folder+datasetName+".csv")

	rand.Seed(time.Now().UnixNano())

	tzGraph.LoadLandmarksFromCsv(folder + landmarkFile)

	tzGraph.Preprocess()

	grFiltered := "-GR"

	tz.WriteWitnessesToCsv(folder+datasetName+grFiltered+"-witnesses-"+u.Str(landmarkStrategy)+".csv", &tzGraph.Witnesses)
	tz.WriteToCsv(folder+datasetName+grFiltered+"-bunches-"+u.Str(landmarkStrategy)+".csv", &map[int]Serializable{0: &tzGraph.Bunches})

	return tzGraph
}

/////////////////////////////
//       DISCLAIMER        //
// Use only paths with     //
// single slash (/)        //
/////////////////////////////

func main() {

	//tzGraph := restoreTZ("./data/", "202003-full-edges", 3, tz.HarmonicStrategy)
	//loadAndProcessTZ("./data/", "202003-full-edges", 3, tz.RandomStrategy)

	grTzGraph := loadAndProcessWithLandmarksTZ("./data/", "202003-full-edges", 3, tz.HarmonicStrategy, "202003-full-edges-landmarks-2.csv")
	//restoreTZ("./data/", "202003-full-edges", "202003-full-edges-GR", 3, tz.HarmonicStrategy)
	//loadAndProcessWithLandmarksTZ("./data/", "202003-full-edges", 3, tz.HarmonicStrategy, "202003-full-edges-landmarks-2.csv")

	bgpGraph := bgp.InitGraph()
	bgp.LoadFromCsv(&bgpGraph, "./data/202003-full-edges.csv")

	// bgpPointer := AbstractGraph(&bgpGraph)
	// tzPointer := AbstractGraph(&tzGraph)

	// Measure cumulative effects of deletions over stretch
	// audit.InitRecorder("./data/cumulative-deletions-12x.05-(4).csv")
	// avgCumulIncrease, maxCumulIncrease := audit.MeasureRandomDeletionsStretch(&bgpPointer, &tzPointer, 12, .05)
	// fmt.Printf("Average stretch increase (by round): %f		Maximum stretch increase (by round): %f\n", avgCumulIncrease, maxCumulIncrease)

	// Compute TZ from scratch on graph with missing edges
	// refreshedTzGraph := loadAndProcessTZ("./data/", "missing-edges-12x0.050", 3, tz.HarmonicStrategy)

	// Perform stretch measurements on fresh TZ graph and progressively adapted one
	// audit.InitRecorder("./data/missing-edges-12x0.05-stretch-3000.csv")
	// avgMissingStretch, maxMissingStretch := audit.MeasureStretch(&refreshedTzGraph, tzPointer, 2, 1500)
	// fmt.Printf("Missing edges graph, Average stretch: %f		Maximum stretch: %f\n", avgMissingStretch, maxMissingStretch)

	// audit.InitRecorder("./data/refreshed-tz-12x0.05-stretch-2000.csv")
	// avgRefreshedStretch, maxRefreshedStretch := audit.MeasureStretch(bgpPointer, &refreshedTzGraph, 2, 1000)
	// fmt.Printf("Refreshed TZ graph, Average stretch: %f		Maximum stretch: %f\n", avgRefreshedStretch, maxRefreshedStretch)

	// Measure stretch
	audit.InitRecorder("./data/full-GR-stretch-4000.csv")
	avgStretch, maxStretch := audit.MeasureStretch(&bgpGraph, &grTzGraph, 4, 1000)
	fmt.Printf("Average stretch: %f		Maximum stretch: %f\n", avgStretch, maxStretch)

	// audit.InitRecorder("./data/full-impact-2000.csv")
	// avgImpact, maxImpact := audit.MeasureEdgeDeletionImpact(&bgpGraph, &tzGraph, 40)
	// fmt.Printf("Average impact: %f		Maximum impact: %f\n", avgImpact, maxImpact)

	// audit.InitRecorder("./data/full-deletion-stretch-2000.csv")
	// avgDelStretch, maxDelStretch := audit.MeasureDeletionStretch(&bgpGraph, &tzGraph, 2000)
	// fmt.Printf("Average stretch increase: %f 	Max stretch increase: %f\n", avgDelStretch, maxDelStretch)

	// if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
	// 	fmt.Println("Failed to initiate shutdown:", err)
	// }

	tz.SetupShell()
	bgp.SetupShell()

	for grTzGraph.ExecCommand() {
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
