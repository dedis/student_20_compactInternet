package tz

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"

	. "dedis.epfl.ch/core"
	"dedis.epfl.ch/u"
)

// Landmarks models the set of samples A_i (0 <= i < k)
type Landmarks map[int]map[*Node]bool

// Serialize produces a representation of Landmarks suitable to be saved to file
func (l *Landmarks) Serialize(index int) [][]string {

	content := make([][]string, 0, len(*l))

	for level := 0; level < len(*l); level++ {
		for ld := range (*l)[level] {
			content = append(content, []string{u.Str(level), u.Str(ld.Asn)})
		}
	}

	return content
}

const (
	RandomStrategy   = 0
	SplineStrategy   = 1
	HarmonicStrategy = 2
	ImmunityStrategy = 3
)

// Filter some nodes to find the landmarks of a certain level
func (l *Landmarks) filterByLevel(candidates map[int]*Node, level int) map[int]*Node {

	candidatesInLevel := make(map[int]*Node)

	for _, cand := range candidates {
		if _, isInLevel := (*l)[level][cand]; isInLevel {
			candidatesInLevel[cand.Asn] = cand
		}
	}

	return candidatesInLevel
}

// Copy returns a duplicate of Landmarks
func (l *Landmarks) Copy(nodes *map[int]*Node) *Landmarks {
	copyLandmarks := make(Landmarks)

	for r, ld := range *l {
		copyLandmarks[r] = make(map[*Node]bool)
		for n := range ld {
			copyLandmarks[r][(*nodes)[n.Asn]] = true
		}
	}

	return &copyLandmarks
}

func (g *Graph) randomStrategy() {
	// Put all the nodes in A_0
	g.Landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		g.Landmarks[0][v] = true
	}

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1./float64(g.K))

	for i := 1; i < g.K; i++ {
		g.Landmarks[i] = make(map[*Node]bool)
		for key := range g.Landmarks[i-1] {
			extraction := rand.Float64()
			if extraction <= selProbability {
				g.Landmarks[i][key] = true
			}
		}
	}

	g.Landmarks[g.K] = nil
}

func (g *Graph) splineStrategy() {

	fmt.Println("Loading landmark hirerarchy...")

	csvFile, err := os.Open("data/2020-as-hierarchy.csv")
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1./float64(g.K))

	updatedProbs := make(map[int]float64)

	reader := csv.NewReader(csvFile)

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if i < 1000 {
			updatedProbs[u.Int(row[0])] = ((-0.02)*float64(i) + 30) / 4
		} else if i < 20000 {
			updatedProbs[u.Int(row[0])] = ((0.0005105)*float64(i) + 9.49) / 4
		} else {
			updatedProbs[u.Int(row[0])] = .05
		}

	}

	// Pick landmarks
	g.Landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		g.Landmarks[0][v] = true
	}

	for i := 1; i < g.K; i++ {
		g.Landmarks[i] = make(map[*Node]bool)

		for n := range g.Landmarks[i-1] {
			extraction := rand.Float64()
			if extraction < updatedProbs[n.Asn]*selProbability {
				g.Landmarks[i][n] = true
			}
		}
	}

	g.Landmarks[g.K] = nil
}

func (g *Graph) harmonicStrategy() {

	fmt.Println("Loading landmark hirerarchy...")

	csvFile, err := os.Open("data/2020-as-hierarchy.csv")
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1./float64(g.K))

	updatedProbs := make(map[int]float64)

	reader := csv.NewReader(csvFile)

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if i < 216 {
			updatedProbs[u.Int(row[0])] = 36 - 0.0000001*float64(i*i) - 0.000001*float64(i*i*i)
		} else {
			updatedProbs[u.Int(row[0])] = 5600.0 / float64(i)
		}

	}

	// Pick landmarks
	g.Landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		g.Landmarks[0][v] = true
	}

	for i := 1; i < g.K; i++ {
		g.Landmarks[i] = make(map[*Node]bool)

		for n := range g.Landmarks[i-1] {
			extraction := rand.Float64()
			if extraction < math.Pow(updatedProbs[n.Asn], 1/float64(i))*selProbability {
				g.Landmarks[i][n] = true
			}
		}
	}

	g.Landmarks[g.K] = nil
}

func (g *Graph) immunityStrategy() {
	g.Landmarks[0] = make(map[*Node]bool)
	for _, v := range g.Nodes {
		g.Landmarks[0][v] = true
	}

	fmt.Println("Loading landmark hirerarchy...")

	csvFile, err := os.Open("data/2020-as-hierarchy.csv")
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	var selProbability float64 = math.Pow(float64(len(g.Nodes)), -1./float64(g.K))

	updatedProbs := make(map[int]float64)

	reader := csv.NewReader(csvFile)

	for i := 0; ; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		updatedProbs[u.Int(row[0])] = 10000.0 / float64(i+100)

	}

	for i := 1; i < g.K; i++ {
		g.Landmarks[i] = make(map[*Node]bool)

		for n := range g.Landmarks[i-1] {
			extraction := rand.Float64()
			if extraction < math.Pow(updatedProbs[n.Asn], 1/float64(i))*selProbability {
				g.Landmarks[i][n] = true

				for _, neighbor := range n.Links {
					updatedProbs[neighbor] = 0
				}
			}
		}
	}

	g.Landmarks[g.K] = nil
}
