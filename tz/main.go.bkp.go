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
const ToProvider int = 1

// ToPeer specifies the link with a peer
const ToPeer = 0

// ToCustomer specifies the link with a customer
const ToCustomer int = -1

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

func (n Node) String() string {
	return "AS " + u.Str(n.Asn) + ": (" + u.Str(len(n.Links)) + " links)\n"
}

// Speaker represents a (BGP, TZ, ...) speaker
type Speaker struct {
	Fresh        []bool
	Destinations []*Node
	Paths        [][]*Node
}

// Graph represents the AS graph
type Graph struct {
	Nodes    []Node
	Speakers []Speaker
}

func (g *Graph) search(target int) int {
	slice := g.Nodes[:]

	for {
		if len(slice) == 0 {
			return -1
		}

		cursor := len(slice) / 2

		switch x := slice[cursor].Asn {
		case x == target:
			return cursor
		case x < target:
			slice = slice[cursor:]
		case x > target:
			slice = slice[:cursor]
		}
	}
}

// LoadFromCsv imports the structure of the AS graph from a preprocessed .csv file
func LoadFromCsv(filename string) ([]Node, error) {

	csvfile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(csvfile)

	edges := make([]Node, 0, 50000)

	var currAsn int = -1
	var currNode Node

	for i := 0; ; i++ {
		row, err := reader.Read()

		if err == io.EOF {
			if currAsn != -1 {
				edges = append(edges, currNode)
			}
			break
		}
		if u.Int(row[0]) != currAsn {
			if currAsn != -1 {
				edges = append(edges, currNode)
			}
			currAsn = u.Int(row[0])
			currNode = Node{Asn: currAsn, Links: Link{u.Int(row[1])}, Type: []int{u.Int(row[2])}}
		} else {
			currNode.Links = append(currNode.Links, u.Int(row[1]))
			currNode.Type = append(currNode.Type, u.Int(row[2]))
		}
	}

	return edges, nil

}

// InitSpeaker initializes the Speaker associated with a certain Node
func InitSpeaker(node *Node) Speaker {
	return Speaker{Fresh: []bool{true}, Destinations: []*Node{node}, Paths: [][]*Node{{node}}}
}

func (s *Speaker) hasRoute(dest *Node) int {
	for i, d := range s.Destinations {
		if d == dest {
			return i
		}
	}

	return -1
}

func (s *Speaker) addDestination(lastHop *Node, dest *Node, path []*Node) {
	path = append(path, lastHop)
	s.Fresh = append(s.Fresh, true)
	s.Destinations = append(s.Destinations, dest)
	s.Paths = append(s.Paths, path)
}

// Activate evolves the status of a speaker
func (g *Graph) Activate(nodeIndex int) {
	nd := g.Nodes[nodeIndex]
	sp := g.Speakers[nodeIndex]

	for i := 0; i < len(sp.Fresh); i++ {
		if sp.Fresh[i] {
			// Advertise change to neighbors
			for n, link := range nd.Links {
				switch nd.Type[n] {
				case ToCustomer:
					if routeNum := hasRoute; routeNum < 0 {
						// The speaker does not have the destination yet
						// TODO
					}
				}
			}
		}
	}
}

func main() {

	graph, err := LoadFromCsv("../../../simulation/202003-edges.csv")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	fmt.Printf("Nodes: %s", graph[:10])
	fmt.Printf("Egdes of AS#0: %s", graph[0].Links)
	fmt.Printf("Of type: %s", graph[0].Type)
}
