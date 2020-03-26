package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	. "../shell"
	"../u"
)

// ToProvider specifies the link with a provider
const ToProvider int = 1

// ToPeer specifies the link with a peer
const ToPeer int = 0

// ToCustomer specifies the link with a customer
const ToCustomer int = (-1)

// Link represents and edge in AS graph
type Link []int

func (l Link) String() string {
	var sb strings.Builder
	for i := 0; i < len(l); i++ {
		sb.WriteString("-> " + u.Str(l[i]))
	}
	return sb.String()
}

// Rel represents the type of edge in AS graph
type Rel []int

func (r Rel) String() string {
	var sb strings.Builder
	for i := 0; i < len(r); i++ {
		switch r[i] {
		case 1:
			sb.WriteString("provider ")
		case 0:
			sb.WriteString("peer ")
		case -1:
			sb.WriteString("customer ")
		}
	}
	return sb.String()
}

// Node epresents an AS in the graph
type Node struct {
	Asn   int
	Links Link
	Type  Rel
}

// ToNode initializes a Node, copying the arrays
func ToNode(asn int, links Link, types Rel) Node {
	var temp Node = Node{Asn: asn, Links: make(Link, len(links)), Type: make(Rel, len(types))}
	copy(temp.Links, links)
	copy(temp.Type, types)
	return temp
}

func (n Node) String() string {
	var sb strings.Builder
	sb.WriteString("AS " + u.Str(n.Asn) + ": (" + u.Str(len(n.Links)) + " links)\n")
	return sb.String()
}

// Speaker represents a (BGP, TZ, ...) speaker
type Speaker struct {
	Fresh        []bool
	Destinations []*Node
	NextHop      []*Node
	Length       []int
}

// Unicode alternative symbols: '🡕''🡒''🡖'
func linkTypeToSymbol(linkType int) string {
	if linkType == 1 {
		return `^`
	} else if linkType == 0 {
		return `=`
	} else if linkType == -1 {
		return `v`
	} else {
		return "?"
	}
}

func (s *Speaker) String(n *Node) string {
	// TODO: Refine it
	var sb strings.Builder
	for idx, dest := range s.Destinations {
		sb.WriteString("	")
		nextHopType := n.getNextHopType(s, idx)

		sb.WriteString("(")
		sb.WriteString(u.Str(s.Length[idx]))
		sb.WriteString(") ")

		switch s.Length[idx] {
		case 0:
		case 1:
			if dest.Asn != s.NextHop[idx].Asn {
				panic("Destination != NextHop in length=1 paths")
			}
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
		case 2:
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
			sb.WriteString(u.Str(s.NextHop[idx].Asn))
			sb.WriteString(" > ")
		default:
			sb.WriteString(linkTypeToSymbol(nextHopType))
			sb.WriteString(" ")
			sb.WriteString(u.Str(s.NextHop[idx].Asn))
			sb.WriteString(" > ")
			sb.WriteString("...")
			sb.WriteString(" > ")
		}

		sb.WriteString(u.Str(dest.Asn))
		sb.WriteString("\n")
	}
	return sb.String()
}

// Graph represents the AS graph
type Graph struct {
	Nodes     map[int]*Node
	Speakers  map[int]*Speaker
	unstable  map[*Node]bool
	remaining int
}

func (l *Link) search(target int) int {
	slice := (*l)[:]

	var global int = 0

	for {
		if len(slice) == 0 {
			panic("Element " + u.Str(target) + " not found in " + (*l)[:].String())
		}
		if len(slice) == 1 && slice[0] != target {
			panic("Element " + u.Str(target) + " not found in " + (*l)[:].String())
		}

		cursor := len(slice) / 2

		switch x := slice[cursor]; {
		case x == target:
			return global + cursor
		case x < target:
			slice = slice[cursor+1:]
			global += cursor + 1
		case x > target:
			slice = slice[:cursor]
		}
	}
}

// This function assumes, for performance reasons, that the node n does NOT appear in path
func (n *Node) getNextHopType(s *Speaker, destIndex int) int {
	if s.Length[destIndex] == 0 {
		// The node *n is the origin, return ToCustomer
		return ToCustomer
	}
	return n.getNeighborType(s.NextHop[destIndex])
}

func (n *Node) getNeighborType(neighborNode *Node) int {
	linkIndex := n.Links.search(neighborNode.Asn)
	return n.Type[linkIndex]
}

// LoadFromCsv imports the structure of the AS graph from a preprocessed .csv file
func LoadFromCsv(filename string) (Graph, error) {

	csvfile, err := os.Open(filename)
	if err != nil {
		return Graph{}, err
	}

	reader := csv.NewReader(csvfile)

	nodes := make(map[int]*Node)
	speakers := make(map[int]*Speaker)
	unstableSet := make(map[*Node]bool)
	remaining := 0

	var currAsn int = -1
	var currLinks Link = make(Link, 5)
	var currTypes Rel = make(Rel, 5)

	for i := 0; ; i++ {
		row, err := reader.Read()

		if err == io.EOF {
			if currAsn != -1 {
				tempNode := ToNode(currAsn, currLinks, currTypes)
				nodes[currAsn] = &tempNode
				speakers[currAsn] = InitSpeaker(&tempNode)
				unstableSet[nodes[currAsn]] = true
				remaining++
			}
			break
		}
		if u.Int(row[0]) != currAsn {
			if currAsn != -1 {
				tempNode := ToNode(currAsn, currLinks, currTypes)
				nodes[currAsn] = &tempNode
				speakers[currAsn] = InitSpeaker(&tempNode)
				unstableSet[nodes[currAsn]] = true
				remaining++
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

	return Graph{Nodes: nodes, Speakers: speakers, unstable: unstableSet, remaining: remaining}, nil

}

// InitSpeaker initializes the Speaker associated with a certain Node
func InitSpeaker(node *Node) *Speaker {
	// TODO: Clean method
	speaker := Speaker{Fresh: nil, Destinations: nil, NextHop: nil, Length: nil}
	return &speaker
}

// SetDestinations updates the speakers according to the set of chosen destinations
func (g *Graph) SetDestinations(dest map[int]bool) {
	for asn := range dest {
		if g.Speakers[asn].hasRoute(g.Nodes[asn]) < 0 {
			g.Speakers[asn].Fresh = append(g.Speakers[asn].Fresh, true)
			g.Speakers[asn].Destinations = append(g.Speakers[asn].Destinations, g.Nodes[asn])
			g.Speakers[asn].NextHop = append(g.Speakers[asn].NextHop, g.Nodes[asn])
			g.Speakers[asn].Length = append(g.Speakers[asn].Length, 0)
			g.setUnstable(g.Nodes[asn])
		}
	}
}

func (s *Speaker) hasRoute(dest *Node) int {
	for i, d := range s.Destinations {
		if d == dest {
			return i
		}
	}
	return -1
}

func (s *Speaker) heardFrom(destIndex int, neighbor *Node) bool {
	return s.NextHop[destIndex].Asn == neighbor.Asn
}

func (s *Speaker) addDestination(currNode *Node, dest *Node, nextHop *Node, length int) bool {
	s.Fresh = append(s.Fresh, true)
	s.Destinations = append(s.Destinations, dest)
	s.NextHop = append(s.NextHop, nextHop)
	s.Length = append(s.Length, length+1)
	return true
}

func (s *Speaker) updateDestination(currNode *Node, routeIndex int, dest *Node, nextHop *Node, length int) bool {
	// Compare with current path
	oldNextType := currNode.getNextHopType(s, routeIndex)
	newNextType := currNode.getNeighborType(nextHop)

	// Applies: customer < peer < provider (smaller is better)
	if (newNextType < oldNextType) ||
		(oldNextType == newNextType && length+1 < s.Length[routeIndex]) {
		// Higher preference or same preference, but the new route is shorter
		s.Fresh[routeIndex] = true
		s.Length[routeIndex] = length + 1
		s.NextHop[routeIndex] = nextHop
		return true
	}

	return false
}

func (s *Speaker) advertise(neighborNode *Node, destination *Node, nextHop *Node, length int) bool {
	routeNum := s.hasRoute(destination)
	if routeNum < 0 {
		// The neighbor speaker does not have the destination yet
		return s.addDestination(neighborNode, destination, nextHop, length)
	}
	// The neighbor has the destination, check which one is better
	return s.updateDestination(neighborNode, routeNum, destination, nextHop, length)
}

func (g *Graph) setUnstable(node *Node) {
	_, pr := g.unstable[node]
	if !pr {
		g.remaining++
		g.unstable[node] = true
	}
}

func (g *Graph) setStable(node *Node) {
	_, pr := g.unstable[node]
	if pr {
		g.remaining--
		delete(g.unstable, node)
	}
}

// Activate evolves the status of a speaker
func (g *Graph) Activate(nodeIndex int) int {

	nd := g.Nodes[nodeIndex]
	sp := g.Speakers[nodeIndex]

	g.setStable(nd)

	var messagesSent int

	for i := 0; i < len(sp.Fresh); i++ {
		if sp.Fresh[i] {
			nextHopType := nd.getNextHopType(sp, i)
			// Advertise change to neighbors
			for k, link := range nd.Links {

				// Check that it's not this neighbor that has advertised this route to me
				if !sp.heardFrom(i, g.Nodes[link]) {
					var flag bool = false

					switch linkType := nd.Type[k]; {
					// CUSTOMER: Advertise all routes
					case linkType == ToCustomer:
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++

					// PEER: Advertise routes from customers and peers? (TODO?)
					case linkType == ToPeer && (nextHopType == ToCustomer || nextHopType == ToPeer):
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++

					// PROVIDER: Advertise routes from customers
					case linkType == ToProvider && nextHopType == ToCustomer:
						flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], nd, sp.Length[i])
						messagesSent++
					}

					if flag {
						g.setUnstable(g.Nodes[link])
					}
				}

			}
			sp.Fresh[i] = false
		}
	}

	return messagesSent
}

var sh *Shell

var commandParams = map[string]int{"show": 1, "add-route": 1, "evolve": 0, "route": 2, "help": 0, "exit": 0}

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

	graph, err := LoadFromCsv("../../../simulation/202003-edges.csv")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	/*
		graph.SetDestinations(map[int]bool{
			398255: true,
			12637:  true,
			38191:  true,
		})

		fmt.Printf("Activate node #398255:\n")
		graph.Activate(398255)
		graph.printRoutes(398255)
		graph.printRoutes(174)
		graph.printRoutes(3356)
		graph.printRoutes(4)

		fmt.Printf("Activate node #174:\n")
		graph.Activate(174)
		graph.printRoutes(398255)
		graph.printRoutes(174)
		graph.printRoutes(3356)
		graph.printRoutes(4)

		fmt.Printf("Activate node #4:\n")
		graph.Activate(4)
		graph.printRoutes(398255)
		graph.printRoutes(174)
		graph.printRoutes(3356)
		graph.printRoutes(4)

		fmt.Printf("Activate node #174:\n")
		graph.Activate(174)
		graph.printRoutes(398255)
		graph.printRoutes(174)
		graph.printRoutes(3356)
		graph.printRoutes(4)

		fmt.Println("Remaining: " + u.Str(graph.remaining))

		idx := graph.Nodes[3356].Links.search(174)
		fmt.Println("nodes[" + u.Str(idx) + "] = " + u.Str(graph.Nodes[3356].Links[idx]))
		fmt.Println(graph.Nodes[3356].Type[idx])

		//graph.Evolve()

	*/
	for graph.ExecCommand() {
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

func (g *Graph) validateAsn(asn int) bool {
	_, ok := g.Nodes[asn]
	return ok
}

// GetRoute returns a path (if it exists) from an origin to a destination along with the types of links used
// The first array is 1 ELEMENT LONGER than the second
func (g *Graph) GetRoute(originAsn int, destinationAsn int) ([]*Node, []int) {
	if !g.validateAsn(originAsn) || !g.validateAsn(destinationAsn) {
		fmt.Println("UNKNOWN ROUTE")
		return nil, nil
	} else {
		route := make([]*Node, 0, 5)
		linkTypes := make([]int, 0, 5)

		cursorAsn := originAsn
		for cursorAsn != destinationAsn {
			routeNum := g.Speakers[cursorAsn].hasRoute(g.Nodes[destinationAsn])
			if routeNum < 0 {
				return nil, nil
			}
			route = append(route, g.Nodes[cursorAsn])
			hopType := g.Nodes[cursorAsn].getNextHopType(g.Speakers[cursorAsn], routeNum)
			linkTypes = append(linkTypes, hopType)
			cursorAsn = g.Speakers[cursorAsn].NextHop[routeNum].Asn
		}

		// Add destination AS node
		return append(route, g.Nodes[destinationAsn]), linkTypes
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

func (g *Graph) printSpeakerStatus(asn int) {
	if !g.validateAsn(asn) {
		fmt.Println("INVALID AS number")
	} else {
		fmt.Println("Speaker #" + u.Str(g.Nodes[asn].Asn))
		fmt.Println("	neighbors: " + u.Str(len(g.Nodes[asn].Links)))
		fmt.Println("	ROUTES: ")
		fmt.Println(g.Speakers[asn].String(g.Nodes[asn]))
	}
}
