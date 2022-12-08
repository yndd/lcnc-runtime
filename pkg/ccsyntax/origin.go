package ccsyntax

type OriginContext struct {
	Index      int
	VertexName string
	Origin     Origin
	ForBlock   bool
	Query      bool
	Input      bool
	Output     bool
}

type Origin string

const (
	OriginInvalid  Origin = "invalid"
	OriginFor      Origin = "for"
	OriginOwn      Origin = "own"
	OriginWatch    Origin = "watch"
	OriginVariable Origin = "vars"
	OriginFunction Origin = "functions"
	OriginService  Origin = "services"
)
