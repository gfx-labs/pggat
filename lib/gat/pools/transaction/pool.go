package transaction

import (
	"log"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/maths"
	"pggat2/lib/zap"
	"pggat2/lib/zap/zapbuf"
)

type Pool struct {
	s schedulers.Scheduler

	recipes maps.RWLocked[string, *Recipe]
}

func NewPool() *Pool {
	pool := &Pool{
		s: schedulers.MakeScheduler(),
	}

	return pool
}

func (T *Pool) openOne(r *Recipe) {
	rw, err := r.recipe.Connect()
	if err != nil {
		// TODO(garet) do something here
		log.Printf("Failed to connect: %v", err)
		return
	}
	eqps := eqp.NewServer()
	pss := ps.NewServer()
	mw := interceptor.NewInterceptor(
		rw,
		eqps,
		pss,
	)
	err2 := backends.Accept(mw, r.recipe.GetUser(), r.recipe.GetPassword(), r.recipe.GetDatabase())
	if err2 != nil {
		_ = rw.Close()
		// TODO(garet) do something here
		log.Printf("Failed to connect: %v", err2)
		return
	}
	sink := &Conn{
		rw:  mw,
		eqp: eqps,
		ps:  pss,
	}
	id := T.s.AddSink(0, sink)
	r.open = append(r.open, id)
}

func (T *Pool) closeOne(r *Recipe) {
	if len(r.open) == 0 {
		// none to close
		return
	}

	id := r.open[len(r.open)-1]
	conn := T.s.RemoveSink(id).(*Conn)
	_ = conn.rw.Close()
	r.open = r.open[:len(r.open)-1]
}

func (T *Pool) openCount(r *Recipe) int {
	j := 0
	for i := 0; i < len(r.open); i++ {
		if T.s.GetSink(r.open[i]) != nil {
			r.open[j] = r.open[i]
			j++
		}
	}

	r.open = r.open[:j]
	return j
}

func (T *Pool) scale(r *Recipe, target int) {
	target = maths.Clamp(target, r.recipe.GetMinConnections(), r.recipe.GetMaxConnections())

	target -= T.openCount(r)

	if target > 0 {
		for i := 0; i < target; i++ {
			T.openOne(r)
		}
	} else if target < 0 {
		for i := 0; i > target; i-- {
			T.closeOne(r)
		}
	}
}

func (T *Pool) addRecipe(r *Recipe) {
	r.mu.Lock()
	defer r.mu.Unlock()

	T.scale(r, 0)
}

func (T *Pool) removeRecipe(r *Recipe) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for len(r.open) > 0 {
		T.closeOne(r)
	}
}

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	r := NewRecipe(recipe)
	T.addRecipe(r)
	if old, ok := T.recipes.Swap(name, r); ok {
		T.removeRecipe(old)
	}
}

func (T *Pool) remove(id uuid.UUID) {
	T.s.RemoveSink(id)
}

func (T *Pool) RemoveRecipe(name string) {
	if r, ok := T.recipes.LoadAndDelete(name); ok {
		T.removeRecipe(r)
	}
}

func (T *Pool) Serve(client zap.ReadWriter) {
	source := T.s.NewSource()
	eqpc := eqp.NewClient()
	defer eqpc.Done()
	psc := ps.NewClient()
	client = interceptor.NewInterceptor(
		client,
		eqpc,
		psc,
	)
	buffer := zapbuf.NewBuffer(client)
	defer buffer.Done()
	var ctx rob.Context
	for {
		if err := buffer.Buffer(); err != nil {
			_ = client.Close()
			break
		}
		source.Do(&ctx, Work{
			rw:  buffer,
			eqp: eqpc,
			ps:  psc,
		})
	}
}

func (T *Pool) ReadSchedulerMetrics(metrics *rob.Metrics) {
	T.s.ReadMetrics(metrics)
}

var _ gat.Pool = (*Pool)(nil)
