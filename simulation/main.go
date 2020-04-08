package main

import (
	"fmt"
	"math/rand"
	"time"

	"dedis.epfl.ch/audit"
	"dedis.epfl.ch/bgp"
	tz "dedis.epfl.ch/tz"
)

func main() {

	// BGP graph
	bgpGraph := bgp.InitGraph()

	bgp.LoadFromCsv(&bgpGraph, "./data/202003-edges.csv")

	// TZ graph
	tzGraph := tz.InitGraph()
	tzGraph.K = 3

	tz.LoadFromCsv(&tzGraph, "./data/202003-edges.csv")

	rand.Seed(time.Now().UnixNano())

	/*
		tzGraph.ElectLandmarks(tz.ImmunityStrategy)
		tzGraph.Preprocess()

		tz.WriteWitnessesToCsv("./data/202003-immunity-witnesses.csv", &tzGraph.Witnesses)
		tz.WriteToCsv("./data/202003-immunity-bunches.csv", &map[int]core.Serializable{0: &tzGraph.Bunches})
	*/

	tzGraph.LoadWitnessesFromCsv("./data/202003-harmonic(orig)-witnesses.csv")
	tzGraph.LoadBunchesFromCsv("./data/202003-harmonic(orig)-bunches.csv")

	avgStretch, maxStretch := audit.MeasureStretch(&bgpGraph, &tzGraph, 4, 100)
	// Measure stretch
	fmt.Printf("Average stretch: %f		Maximum stretch: %f\n", avgStretch, maxStretch)

	bgp.SetupShell()
	tz.SetupShell()

	for tzGraph.ExecCommand() {
	}

}
