package dag

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type DAG interface {
	AddVertex(s string, v any) error
	Connect(from, to string)
	AddDownEdge(from, to string)
	AddUpEdge(from, to string)
	VertexExists(s string) bool
	GetVertex(s string) any
	GetVertices() map[string]any
	GetDownVertexes(from string) []string
	GetUpVertexes(from string) []string
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
	vertices map[string]any
	// downEdges/upEdges
	// 1st key is from, 2nd key is to
	mde       sync.RWMutex
	downEdges map[string]map[string]struct{}
	mue       sync.RWMutex
	upEdges   map[string]map[string]struct{}
	// used for transit reduction
	mvd         sync.RWMutex
	vertexDepth map[string]int
	// logging
	l logr.Logger
}

func New() DAG {
	return &dag{
		//dagCtx:    dagCtx,
		vertices:  make(map[string]any),
		downEdges: make(map[string]map[string]struct{}),
		upEdges:   make(map[string]map[string]struct{}),

		l: ctrl.Log.WithName("dag"),
	}
}

func (r *dag) AddVertex(s string, v any) error {
	r.mv.Lock()
	defer r.mv.Unlock()

	// validate duplicate entry
	if _, ok := r.vertices[s]; ok {
		return fmt.Errorf("duplicate vertex entry: %s", s)
	}
	r.vertices[s] = v

	return nil
}

func (r *dag) GetVertices() map[string]any {
	r.mv.RLock()
	defer r.mv.RUnlock()
	vcs := map[string]any{}
	for vertexName, v := range r.vertices {
		vcs[vertexName] = v
	}
	return vcs

}

func (r *dag) VertexExists(s string) bool {
	r.mv.RLock()
	defer r.mv.RUnlock()
	_, ok := r.vertices[s]
	return ok
}

func (r *dag) GetVertex(s string) any {
	r.mv.RLock()
	defer r.mv.RUnlock()
	return r.vertices[s]
}

func (r *dag) Connect(from, to string) {
	//fmt.Printf("connect dag: %s -> %s\n", to, from)
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
