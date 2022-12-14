package ccsyntax

import "fmt"

type ReferenceKind string

const (
	RangeReferenceKind   ReferenceKind = "range" // used for $KEY, $VALUE, $INDEX
	RegularReferenceKind ReferenceKind = "regular"
)

const (
	ValueKey = "VALUE"
	IndexKey = "INDEX"
	KeyKey   = "KEY"
)

type Reference struct {
	Kind  ReferenceKind
	Value string
}

type References interface {
	GetReferences(s string) []*Reference
	//addReference(s string)
}

type references struct {
	refs []*Reference
}

func NewReferences() References {
	return &references{
		refs: make([]*Reference, 0),
	}
}

func (r *references) GetReferences(s string) []*Reference {
	var idx int
	var found bool
	for k, v := range s {
		if v == '$' {
			found = true
			idx = k
			continue
		}
		if (v == '.' || v == ' ' || v == ')') && found {
			r.addReference(s[idx+1 : k])
			found = false
		}
	}
	// if no . was found we store the complete value as a refrence value
	if found {
		r.addReference(s[idx+1:])
	}

	return r.refs
}

func (r *references) addReference(val string) {
	if val == ValueKey || val == KeyKey || val == IndexKey {
		r.refs = append(r.refs, &Reference{
			Kind:  RangeReferenceKind,
			Value: val,
		})
	} else {
		r.refs = append(r.refs, &Reference{
			Kind:  RegularReferenceKind,
			Value: val,
		})
	}
}

func (r *references) Print() {
	for _, ref := range r.refs {
		fmt.Printf("refs: %s kind: %s\n", ref.Value, string(ref.Kind))
	}
}
