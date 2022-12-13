package ccsyntax

type Result struct {
	OriginContext *OriginContext `json:"inline" yaml:"inline"`
	Error         string         `json:"error,omitempty" yaml:"error,omitempty"`
}

type recordResultFn func(Result)

type OriginContext struct {
	//Index      int
	FOW        Fow    `json:"fow,omitempty" yaml:"fow,omitempty"`
	Pipeline   string `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Origin     Origin `json:"origin,omitempty" yaml:"origin,omitempty"`
	VertexName string `json:"vertexname,omitempty" yaml:"vertexname,omitempty"`
	LocalVars  string `json:"localvarName,omitempty" yaml:"localvarName,omitempty"`
}

type Fow string

const (
	FOWFor   Fow = "for"
	FOWOwn   Fow = "own"
	FOWWatch Fow = "watch"
)

type Origin string

const (
	OriginInvalid  Origin = "invalid"
	OriginFow      Origin = "fow"
	OriginVariable Origin = "vars"
	OriginFunction Origin = "function"
	OriginService  Origin = "services"
)
