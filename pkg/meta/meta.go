package meta

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WasDeleted(o metav1.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}
