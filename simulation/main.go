package main

import (
	"fmt"

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

	tzGraph.LoadWitnessesFromCsv("./data/202003-witnesses.csv")
	tzGraph.LoadBunchesFromCsv("./data/202003-bunches.csv")

	// Measure stretch
	fmt.Printf("Measured stretch: %f\n", audit.MeasureStretch(&bgpGraph, &tzGraph, 4, 200))

}
