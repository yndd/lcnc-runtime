package meta

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func AddAnnotations(o metav1.Object, annotations map[string]string) {
	a := o.GetAnnotations()
	if a == nil {
		o.SetAnnotations(annotations)
		return
	}
	for k, v := range annotations {
		a[k] = v
	}
	o.SetAnnotations(a)
}

func RemoveAnnotations(o metav1.Object, annotations ...string) {
	a := o.GetAnnotations()
	if a == nil {
		return
	}
	for _, k := range annotations {
		delete(a, k)
	}
	o.SetAnnotations(a)
}
