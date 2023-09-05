package pool

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/util/maps"
)

type State int

const (
	StateActive State = iota
	StateIdle
	StateAwaitingServer
	StateRunningResetQuery

	StateCount
)

func (T State) String() string {
	switch T {
	case StateActive:
		return "active"
	case StateIdle:
		return "idle"
	case StateAwaitingServer:
		return "awaiting server"
	case StateRunningResetQuery:
		return "running reset query"
	default:
		return "unknown state"
	}
}

type Metrics struct {
	Servers map[uuid.UUID]ItemMetrics
	Clients map[uuid.UUID]ItemMetrics
}

func (T *Metrics) TransactionCount() int {
	var serverTransactions int
	var clientTransactions int

	for _, server := range T.Servers {
		serverTransactions += server.Transactions
	}

	for _, client := range T.Clients {
		clientTransactions += client.Transactions
	}

	if clientTransactions > serverTransactions {
		return clientTransactions
	}
	return serverTransactions
}

func stateCount(items map[uuid.UUID]ItemMetrics) [StateCount]int {
	var states [StateCount]int
	for _, item := range items {
		states[item.State]++
	}
	return states
}

func stateUtil(items map[uuid.UUID]ItemMetrics) [StateCount]float64 {
	var util [StateCount]time.Duration
	var total time.Duration
	for _, item := range items {
		for state, amount := range item.InState {
			util[state] += amount
			total += amount
		}
	}

	var states [StateCount]float64
	for state := range states {
		states[state] = float64(util[state]) / float64(total)
	}

	return states
}

func (T *Metrics) ServerStateCount() [StateCount]int {
	return stateCount(T.Servers)
}

func (T *Metrics) ServerStateUtil() [StateCount]float64 {
	return stateUtil(T.Servers)
}

func (T *Metrics) ClientStateCount() [StateCount]int {
	return stateCount(T.Clients)
}

func (T *Metrics) ClientStateUtil() [StateCount]float64 {
	return stateUtil(T.Clients)
}

func (T *Metrics) Clear() {
	maps.Clear(T.Servers)
	maps.Clear(T.Clients)
}

func stateUtilString(count [StateCount]int, util [StateCount]float64) string {
	var b strings.Builder

	var addSpace bool
	for state, u := range util {
		if u == 0.0 || math.IsNaN(u) {
			continue
		}
		if addSpace {
			b.WriteString(", ")
		} else {
			addSpace = true
		}
		b.WriteString(strconv.Itoa(count[state]))
		b.WriteString(" ")
		b.WriteString(State(state).String())
		b.WriteString(" (")
		b.WriteString(strconv.FormatFloat(u*100, 'f', 2, 64))
		b.WriteString("%)")
	}

	return b.String()
}

func (T *Metrics) String() string {
	return fmt.Sprintf("%d transactions | %d servers (%s) | %d clients (%s)",
		T.TransactionCount(),
		len(T.Servers),
		stateUtilString(T.ServerStateCount(), T.ServerStateUtil()),
		len(T.Clients),
		stateUtilString(T.ClientStateCount(), T.ClientStateUtil()),
	)
}

type ItemMetrics struct {
	// Time is the time of this metrics read
	Time time.Time

	State State
	// Peer is the currently connected server or client
	Peer uuid.UUID
	// Since is the last time that Peer changed.
	Since time.Time

	// InState is how long this item spent in each state
	InState [StateCount]time.Duration

	// Transactions is the number of handled transactions since last metrics reset
	Transactions int
}

func MakeItemMetrics() ItemMetrics {
	now := time.Now()

	return ItemMetrics{
		Time:  now,
		Since: now,
	}
}

func (T *ItemMetrics) commitSince(now time.Time) {
	since := now.Sub(T.Since)
	if T.Since.Before(T.Time) {
		since = now.Sub(T.Time)
	}

	T.InState[T.State] += since
}

func (T *ItemMetrics) SetState(state State, peer uuid.UUID) {
	now := time.Now()

	T.commitSince(now)

	T.Peer = peer
	T.Since = now
	T.State = state
}

func (T *ItemMetrics) Read(metrics *ItemMetrics) {
	now := time.Now()

	*metrics = *T

	metrics.commitSince(now)

	T.Time = now
	T.InState = [StateCount]time.Duration{}
	T.Transactions = 0
}
