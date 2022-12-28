package functions

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
)

func Init(c *fnmap.Config) fnmap.FuncMap {
	fnMap := fnmap.New(c)
	fnMap.Register(ctrlcfgv1.RootType, func() fnmap.Function {
		return NewRootFn()
	})
	fnMap.Register(ctrlcfgv1.BlockType, func() fnmap.Function {
		return NewBlockFn()
	})
	fnMap.Register(ctrlcfgv1.SliceType, func() fnmap.Function {
		return NewSliceFn()
	})
	fnMap.Register(ctrlcfgv1.MapType, func() fnmap.Function {
		return NewMapFn()
	})
	fnMap.Register(ctrlcfgv1.QueryType, func() fnmap.Function {
		return NewQueryFn()
	})
	fnMap.Register(ctrlcfgv1.GoTemplateType, func() fnmap.Function {
		return NewGTFn()
	})
	fnMap.Register(ctrlcfgv1.JQType, func() fnmap.Function {
		return NewJQFn()
	})
	fnMap.Register(ctrlcfgv1.WasmType, func() fnmap.Function {
		return NewImageFn()
	})
	fnMap.Register(ctrlcfgv1.ContainerType, func() fnmap.Function {
		return NewImageFn()
	})
	return fnMap
}
