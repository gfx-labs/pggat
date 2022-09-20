package cmux

import (
	"strings"
	"sync"

	"github.com/looplab/fsm"
)

type Mux[T any] interface {
	Register([]string, func([]string) T)
	Call([]string) T
}

type funcSet[T any] struct {
	Ref  []string
	Call func([]string) T
}

type FsmMux[T any] struct {
	f     *fsm.FSM
	funcs map[string]funcSet[T]

	sync.RWMutex
}

func (f *FsmMux[T]) Register(path []string, fn func([]string) T) {
	execkey := strings.Join(path, "|")
	f.funcs[execkey] = funcSet[T]{
		Ref:  path,
		Call: fn,
	}
	f.construct()
}

func (f *FsmMux[T]) construct() {
	evts := fsm.Events{}
	cbs := fsm.Callbacks{}
	for _, fset := range f.funcs {
		path := fset.Ref
		lp := len(path)
		switch lp {
		case 0:
		case 1:
			evts = append(evts, fsm.EventDesc{
				Name: path[0],
				Src:  []string{"_"},
				Dst:  path[0],
			})
		default:
			evts = append(evts, fsm.EventDesc{
				Name: path[0],
				Src:  []string{"_"},
				Dst:  path[0],
			})
			for i := 1; i < len(path); i++ {
				ee := fsm.EventDesc{
					Name: path[i],
					Src:  []string{path[i-1]},
					Dst:  path[i],
				}
				evts = append(evts, ee)
			}
		}
	}
	f.f = fsm.NewFSM("_", evts, cbs)
}

func (f *FsmMux[T]) Call(k []string) T {
	fn := f.funcs[""].Call
	args := k
	path := k
	lp := len(path)
	switch lp {
	case 0:
	case 1:
		args = args[1:]
		fn = f.funcs[k[0]].Call
	default:
		f.Lock()
		f.f.SetState("_")
		for i := 0; i < len(path); i++ {
			key := strings.Join(path[:i], "|")
			if mb, ok := f.funcs[key]; ok {
				fn = mb.Call
			}
			if f.f.Can(path[i]) {
				f.f.Event(path[i])
			} else {
				key := strings.Join(path[:i], "|")
				if _, ok := f.funcs[key]; ok {
					args = args[i:]
					break
				}
			}
		}
		f.Unlock()
	}
	return fn(args)
}

func NewFsmMux[T any]() Mux[T] {
	o := &FsmMux[T]{
		funcs: map[string]funcSet[T]{
			"": {
				Ref:  []string{},
				Call: func([]string) T { return *new(T) },
			},
		},
	}
	return o
}
