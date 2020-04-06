package main

import (
	"strings"

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

// Unicode alternative symbols: '🡕''🡒''🡖'
func linkTypeToSymbol(linkType int) string {
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

func (n *Node) getNeighborType(neighborNode *Node) int {
	linkIndex := n.Links.search(neighborNode.Asn)
	return n.Type[linkIndex]
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
