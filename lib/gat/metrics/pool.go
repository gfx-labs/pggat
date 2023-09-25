package metrics

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/util/maps"
)

type Pool struct {
	Servers map[uuid.UUID]Conn
	Clients map[uuid.UUID]Conn
}

func (T *Pool) TransactionCount() int {
	var serverTransactions int
	var clientTransactions int

	for _, server := range T.Servers {
		serverTransactions += server.TransactionCount
	}

	for _, client := range T.Clients {
		clientTransactions += client.TransactionCount
	}

	if clientTransactions > serverTransactions {
		return clientTransactions
	}
	return serverTransactions
}

func connStateCounts(items map[uuid.UUID]Conn) [ConnStateCount]int {
	var states [ConnStateCount]int
	for _, item := range items {
		states[item.State]++
	}
	return states
}

func connStateUtils(items map[uuid.UUID]Conn) [ConnStateCount]float64 {
	var util [ConnStateCount]time.Duration
	var total time.Duration
	for _, item := range items {
		for state, amount := range item.Utilization {
			util[state] += amount
			total += amount
		}
	}

	var states [ConnStateCount]float64
	for state := range states {
		states[state] = float64(util[state]) / float64(total)
	}

	return states
}

func connStateUtilString(count [ConnStateCount]int, util [ConnStateCount]float64) string {
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
		b.WriteString(ConnState(state).String())
		b.WriteString(" (")
		b.WriteString(strconv.FormatFloat(u*100, 'f', 2, 64))
		b.WriteString("%)")
	}

	return b.String()
}

func (T *Pool) Clear() {
	maps.Clear(T.Servers)
	maps.Clear(T.Clients)
}

func (T *Pool) String() string {
	return fmt.Sprintf("%d transactions | %d servers (%s) | %d clients (%s)",
		T.TransactionCount(),
		len(T.Servers),
		connStateUtilString(connStateCounts(T.Servers), connStateUtils(T.Servers)),
		len(T.Clients),
		connStateUtilString(connStateCounts(T.Clients), connStateUtils(T.Clients)),
	)
}
