package main

import (
	tz "dedis.epfl.ch/tz"
)

func main() {

	tzGraph := tz.InitGraph()
	tzGraph.K = 3

	tz.LoadFromCsv(&tzGraph, "./data/202003-edges.csv")

	tzGraph.LoadWitnessesFromCsv("./data/202003-witnesses.csv")
	tzGraph.LoadBunchesFromCsv("./data/202003-bunches.csv")

	tz.SetupShell()

	for tzGraph.ExecCommand() {
	}

}
