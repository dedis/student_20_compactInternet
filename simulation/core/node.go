package core

import (
	"encoding/csv"
	"os"
	"strings"

	"dedis.epfl.ch/u"
)

// ToProvider specifies the link with a provider
const ToProvider int = 1

// ToPeer specifies the link with a peer
const ToPeer int = 0

// ToCustomer specifies the link with a customer
const ToCustomer int = (-1)

// Link represents and edge in AS graph
type Link []int

// GraphStructure represents the nodes and edges in AS graph
type GraphStructure map[int]*Node

func (l Link) String() string {
	var sb strings.Builder
	for i := 0; i < len(l); i++ {
		sb.WriteString("-> " + u.Str(l[i]))
	}
	return sb.String()
}

// LinkTypeToSymbol prints the type of link: Unicode alternative symbols: '🡕''🡒''🡖'
func LinkTypeToSymbol(linkType int) string {
	if linkType == 1 {
		return `^`
	} else if linkType == 0 {
		return `=`
	} else if linkType == -1 {
		return `v`
	} else {
		return ">"
	}
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

// CanTellAbout enforces Gao-Rexford rules, determining if the presence of
// a link between 'n' and 'subject' can be revealed to 'target'
func (n *Node) CanTellAbout(subject *Node, target *Node) bool {
	if n.Asn == subject.Asn {
		// Can always tell about itself
		return true
	}

	heardFromType := n.GetNeighborType(subject)
	advertisedToType := n.GetNeighborType(target)

	switch {
	// CUSTOMER: Advertise all routes
	case advertisedToType == ToCustomer:
		return true

	// PEER: Advertise routes from customers and peers
	case advertisedToType == ToPeer:
		return heardFromType == ToCustomer

	// PROVIDER: Advertise routes from customers
	case advertisedToType == ToProvider:
		return heardFromType == ToCustomer

	default:
		panic("Invalid link type")
	}
}

// GetNeighborType returns the type of the link connecting the node to a neighbor
func (n *Node) GetNeighborType(neighborNode *Node) int {
	linkIndex := n.Links.search(neighborNode.Asn)
	return n.Type[linkIndex]
}

// GetNeighborIndex returns the index of the neighbor in the list or -1 (if it's absent)
func (n *Node) GetNeighborIndex(neighborNode *Node) int {
	return n.Links.searchOrDefault(neighborNode.Asn)
}

// DeleteLink removes an edge (if it exists) and that does not disconnect the node
// from the rest of the graph. In that case it returns false
func (n *Node) DeleteLink(neighborNode *Node) bool {
	idx := n.GetNeighborIndex(neighborNode)
	if idx < 0 {
		// The link does not exist
		return false
	}

	linksNum := len(n.Links)

	if linksNum > 1 {
		for c := idx; c < linksNum-1; c++ {
			n.Links[c] = n.Links[c+1]
			n.Type[c] = n.Type[c+1]
		}

		// Delete from links
		n.Links[linksNum-1] = 0
		n.Links = n.Links[:(linksNum - 1)]

		//Delete from types
		n.Type[linksNum-1] = 0
		n.Type = n.Type[:(linksNum - 1)]

		return true
	} else {
		return false
	}
}

func (l *Link) searchOrDefault(target int) int {
	slice := (*l)[:]

	var global int = 0

	for {
		if len(slice) == 0 {
			return -1
		}
		if len(slice) == 1 && slice[0] != target {
			return -1
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

func (l *Link) search(target int) int {
	if idx := l.searchOrDefault(target); idx == -1 {
		panic("Element " + u.Str(target) + " not found in " + (*l)[:].String())
	} else {
		return idx
	}
}

// Copy returns a new Graph
func (n *Node) Copy() *Node {
	copyNode := Node{
		Asn:   n.Asn,
		Links: make(Link, len(n.Links)),
		Type:  make(Rel, len(n.Type)),
	}

	copy(copyNode.Links, n.Links)
	copy(copyNode.Type, n.Type)

	return &copyNode
}

// Serialize implements the interface Serializable for *Node
func (n *Node) Serialize() [][]string {
	stream := make([][]string, 0, len(n.Links))
	for idx := 0; idx < len(n.Links); idx++ {
		stream = append(stream, []string{u.Str(n.Asn), u.Str(n.Links[idx]), u.Str(n.Type[idx])})
	}
	return stream
}

// WriteStructureToCsv saves the graph structure to a CSV files
func (nodes GraphStructure) WriteStructureToCsv(filename string) {
	csvFile, err := os.Create(filename)
	if err != nil {
		panic("Unable to open the file")
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	for _, n := range nodes {
		writer.WriteAll(n.Serialize())
	}

	writer.Flush()
}
