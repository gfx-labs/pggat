package race

import (
	"gfx.cafe/util/go/generic"
	"reflect"
)

var casePool = generic.HookPool[[]reflect.SelectCase]{
	New: func() []reflect.SelectCase {
		return nil
	},
}
