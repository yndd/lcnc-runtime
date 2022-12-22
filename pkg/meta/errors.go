package meta

import (
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	errUpdateObject = "cannot update k8s resource"
)

type ErrorIs func(err error) bool

func Ignore(is ErrorIs, err error) error {
	if is(err) {
		return nil
	}
	return err
}

// IgnoreNotFound returns the supplied error, or nil if the error indicates a
// Kubernetes resource was not found.
func IgnoreNotFound(err error) error {
	return Ignore(errors.IsNotFound, err)
}
