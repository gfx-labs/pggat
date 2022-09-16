package cmux

import (
	"strings"
	"sync"

	"github.com/looplab/fsm"
)

type Mux[IN, OUT any] interface {
	Register([]string, func(IN, []string) OUT)
	Call(IN, []string) (OUT, bool)
}

type MapMux[IN, OUT any] struct {
	sub map[string]*MapMux[IN, OUT]
	fn  func(IN, []string) OUT
}

func NewMapMux[IN, OUT any]() *MapMux[IN, OUT] {
	return &MapMux[IN, OUT]{
		sub: make(map[string]*MapMux[IN, OUT]),
	}
}

func (m *MapMux[IN, OUT]) Register(path []string, fn func(IN, []string) OUT) {
	mux := m
	for {
		if len(path) == 0 {
			mux.fn = fn
			return
		}

		var ok bool
		if _, ok = mux.sub[path[0]]; !ok {
			mux.sub[path[0]] = NewMapMux[IN, OUT]()
		}
		mux = mux.sub[path[0]]
		path = path[1:]
	}
}

func (m *MapMux[IN, OUT]) Call(arg IN, path []string) (o OUT, exists bool) {
	mux := m
	for {
		if len(path) != 0 {
			if sub, ok := mux.sub[path[0]]; ok {
				mux = sub
				path = path[1:]
				continue
			}
		}

		if mux.fn != nil {
			o = mux.fn(arg, path)
			exists = true
		}
		return
	}
}

type funcSet[IN, OUT any] struct {
	Ref  []string
	Call func(IN, []string) OUT
}

type FsmMux[IN, OUT any] struct {
	f     *fsm.FSM
	funcs map[string]funcSet[IN, OUT]

	sync.RWMutex
}

func (f *FsmMux[IN, OUT]) Register(path []string, fn func(IN, []string) OUT) {
	execkey := strings.Join(path, "|")
	f.funcs[execkey] = funcSet[IN, OUT]{
		Ref:  path,
		Call: fn,
	}
	f.construct()
}

func (f *FsmMux[IN, OUT]) construct() {
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

func (f *FsmMux[IN, OUT]) Call(arg IN, k []string) (r OUT, matched bool) {
	var fn func(IN, []string) OUT
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
	if fn != nil {
		r = fn(arg, args)
		matched = true
	}
	return
}

func NewFsmMux[IN, OUT any]() Mux[IN, OUT] {
	o := &FsmMux[IN, OUT]{
		funcs: map[string]funcSet[IN, OUT]{},
	}
	return o
}
