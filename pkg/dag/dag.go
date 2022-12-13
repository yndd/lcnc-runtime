package dag

import (
	"fmt"
	"os"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type DAG interface {
	//GetParentDag() DAG
	AddVertex(s string, v *VertexContext) error
	Connect(from, to string)
	AddDownEdge(from, to string)
	AddUpEdge(from, to string)
	VertexExists(s string) bool
	GetRootVertex() string
	GetVertex(s string) *VertexContext
	GetVertices() map[string]*VertexContext
	GetDownVertexes(from string) []string
	GetUpVertexes(from string) []string

	GetDependencyMap(from string)
	// Walk(ctx context.Context, from string)
	// GetWalkResult()
	TransitiveReduction()
	// used for the resolution
	//Lookup(s []string) bool
	Lookup(s []string) bool
	// used for the edge connectivity
	LookupRootVertex(s []string) (string, error)
	lookupRootVertex(idx int, s []string) (int, string, error)
}

// used for returning
type Edge struct {
	From string
	To   string
}

type dag struct {
	//dagCtx *DagContext
	// vertices first key is the vertexName
	mv       sync.RWMutex
	vertices map[string]*VertexContext
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

type VertexKind string

const (
	RootVertexKind VertexKind = "root"
	//BlockVertexKind    VertexKind = "block"
	FunctionVertexKind VertexKind = "function"
	OutputVertexKind   VertexKind = "output"
	LocalVarVertexKind VertexKind = "localvar"
)

type VertexContext struct {
	// block indicates we have to execute the pipeline or not
	Name        string
	Kind        VertexKind
	OutputDAG   DAG
	LocalVarDag DAG
	// if it is a block we execute the block and call the dag
	//ControllerConfigBlock ctrlcfgv1.ControllerConfigBlock
	//ChildDAG              DAG
	// if is not a block we execute the function with the functionconfig
	// root is modelled as a function
	Function *ctrlcfgv1.ControllerConfigFunction
}

//type DagContext struct {
//	ParentDag DAG
//}

func New() DAG {
	return &dag{
		//dagCtx:    dagCtx,
		vertices:  make(map[string]*VertexContext),
		downEdges: make(map[string]map[string]struct{}),
		upEdges:   make(map[string]map[string]struct{}),
	}
}

/*
func (r *dag) GetParentDag() DAG {
	return r.dagCtx.ParentDag
}
*/

func (r *dag) AddVertex(s string, v *VertexContext) error {
	r.mv.Lock()
	defer r.mv.Unlock()

	// validate duplicate entry
	if _, ok := r.vertices[s]; ok {
		return fmt.Errorf("duplicate vertex entry: %s", s)
	}
	/*
		if r.dagCtx.ParentDag == nil {
			if len(r.GetVertices()) > 0 {
				return fmt.Errorf("we can only have 1 root dag")
			}
		}
	*/
	r.vertices[s] = v

	return nil
}

func (r *dag) GetVertices() map[string]*VertexContext {
	r.mv.RLock()
	defer r.mv.RUnlock()
	return r.vertices
}

func (r *dag) VertexExists(s string) bool {
	r.mv.RLock()
	defer r.mv.RUnlock()
	_, ok := r.vertices[s]
	return ok
}

func (r *dag) GetVertex(s string) *VertexContext {
	r.mv.RLock()
	defer r.mv.RUnlock()
	return r.vertices[s]
}

func (r *dag) GetRootVertex() string {
	for vertexName, v := range r.GetVertices() {
		if v.Kind == RootVertexKind {
			return vertexName
		}
	}
	return ""
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
	for vertexName := range r.GetVertices() {
		if vertexName == s {
			return true
		}
	}
	return false
}

func (r *dag) Lookup(s []string) bool {
	// we hit the root of the tree
	if len(s) == 0 {
		// should never happen with our logic since there is a check for len
		return false
	}
	v := r.GetVertex(s[0])
	if v == nil {
		return false
	}
	if len(s) == 1 {
		return true
	}
	if v.OutputDAG != nil {
		return v.OutputDAG.Lookup(s[1:])
	}
	return false
}

func (r *dag) LookupRootVertex(s []string) (string, error) {
	// we hit the root of the tree
	if len(s) == 0 {
		// should never happen with our logic since there is a check for len
		return "", fmt.Errorf("lookup root vertex should always have some input: %v", s)
	}
	_, vertexName, err := r.lookupRootVertex(1, s)
	if err != nil {
		return "", err
	}
	return vertexName, nil

}

func (r *dag) lookupRootVertex(idx int, s []string) (int, string, error) {
	v := r.GetVertex(s[0])
	if v == nil {
		return idx, "", fmt.Errorf("lookup root vertex not found: %v", s)
	}
	if len(s) == idx {
		if idx == 1 {
			return idx, s[idx-1], nil
		}
		if idx == 2 {
			return idx, s[idx-2], nil
		}
		
	}
	if v.OutputDAG != nil {
		idx++
		return v.OutputDAG.lookupRootVertex(idx, s)
	}
	return idx, "", fmt.Errorf("lookup root vertex not found: %v", s)
}
