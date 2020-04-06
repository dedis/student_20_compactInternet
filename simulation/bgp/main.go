package bgp

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	. "dedis.epfl.ch/shell"
	"dedis.epfl.ch/u"
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
				graph.Speakers[currAsn] = InitSpeaker(&tempNode)
				graph.unstable[graph.Nodes[currAsn]] = true
				graph.remaining++
			}
			break
		}
		if u.Int(row[0]) != currAsn {
			if currAsn != -1 {
				tempNode := ToNode(currAsn, currLinks, currTypes)
				graph.Nodes[currAsn] = &tempNode
				graph.Speakers[currAsn] = InitSpeaker(&tempNode)
				graph.unstable[graph.Nodes[currAsn]] = true
				graph.remaining++
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

var sh *Shell

var commandParams = map[string]int{"show": 1, "test-link": 2, "add-route": 1, "evolve": 0, "route": 2, "help": 0, "exit": 0}

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
	case "show":
		g.printSpeakerStatus(u.Int(cmd[1]))

	case "test-link":
		g.TestLink(u.Int(cmd[1]), u.Int(cmd[2]))

	case "add-route":
		g.SetDestinations(map[int]bool{u.Int(cmd[1]): true})

	case "evolve":
		convergenceSteps := g.Evolve()
		fmt.Printf("Equilibrium reached, %d messages exchanged on the network\n", convergenceSteps)

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

func main() {

	sh = InitShell("$", " ")

	graph := InitGraph()

	err := LoadFromCsv(&graph, "../../../simulation/202003-full-edges.csv")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	for graph.ExecCommand() {
	}
}

// PrintRoute nicely prints the route from origin to destination returned by GetRoute
func (g *Graph) PrintRoute(originAsn int, destinationAsn int) {
	path, types := g.GetRoute(originAsn, destinationAsn)

	if path != nil {
		fmt.Printf("length (%d): ", len(path))
		fmt.Printf("	%d", originAsn)

		for idx, step := range path[1:] {
			fmt.Printf(" %s %d", linkTypeToSymbol(types[idx]), step.Asn)
		}

		fmt.Print("\n")
	} else {
		fmt.Printf("NO ROUTE FOUND\n")
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
		fmt.Printf("	LINK: #%d %s #%d\n", originAsn, linkTypeToSymbol(origNode.getNeighborType(destNode)), destinationAsn)
	}
}

// Evolve updates the system until its convergence
func (g *Graph) Evolve() (stepsToConvergence int) {
	stepsToConvergence = 0

	var roundNum int = 0
	for g.remaining > 0 {
		fmt.Printf("Round %d : %d activation queued\n", roundNum, g.remaining)

		for k := range g.unstable {
			sh.Overwrite("	Activating AS#", Green, u.Str(k.Asn), Clear)
			stepsToConvergence += g.Activate(k.Asn)
		}
		fmt.Print("\n")
		roundNum++
	}

	return
}
