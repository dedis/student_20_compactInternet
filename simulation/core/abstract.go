package core

// AbstractGraph defines the interface of Graphs
type AbstractGraph interface {
	GetRoute(a int, b int) ([]*Node, []int)
	GetNodes() *map[int]*Node
	SetDestinations(dest map[int]bool)
	Evolve() int
	Copy() AbstractGraph
}
