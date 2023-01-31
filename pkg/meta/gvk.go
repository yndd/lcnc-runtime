package meta

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	emptyGvk  = "empty gvk"
	emptyKind = "empty kind in gvk"
)

func GVKToString(gvk *schema.GroupVersionKind) string {
	if gvk == nil {
		return emptyGvk
	}

	if gvk.Kind == "" {
		return emptyKind
	}
	var sb strings.Builder
	sb.WriteString(gvk.Kind)
	if gvk.Version != "" {
		sb.WriteString("." + gvk.Version)
	}
	if gvk.Group != "" {
		sb.WriteString("." + gvk.Group)
	}
	return sb.String()
}

func StringToGroupVersionKind(s string) (string, string, string) {
	if strings.Count(s, ".") >= 2 {
		s := strings.SplitN(s, ".", 3)
		return s[2], s[1], s[0]
	}
	return "", "", ""
}

func StringToGVK(s string) *schema.GroupVersionKind {
	group, version, kind := StringToGroupVersionKind(s)
	return &schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
}

func GetGVKFromObject(o client.Object) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   o.GetObjectKind().GroupVersionKind().Group,
		Version: o.GetObjectKind().GroupVersionKind().Version,
		Kind:    o.GetObjectKind().GroupVersionKind().Kind,
	}
}
