package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"../shell"
	. "../shell"
	"../u"
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
			currLinks = currLinks[:0]
			currLinks = append(currLinks, u.Int(row[1]))
			currTypes = currTypes[:0]
			currTypes = append(currTypes, u.Int(row[2]))
		} else {
			currLinks = append(currLinks, u.Int(row[1]))
			currTypes = append(currTypes, u.Int(row[2]))
		}
	}

	return nil

}

// Serializable represents an object that can be transferred to file
type Serializable interface {
	Serialize(index int) [][]string
}

// WriteWitnessesToCsv stores witnesses to a csv file
func WriteWitnessesToCsv(filename string, payload *map[int]*DijkstraGraph) {

	csvFile, err := os.Create(filename)
	defer csvFile.Close()
	if err != nil {
		panic("Unable to open the file")
	}

	writer := csv.NewWriter(csvFile)
	for index := range *payload {
		writer.WriteAll((*payload)[index].Serialize(index))
	}
}

// WriteToCsv stores a map of Serializable objects to a csv file
func WriteToCsv(filename string, payload *map[int]Serializable) {

	csvFile, err := os.Create(filename)
	defer csvFile.Close()
	if err != nil {
		panic("Unable to open the file")
	}

	writer := csv.NewWriter(csvFile)
	for index := range *payload {
		writer.WriteAll((*payload)[index].Serialize(index))
	}
}

func (g *Graph) LoadWitnessesFromCsv(filename string) {
	csvFile, err := os.Open(filename)
	if err != nil {
		panic("Could not open witness file")
	}

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
			reference: u.Int(row[1]),
			distance:  u.Int64(row[2]),
			parent:    nil, // TODO: Change structure in the future
			nextHop:   g.Nodes[u.Int(row[3])],
		}
	}
}

var sh *Shell

var commandParams = map[string]int{"route": 2, "help": 0, "exit": 0} //map[string]int{"show": 1, "add-route": 1, "evolve": 0, "route": 2, "help": 0, "exit": 0}

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

func main() {

	sh = InitShell("$", " ")

	// Initialize the graph
	graph := InitGraph()
	graph.K = 3

	err := LoadFromCsv(&graph, "../../../simulation/test.csv") //LoadFromCsv("../../../simulation/test.csv") //LoadFromCsv("../../../simulation/202003-edges.csv")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	//landmarks := graph.ElectLandmarks(k)

	// TODO: This calculates witnesses and bunches from scratch (it will become a different command)
	graph.Landmarks[0] = map[*Node]bool{
		graph.Nodes[1]: true,
		graph.Nodes[2]: true,
		graph.Nodes[3]: true,
		graph.Nodes[4]: true,
		graph.Nodes[5]: true,
		graph.Nodes[6]: true,
		graph.Nodes[7]: true,
	}
	graph.Landmarks[1] = map[*Node]bool{
		graph.Nodes[1]: true,
		graph.Nodes[2]: true,
		graph.Nodes[3]: true,
		graph.Nodes[4]: true,
		graph.Nodes[7]: true,
	}
	graph.Landmarks[2] = map[*Node]bool{
		graph.Nodes[4]: true,
		graph.Nodes[7]: true,
	}
	graph.Landmarks[3] = map[*Node]bool{}

	/*
			landmarks[0] = map[*Node]bool{
				graph.Nodes[1]: true,
				graph.Nodes[2]: true,
				graph.Nodes[3]: true,
				graph.Nodes[4]: true,
				graph.Nodes[5]: true,
				graph.Nodes[6]: true,
				graph.Nodes[7]: true,
			}
			landmarks[1] = map[*Node]bool{
				graph.Nodes[4]: true,
				graph.Nodes[6]: true,
				graph.Nodes[7]: true,
			}
			landmarks[2] = map[*Node]bool{
				graph.Nodes[6]: true,
			}
			landmarks[3] = map[*Node]bool{}

		fmt.Println("Landmarks:")
		fmt.Println(landmarks)
		witnesses, bunches := graph.CalculateWitnesses(k, landmarks)

		WriteWitnessesToCsv("../../../simulation/202003-witnesses.csv", witnesses)
		WriteToCsv("../../../simulation/202003-bunches.csv", &map[int]Serializable{0: bunches})

		fmt.Println("...............")

		/*
			for i := 0; i <= k; i++ {
				fmt.Printf("Round %d:\n", i)
				fmt.Println((*witnesses)[i])
			}
			fmt.Println("---")
			fmt.Println(bunches)
	*/

	graph.Evolve()

	WriteWitnessesToCsv("../../../simulation/test-witnesses.csv", &graph.Witnesses)
	WriteToCsv("../../../simulation/test-bunches.csv", &map[int]Serializable{0: &graph.Bunches})

	fmt.Println("/////////////")

	// TODO: Clean this function

	// Here, data are loaded from file
	sh.Write("Loading witnesses...")
	graph.LoadWitnessesFromCsv("../../../simulation/test-witnesses.csv")
	sh.Write("	", Yellow, "[OK]", Clear, "\n")
	sh.Write("Loading bunches...")
	graph.LoadBunchesFromCsv("../../../simulation/test-bunches.csv")
	sh.Write("	", Yellow, "[OK]", Clear, "\n")

	//fmt.Println(graph.ApproximateDistance(k, 12637, 174, witnesses, bunches))
	//fmt.Println(graph.ApproximateDistance(k, 31976, 3269, witnesses, bunches))

	for graph.ExecCommand() {
	}

}

// PrintRoute nicely prints the route from origin to destination returned by GetRoute
func (g *Graph) PrintRoute(originAsn int, destinationAsn int) {
	path, types := g.GetRoute(originAsn, destinationAsn)

	if path != nil {
		sh.Write("PATH:", "\t", "(", shell.Green, u.Str(len(path)-1), shell.Clear, ") ")
		sh.Write(fmt.Sprintf("\t%d", originAsn))

		for idx, step := range path[1:] {
			sh.Write(fmt.Sprintf(" %s %d", linkTypeToSymbol(types[idx]), step.Asn))
		}

		sh.Write("\n")
	} else {
		sh.Write("NO ROUTE FOUND\n")
	}
}
