package ccsyntax

type ReferenceKind string

const (
	RangeReferenceKind   ReferenceKind = "range" // used for $KEY, $VALUE, $INDEX
	RegularReferenceKind ReferenceKind = "regular"
)

type Reference struct {
	Kind  ReferenceKind
	Value string
}

func GetReferences(s string) []*Reference {
	var idx int
	refs := make([]*Reference, 0)
	for k, v := range s {
		if v == '$' && idx == 0 {
			idx = k
			continue
		}
		if v == '.' && idx > 0 {
			refs = append(refs, &Reference{
				Kind:  RegularReferenceKind,
				Value: s[idx+1 : k]})
			idx = 0
		}
	}
	// if no . was found we store the complete value as a refrence value
	if idx > 0 {
		refs = append(refs, &Reference{
			Kind:  RegularReferenceKind,
			Value: s[idx+1:]})
	}

	// TODO handle VALUE, KEY, INDEX

	return refs
}
