package dag

import (
	"fmt"
)

type WalkInitFn func()

type WalkEntryFn func(from string, depth int)

type WalkConfig struct {
	WalkInitFn  WalkInitFn
	WalkEntryFn WalkEntryFn
}

func (r *dag) TransitiveReduction() {
	for vertexName := range r.GetVertices() {
		//fmt.Printf("##### TRANSIT REDUCTION VERTEX START: %s ###############\n", vertexName)
		// we initialize the vertexdeptch map as 1 since 0 is used for uninitialized verteces
		// 0 is also used to avoid adding the vertex back in the graph
		// walk used to intialize the vertex depth map
		r.trwalk(vertexName, true, 1, WalkConfig{
			WalkInitFn:  r.initVertexDepthMap,
			WalkEntryFn: r.addVertexDepth,
		})

		// walk to execute the transit reduction
		r.trwalk(vertexName, true, 1, WalkConfig{
			WalkEntryFn: r.processTransitiveReducation,
		})
		//fmt.Printf("##### TRANSIT REDUCTION VERTEX ENDED: %s ###############\n", vertexName)
	}
}

func (r *dag) trwalk(from string, init bool, depth int, wc WalkConfig) {
	//wg := new(sync.WaitGroup)
	if wc.WalkInitFn != nil {
		if init {
			wc.WalkInitFn()
		}
	}
	if wc.WalkEntryFn != nil {
		wc.WalkEntryFn(from, depth)
	}
	// continue walking the graph
	downEdges := r.GetDownVertexes(from)
	if len(downEdges) == 0 {
		return
	}
	// increment the depth
	depth++
	for _, downEdge := range downEdges {
		r.trwalk(downEdge, false, depth, wc)
	}
}

func (r *dag) processTransitiveReducation(from string, depth int) {
	//fmt.Printf("from: %s, upVerteces: %v\n", from, r.GetUpVertexes(from))
	bestVertexDepth := r.getbestVertexDepth(from)
	//fmt.Printf("from: %s, bestVertexDepth: %v\n", from, bestVertexDepth)
	for _, upVertex := range r.GetUpVertexes(from) {
		// if bestVertexDepth == 0 it means we refer to an uninitialized vertex and we dont need
		// to process this.
		if bestVertexDepth != 0 {
			// if an upvertex has a depth of 0 it should not be considered
			// delete the edges for links that have different vertexDepths
			if r.getVertexDepth(upVertex) != 0 && r.getVertexDepth(upVertex) != bestVertexDepth {
				fmt.Printf("transitive reduction %s -> %s\n", upVertex, from)
				r.Disconnect(upVertex, from)
			}
		}
	}
}

func (r *dag) initVertexDepthMap() {
	r.mvd.Lock()
	defer r.mvd.Unlock()
	r.vertexDepth = map[string]int{}
}

func (r *dag) getVertexDepth(n string) int {
	r.mvd.RLock()
	defer r.mvd.RUnlock()
	if depth, ok := r.vertexDepth[n]; ok {
		return depth
	}
	return 0
}

func (r *dag) addVertexDepth(n string, depth int) {
	r.mvd.Lock()
	defer r.mvd.Unlock()
	r.vertexDepth[n] = depth
}

func (r *dag) getbestVertexDepth(from string) int {
	bestVertexDepth := 0
	for _, upVertex := range r.GetUpVertexes(from) {
		upVertexDepth := r.getVertexDepth(upVertex)
		if upVertexDepth > bestVertexDepth {
			bestVertexDepth = upVertexDepth
		}
	}
	return bestVertexDepth
}
