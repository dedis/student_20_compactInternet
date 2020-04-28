package core

// AbstractGraph defines the interface of Graphs
type AbstractGraph interface {
	GetRoute(a int, b int) ([]*Node, []int)
	GetNodes() *map[int]*Node
	CountLinks() int
	DeleteDestination(dest int)
	SetDestinations(dest map[int]bool)
	Evolve() int
	RemoveEdge(a int, b int) (bool, int)
	Copy() AbstractGraph
}

// Serializable represents an object that can be transferred to file
type Serializable interface {
	Serialize(index int) [][]string
}
