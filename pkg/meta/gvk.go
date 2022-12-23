package meta

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GVKToString(gvk *schema.GroupVersionKind) string {
	return fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group)
}

func StringToGVK(s string) *schema.GroupVersionKind {
	var gvk *schema.GroupVersionKind
	if strings.Count(s, ".") >= 2 {
		s := strings.SplitN(s, ".", 3)
		gvk = &schema.GroupVersionKind{Group: s[2], Version: s[1], Kind: s[0]}
	}
	return gvk
}
