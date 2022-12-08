package dag

import (
	"fmt"
	"os"
	"sync"
)

type DAG interface {
	AddVertex(s string, v interface{}) error
	Connect(from, to string)
	AddDownEdge(from, to string)
	AddUpEdge(from, to string)
	GetVertex(s string) bool
	GetVertices() map[string]interface{}
	GetDownVertexes(from string) []string
	GetUpVertexes(from string) []string

	GetDependencyMap(from string)
	// Walk(ctx context.Context, from string)
	// GetWalkResult()
	TransitiveReduction()
}

// used for returning
type Edge struct {
	From string
	To   string
}

type dag struct {
	// vertices first key is the vertexName
	mv       sync.RWMutex
	vertices map[string]interface{}
	// downEdges/upEdges
	// 1st key is from, 2nd key is to
	mde       sync.RWMutex
	downEdges map[string]map[string]struct{}
	mue       sync.RWMutex
	upEdges   map[string]map[string]struct{}
	// used for transit reduction
	mvd         sync.RWMutex
	vertexDepth map[string]int
}

func NewDAG() DAG {
	return &dag{
		vertices:  make(map[string]interface{}),
		downEdges: make(map[string]map[string]struct{}),
		upEdges:   make(map[string]map[string]struct{}),
	}
}

func (r *dag) AddVertex(s string, v interface{}) error {
	r.mv.Lock()
	defer r.mv.Unlock()

	// validate duplicate entry
	if _, ok := r.vertices[s]; ok {
		return fmt.Errorf("duplicate vertex entry: %s", s)
	}
	r.vertices[s] = v
	return nil
}

func (r *dag) GetVertices() map[string]interface{} {
	r.mv.RLock()
	defer r.mv.RUnlock()
	return r.vertices
}

func (r *dag) GetVertex(s string) bool {
	r.mv.RLock()
	defer r.mv.RUnlock()
	_, ok := r.vertices[s]
	return ok
}

func (r *dag) Connect(from, to string) {
	r.AddDownEdge(from, to)
	r.AddUpEdge(to, from)
}

func (r *dag) Disconnect(from, to string) {
	r.DeleteDownEdge(from, to)
	r.DeleteUpEdge(to, from)
}

func (r *dag) AddDownEdge(from, to string) {
	r.mde.Lock()
	defer r.mde.Unlock()

	//fmt.Printf("addDownEdge: from: %s, to: %s\n", from, to)

	// initialize the from entry if it does not exist
	if _, ok := r.downEdges[from]; !ok {
		r.downEdges[from] = make(map[string]struct{})
	}
	if _, ok := r.downEdges[from][to]; ok {
		//  down edge entry already exists
		return
	}
	// add entry
	r.downEdges[from][to] = struct{}{}
}

func (r *dag) DeleteDownEdge(from, to string) {
	r.mde.Lock()
	defer r.mde.Unlock()

	//fmt.Printf("deleteDownEdge: from: %s, to: %s\n", from, to)
	if de, ok := r.downEdges[from]; ok {
		if _, ok := r.downEdges[from][to]; ok {
			delete(de, to)
		}
	}
}

func (r *dag) GetDownVertexes(from string) []string {
	r.mde.RLock()
	defer r.mde.RUnlock()

	edges := make([]string, 0)
	if fromVertex, ok := r.downEdges[from]; ok {
		for to := range fromVertex {
			edges = append(edges, to)
		}
	}
	return edges
}

func (r *dag) AddUpEdge(from, to string) {
	r.mue.Lock()
	defer r.mue.Unlock()

	//fmt.Printf("addUpEdge: from: %s, to: %s\n", from, to)

	// initialize the from entry if it does not exist
	if _, ok := r.upEdges[from]; !ok {
		r.upEdges[from] = make(map[string]struct{})
	}
	if _, ok := r.upEdges[from][to]; ok {
		//  up edge entry already exists
		return
	}
	// add entry
	r.upEdges[from][to] = struct{}{}
}

func (r *dag) DeleteUpEdge(from, to string) {
	r.mue.Lock()
	defer r.mue.Unlock()

	//fmt.Printf("deleteUpEdge: from: %s, to: %s\n", from, to)
	if ue, ok := r.upEdges[from]; ok {
		if _, ok := r.upEdges[from][to]; ok {
			delete(ue, to)
		}
	}
}

func (r *dag) GetUpEdges(from string) []Edge {
	r.mue.RLock()
	defer r.mue.RUnlock()

	edges := make([]Edge, 0)
	if fromVertex, ok := r.upEdges[from]; ok {
		for to := range fromVertex {
			edges = append(edges, Edge{From: from, To: to})
		}
	}
	return edges
}

func (r *dag) GetUpVertexes(from string) []string {
	r.mue.RLock()
	defer r.mue.RUnlock()

	upVerteces := []string{}
	if fromVertex, ok := r.upEdges[from]; ok {
		for to := range fromVertex {
			upVerteces = append(upVerteces, to)
		}
	}
	return upVerteces
}

func (r *dag) GetDependencyMap(from string) {
	fmt.Println("######### dependency map verteces start ###########")
	for vertexName := range r.GetVertices() {
		fmt.Printf("%s\n", vertexName)
	}
	fmt.Println("######### dependency map verteces end ###########")
	fmt.Println("######### dependency map start ###########")
	r.getDependencyMap(from, 0)
	fmt.Println("######### dependency map end   ###########")
}

func (r *dag) getDependencyMap(from string, indent int) {
	fmt.Printf("%s:\n", from)
	for _, upVertex := range r.GetUpVertexes(from) {
		found := r.checkVertex(upVertex)
		if !found {
			fmt.Printf("upVertex %s no found in vertices\n", upVertex)
			os.Exit(1)
		}
		fmt.Printf("-> %s\n", upVertex)
	}
	indent++
	for _, downVertex := range r.GetDownVertexes(from) {
		found := r.checkVertex(downVertex)
		if !found {
			fmt.Printf("upVertex %s no found in vertices\n", downVertex)
			os.Exit(1)
		}
		r.getDependencyMap(downVertex, indent)
	}
}

func (r *dag) checkVertex(s string) bool {
	found := false
	for vertexName := range r.GetVertices() {
		if vertexName == s {
			return true
		}
	}
	return found
}
