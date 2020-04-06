package bgp

import (
	"strings"

	"dedis.epfl.ch/u"
)

// Speaker represents a (BGP, TZ, ...) speaker
type Speaker struct {
	Fresh        []bool
	Destinations []*Node
	NextHop      []*Node
	Length       []int
}

func (s *Speaker) String(n *Node) string {
	// TODO: Refine it
	var sb strings.Builder
	for idx, dest := range s.Destinations {
		sb.WriteString("	")
		nextHopType := s.getNextHopType(n, idx)

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

// InitSpeaker initializes the Speaker associated with a certain Node
func InitSpeaker(node *Node) *Speaker {
	// TODO: Clean method
	speaker := Speaker{Fresh: nil, Destinations: nil, NextHop: nil, Length: nil}
	return &speaker
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
	oldNextType := s.getNextHopType(currNode, routeIndex)
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

// This function assumes, for performance reasons, that the node n does NOT appear in path
func (s *Speaker) getNextHopType(n *Node, destIndex int) int {
	if s.Length[destIndex] == 0 {
		// The node *n is the origin, return ToCustomer
		return ToCustomer
	}
	return n.getNeighborType(s.NextHop[destIndex])
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
