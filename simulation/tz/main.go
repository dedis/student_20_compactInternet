package tz

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/shell"
	. "dedis.epfl.ch/shell"
	u "dedis.epfl.ch/u"
)

// LoadFromCsv imports the structure of the AS graph from a preprocessed .csv file
func LoadFromCsv(graph *Graph, filename string) error {

	csvfile, err := os.Open(filename)
	if err != nil {
		return err
	}

	reader := csv.NewReader(csvfile)

	var currAsn int = -1
	var currLinks Link = make(Link, 5)
	var currTypes Rel = make(Rel, 5)

	for i := 0; ; i++ {
		row, err := reader.Read()

		if err == io.EOF {
			if currAsn != -1 {
				tempNode := ToNode(currAsn, currLinks, currTypes)
				graph.Nodes[currAsn] = &tempNode
			}
			break
		}
		if u.Int(row[0]) != currAsn {
			if currAsn != -1 {
				tempNode := ToNode(currAsn, currLinks, currTypes)
				graph.Nodes[currAsn] = &tempNode
			}
			currAsn = u.Int(row[0])
			currLinks = []int{}
			currLinks = append(currLinks, u.Int(row[1]))
			currTypes = []int{}
			currTypes = append(currTypes, u.Int(row[2]))
		} else {
			currLinks = append(currLinks, u.Int(row[1]))
			currTypes = append(currTypes, u.Int(row[2]))
		}
	}

	return nil

}

// WriteLandmarksToCsv stores landmarks to a csv file
// TODO: Could use WriteToCsv
func WriteLandmarksToCsv(filename string, payload *Landmarks) {

	csvFile, err := os.Create(filename)
	if err != nil {
		panic("Unable to open Landmarks file")
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	writer.WriteAll(payload.Serialize(0))
}

// WriteWitnessesToCsv stores witnesses to a csv file
func WriteWitnessesToCsv(filename string, payload *map[int]*DijkstraGraph) {

	csvFile, err := os.Create(filename)
	if err != nil {
		panic("Unable to open Witnesses file")
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	for index := range *payload {
		writer.WriteAll((*payload)[index].Serialize(index))
	}
}

// WriteToCsv stores a map of Serializable objects to a csv file
func WriteToCsv(filename string, payload *map[int]Serializable) {

	csvFile, err := os.Create(filename)
	if err != nil {
		panic("Unable to open the file")
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	for index := range *payload {
		writer.WriteAll((*payload)[index].Serialize(index))
	}
}

// LoadLandmarksFromCsv retrieves landmarks a csv file
func (g *Graph) LoadLandmarksFromCsv(filename string) {
	csvFile, err := os.Open(filename)
	if err != nil {
		panic("Could not open landmarks file")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	for lvl := 0; lvl <= g.K; lvl++ {
		g.Landmarks[lvl] = make(map[*Node]bool)
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			// Done
			break
		}

		g.Landmarks[u.Int(row[0])][g.Nodes[u.Int(row[1])]] = true
	}
}

// LoadWitnessesFromCsv retrieves witnesses from a csv file
func (g *Graph) LoadWitnessesFromCsv(filename string) {
	csvFile, err := os.Open(filename)
	if err != nil {
		panic("Could not open witness file")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	var currRound int = -1

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			// Done
			break
		}

		if nxRound := u.Int(row[0]); nxRound != currRound {
			currRound = nxRound
			tempGraph := make(DijkstraGraph)
			g.Witnesses[currRound] = &tempGraph
		}

		(*g.Witnesses[currRound])[u.Int(row[1])] = &dijkstraNode{
			reference: u.Int(row[1]),
			distance:  u.Int64(row[2]),
			parent:    g.Nodes[u.Int(row[3])],
			nextHop:   g.Nodes[u.Int(row[4])],
		}
	}
}

// LoadBunchesFromCsv imports bunches from a csv file
func (g *Graph) LoadBunchesFromCsv(filename string) {
	csvFile, err := os.Open(filename)
	if err != nil {
		panic("Could not open witness file")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		bunchOf := u.Int(row[0])

		if _, exists := g.Bunches[bunchOf]; !exists {
			g.Bunches[bunchOf] = make(map[int]*dijkstraNode)
		}

		g.Bunches[bunchOf][u.Int(row[1])] = &dijkstraNode{
			reference: bunchOf,
			distance:  u.Int64(row[2]),
			parent:    g.Nodes[u.Int(row[1])],
			nextHop:   g.Nodes[u.Int(row[3])],
		}
	}

	// Debug check
	for k, bc := range g.Bunches {
		if len(bc) < len(g.Landmarks[g.K-1]) {
			panic("Node " + u.Str(k) + "was not loaded correctly")
		}
	}
}

var commandParams = map[string]int{"route": 2, "test-link": 2, "bunch": 1, "witness": 2, "delete": 2, "help": 0, "exit": 0} //map[string]int{"show": 1, "add-route": 1, "evolve": 0, "route": 2, "help": 0, "exit": 0}

var sh *Shell

// SetupShell initializes a global shell for this module
func SetupShell() {
	sh = InitShell("$", " ")
}

// ExecCommand executes an instruction
func (g *Graph) ExecCommand() bool {
	cmd := sh.GetCommand()

	if len(cmd) == 0 {
		return true
	}

	requiredParams, commandOk := commandParams[cmd[0]]

	if !commandOk {
		fmt.Println("Unknown command")
		return true
	} else if len(cmd[1:]) != requiredParams {
		fmt.Printf("Mismatching arguments. %s command requires %d params\n", strings.ToUpper(cmd[0]), requiredParams)
		return true
	}

	switch cmd[0] {
	case "route":
		g.PrintRoute(u.Int(cmd[1]), u.Int(cmd[2]))

	case "test-link":
		g.TestLink(u.Int(cmd[1]), u.Int(cmd[2]))

	case "bunch":
		fmt.Printf("Size:%d\t[", len(g.Bunches[u.Int(cmd[1])]))
		for b := range g.Bunches[u.Int(cmd[1])] {
			fmt.Printf(" %d ", b)
		}
		fmt.Println("]")

	case "witness":
		witness := (*g.Witnesses[u.Int(cmd[1])])[u.Int(cmd[2])]
		fmt.Printf("\tLevel %d witness of %d is %d (distance %d, next-hop %d)\n",
			u.Int(cmd[1]),
			u.Int(cmd[2]),
			witness.parent.Asn,
			witness.distance,
			witness.nextHop.Asn)

	case "delete":
		_, asnUpdated, asnDistance := g.RemoveEdge(u.Int(cmd[1]), u.Int(cmd[2]))
		fmt.Printf("Graph updated, %d nodes exchanged updates, the average distance from link is %f\n", len(asnUpdated), asnDistance.Mean())

	case "help":
		fmt.Println("The available commands are:")
		for keyword := range commandParams {
			fmt.Printf("\t%s\t (%d args)\n", keyword, commandParams[keyword])
		}

	case "exit":
		return false
	}

	return true
}

//var k int

//var witnesses *map[int]*DijkstraGraph
//var bunches *Clusters

// func Main() {

// 	sh := InitShell("$", " ")

// 	// Initialize the graph
// 	graph := InitGraph()
// 	graph.K = 3

// 	err := LoadFromCsv(&graph, "./data/202003-edges.csv") //LoadFromCsv("../../../simulation/test.csv") //LoadFromCsv("../../../simulation/202003-edges.csv")
// 	if err != nil {
// 		fmt.Println(err)
// 		panic(err)
// 	}

// 	rand.Seed(time.Now().UnixNano())

// 	graph.ElectLandmarks(RandomStrategy)

// 	graph.Evolve()

// 	WriteWitnessesToCsv("./data/202003-witnesses.csv", &graph.Witnesses)
// 	WriteToCsv("./data/202003-bunches.csv", &map[int]Serializable{0: &graph.Bunches})

// 	fmt.Println("/////////////")

// 	// Here, data are loaded from file
// 	sh.Write("Loading witnesses...")
// 	graph.LoadWitnessesFromCsv("./data/202003-witnesses.csv")
// 	sh.Write("	", Yellow, "[OK]", Clear, "\n")
// 	sh.Write("Loading bunches...")
// 	graph.LoadBunchesFromCsv("./data/202003-bunches.csv")
// 	sh.Write("	", Yellow, "[OK]", Clear, "\n")

// 	for graph.ExecCommand() {
// 	}

// }

// PrintRoute nicely prints the route from origin to destination returned by GetRoute
func (g *Graph) PrintRoute(originAsn int, destinationAsn int) {
	path, types := g.GetRoute(originAsn, destinationAsn)

	if path != nil {
		sh.Write("PATH: (", shell.Green, u.Str(len(path)-1), shell.Clear, ") ")
		sh.Write(fmt.Sprintf("\t%d", originAsn))

		for idx, step := range path[1:] {
			sh.Write(fmt.Sprintf(" %s %d", LinkTypeToSymbol(types[idx]), step.Asn))
		}

		sh.Write("\n")
	} else {
		sh.Write("NO ROUTE FOUND\n")
	}
}

// TestLink nicely prints the type of a link between two ASes, if it exists
func (g *Graph) TestLink(originAsn int, destinationAsn int) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("LINK NOT FOUND")
		}
	}()

	origNode, origOk := g.Nodes[originAsn]
	destNode, destOk := g.Nodes[destinationAsn]
	if !origOk || !destOk {
		fmt.Println("INVALID AS SPECIFIED")
	} else {
		fmt.Printf("	LINK: #%d %s #%d\n", originAsn, LinkTypeToSymbol(origNode.GetNeighborType(destNode)), destinationAsn)
	}
}
