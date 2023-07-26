package transaction

import (
	"log"
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1"
	"pggat2/lib/util/maths"
	"pggat2/lib/zap"
	"pggat2/lib/zap/zapbuf"
)

type Pool struct {
	s schedulers.Scheduler

	recipes map[string]*Recipe
	mu      sync.RWMutex
}

func NewPool() *Pool {
	pool := &Pool{
		s: schedulers.MakeScheduler(),
	}

	return pool
}

func (T *Pool) scaleUpRecipe(r *Recipe) bool {
	if len(r.open) >= r.recipe.GetMaxConnections() {
		return false
	}

	rw, err := r.recipe.Connect()
	if err != nil {
		// TODO(garet) do something here
		log.Printf("Failed to connect: %v", err)
		return false
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
		return false
	}
	sink := &Conn{
		rw:  mw,
		eqp: eqps,
		ps:  pss,
	}
	id := T.s.AddWorker(0, sink)
	r.open = append(r.open, id)
	return true
}

func (T *Pool) scaleDownRecipe(r *Recipe) bool {
	if len(r.open) <= r.recipe.GetMinConnections() {
		return false
	}

	if len(r.open) == 0 {
		// none to close
		return false
	}

	id := r.open[len(r.open)-1]
	conn := T.s.RemoveWorker(id).(*Conn)
	_ = conn.rw.Close()
	r.open = r.open[:len(r.open)-1]
	return true
}

func (T *Pool) recipeOpenCount(r *Recipe) int {
	j := 0
	for i := 0; i < len(r.open); i++ {
		if T.s.GetWorker(r.open[i]) != nil {
			r.open[j] = r.open[i]
			j++
		}
	}

	r.open = r.open[:j]
	return j
}

func (T *Pool) scaleRecipe(r *Recipe, target int) (remaining int) {
	if target > 0 {
		for i := 0; i < target; i++ {
			if !T.scaleUpRecipe(r) {
				break
			}
			target--
		}
	} else if target < 0 {
		for i := 0; i > target; i-- {
			if !T.scaleDownRecipe(r) {
				break
			}
			target++
		}
	}
	return target
}

func (T *Pool) scaleRecipeTo(r *Recipe, target int) bool {
	target = maths.Clamp(target, r.recipe.GetMinConnections(), r.recipe.GetMaxConnections())
	target -= T.recipeOpenCount(r)
	return T.scaleRecipe(r, target) == 0
}

func (T *Pool) addRecipe(r *Recipe) {
	r.mu.Lock()
	defer r.mu.Unlock()

	T.scaleRecipe(r, 0)
}

func (T *Pool) removeRecipe(r *Recipe) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for len(r.open) > 0 {
		T.scaleDownRecipe(r)
	}
}

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	r := NewRecipe(recipe)
	T.addRecipe(r)
	T.mu.Lock()
	old, ok := T.recipes[name]
	if T.recipes == nil {
		T.recipes = make(map[string]*Recipe)
	}
	T.recipes[name] = r
	T.mu.Unlock()
	if ok {
		T.removeRecipe(old)
	}
}

func (T *Pool) remove(id uuid.UUID) {
	T.s.RemoveWorker(id)
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	r, ok := T.recipes[name]
	T.mu.Unlock()
	if ok {
		T.removeRecipe(r)
	}
}

func (T *Pool) scale(amount int) bool {
	T.mu.RLock()
	for _, r := range T.recipes {
		T.mu.RUnlock()
		amount = T.scaleRecipe(r, amount)
		T.mu.RLock()
		if amount == 0 {
			break
		}
	}
	T.mu.RUnlock()

	return amount == 0
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
			break
		}
		source.Do(&ctx, Work{
			rw:  buffer,
			eqp: eqpc,
			ps:  psc,
		})
	}
	_ = client.Close()
}

func (T *Pool) ReadSchedulerMetrics(metrics *rob.Metrics) {
	T.s.ReadMetrics(metrics)
	avgUtil := metrics.AverageWorkerUtilization()
	workerCount := len(metrics.Workers)
	if avgUtil > 0.7 {
		T.scale(1)
	}
	if avgUtil < 0.7 && ((float64(workerCount)*avgUtil)/(float64(workerCount-1)*100.0) < 0.7 || avgUtil == 0) {
		T.scale(-1)
	}
}

var _ gat.Pool = (*Pool)(nil)
