package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"../u"
)

// ToProvider specifies the link with a provider
const ToProvider = 1

// ToPeer specifies the link with a peer
const ToPeer = 0

// ToCustomer specifies the link with a customer
const ToCustomer = -1

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

// Copy performs a copy
func (n *Node) Copy() *Node {
	temp := *n
	return &temp
}

func (n Node) String() string {
	var sb strings.Builder
	sb.WriteString("AS " + u.Str(n.Asn) + ": (" + u.Str(len(n.Links)) + " links)\n")

	for i := 0; i < len(n.Links); i++ {
		sb.WriteString(u.Str(n.Links[i]))
		sb.WriteString(" - ")
		sb.WriteString(u.Str(n.Type[i]))
		sb.WriteString("\n")
	}
	return sb.String()
}

// Speaker represents a (BGP, TZ, ...) speaker
type Speaker struct {
	Fresh        []bool
	Destinations []*Node
	Paths        [][]*Node
}

func linkTypeToString(linkType int) string {
	switch linkType {
	case ToProvider:
		return string('🡕') //string('↑')
	case ToPeer:
		return string('🡒') //string('→')
	case ToCustomer:
		return string('🡖') //string('↓')
	default:
		return "[UNK]"
	}

}

func (s *Speaker) String() string {
	var sb strings.Builder
	for idx, dest := range s.Destinations {
		sb.WriteString("	(")
		sb.WriteString(u.Str(len(s.Paths[idx])))
		sb.WriteString(") -> ")
		sb.WriteString(u.Str(dest.Asn))
		sb.WriteString("		[")

		i := len(s.Paths[idx]) - 1
		var prevNode *Node = s.Paths[idx][i]
		sb.WriteString(" ")
		sb.WriteString(u.Str(prevNode.Asn))
		sb.WriteString(" ")
		for i > 0 {
			i--
			sb.WriteString(linkTypeToString(prevNode.getNeighborType(s.Paths[idx][i])))
			sb.WriteString(" ")
			sb.WriteString(u.Str(s.Paths[idx][i].Asn))
			sb.WriteString(" ")
			prevNode = s.Paths[idx][i]
		}
		sb.WriteString("]\n")
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
			return cursor
		case x < target:
			slice = slice[cursor+1:]
		case x > target:
			slice = slice[:cursor]
		}
	}
}

// This function assumes, for performance reasons, that the node n does NOT appear in path
func (n *Node) getNextHopType(path []*Node) int {
	if len(path) == 0 {
		// The node *n is the origin, return ToCustomer
		return ToCustomer
	}
	return n.getNeighborType(path[len(path)-1])
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
	var currNode Node

	for i := 0; ; i++ {
		row, err := reader.Read()

		if err == io.EOF {
			if currAsn != -1 {
				nodes[currAsn] = (&currNode).Copy()
				speakers[currAsn] = InitSpeaker(nodes[currAsn])
				unstableSet[nodes[currAsn]] = true
				remaining++
			}
			break
		}
		if u.Int(row[0]) != currAsn {
			if currAsn != -1 {
				nodes[currAsn] = (&currNode).Copy()
				speakers[currAsn] = InitSpeaker(nodes[currAsn])
				unstableSet[nodes[currAsn]] = true
				remaining++
			}
			currAsn = u.Int(row[0])
			currNode = Node{Asn: currAsn, Links: Link{u.Int(row[1])}, Type: []int{u.Int(row[2])}}
		} else {
			currNode.Links = append(currNode.Links, u.Int(row[1]))
			currNode.Type = append(currNode.Type, u.Int(row[2]))
		}
	}

	return Graph{Nodes: nodes, Speakers: speakers, unstable: unstableSet, remaining: remaining}, nil

}

// InitSpeaker initializes the Speaker associated with a certain Node
func InitSpeaker(node *Node) *Speaker {
	return &Speaker{Fresh: []bool{true}, Destinations: []*Node{node}, Paths: [][]*Node{{node}}}
}

func (s *Speaker) hasRoute(dest *Node) int {
	for i, d := range s.Destinations {
		if d == dest {
			return i
		}
	}
	return -1
}

func (s *Speaker) isInPath(destIndex int, targetNode *Node) bool {
	for _, traversed := range s.Paths[destIndex] {
		if traversed.Asn == targetNode.Asn {
			return true
		}
	}
	return false
}

func (s *Speaker) addDestination(currNode *Node, dest *Node, path []*Node) bool {
	// Allocate space for new path
	var tempPath []*Node
	copy(tempPath, path)
	tempPath = append(tempPath, currNode)

	s.Fresh = append(s.Fresh, true)
	s.Destinations = append(s.Destinations, dest)
	s.Paths = append(s.Paths, tempPath)
	return true
}

func (s *Speaker) updateDestination(currNode *Node, routeIndex int, dest *Node, newPath []*Node) bool {
	// Compare with current path
	oldNextType := currNode.getNextHopType(s.Paths[routeIndex][:len(s.Paths[routeIndex])-1])
	newNextType := currNode.getNextHopType(newPath)

	// Applies: customer < peer < provider (smaller is better)
	if (newNextType > oldNextType) ||
		(oldNextType == newNextType && len(s.Paths[routeIndex]) > len(newPath)+1) {
		// Higher preference or same preference, but the new route is shorter
		s.Fresh[routeIndex] = true
		// Copy array
		copy(s.Paths[routeIndex], newPath)
		s.Paths[routeIndex] = append(s.Paths[routeIndex], currNode)
		return true
	}

	return false
}

func (s *Speaker) advertise(neighborNode *Node, destination *Node, path []*Node) bool {
	if routeNum := s.hasRoute(destination); routeNum < 0 {
		// The neighbor speaker does not have the destination yet
		return s.addDestination(neighborNode, destination, path)
	} else if !s.isInPath(routeNum, neighborNode) { // Prevent loops
		// The neighbor has the destination, check which one is better
		return s.updateDestination(neighborNode, routeNum, destination, path)
	}

	return false
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
func (g *Graph) Activate(nodeIndex int) {

	nd := g.Nodes[nodeIndex]
	sp := g.Speakers[nodeIndex]

	fmt.Println("Activating " + u.Str(nd.Asn))

	g.setStable(nd)

	for i := 0; i < len(sp.Fresh); i++ {
		if sp.Fresh[i] {
			nextHopType := nd.getNextHopType(sp.Paths[i][:len(sp.Paths[i])-1])
			// Advertise change to neighbors
			for k, link := range nd.Links {
				var flag bool = false
				switch linkType := nd.Type[k]; {
				// CUSTOMER: Advertise all routes
				case linkType == ToCustomer:
					flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], sp.Paths[i])
				// PEER: Advertise routes from customers and peers? (TODO?)
				case linkType == ToPeer && (nextHopType == ToCustomer || nextHopType == ToPeer):
					flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], sp.Paths[i])
				// PROVIDER: Advertise routes from customers
				case linkType == ToProvider && nextHopType == ToCustomer:
					flag = g.Speakers[link].advertise(g.Nodes[link], sp.Destinations[i], sp.Paths[i])
				}

				if flag {
					g.setUnstable(g.Nodes[link])
				}
			}
			sp.Fresh[i] = false
		}
	}
}

func main() {

	graph, err := LoadFromCsv("../../../simulation/202003-edges.csv")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	/*
		fmt.Printf("%s", graph.Nodes[2])
		fmt.Printf("Egdes of AS#0: %s\n", graph.Nodes[278].Links)
		fmt.Printf("Of type: %s\n", graph.Nodes[278].Type)

		fmt.Printf("Node #398255: %s\n", graph.Nodes[398255])
		fmt.Printf("Speaker #398255: %s\n", graph.Speakers[398255].Destinations)
	*/

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

	graph.Evolve()
	graph.printRoutes(398255)

}

// Evolve updates the system until its convergence
func (g *Graph) Evolve() {
	for g.remaining > 0 {
		fmt.Println(":: ROUND :: Remaining " + u.Str(g.remaining))
		for k := range g.unstable {
			g.Activate(k.Asn)
		}
	}
}

func (g *Graph) printRoutes(asn int) {
	fmt.Println("Speaker #" + u.Str(g.Nodes[asn].Asn))
	fmt.Println("	neighbors: " + u.Str(len(g.Nodes[asn].Links)))
	fmt.Println("	ROUTES: ")
	fmt.Println(g.Speakers[asn])
}
