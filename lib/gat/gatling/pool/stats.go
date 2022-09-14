package pool

import (
	"gfx.cafe/gfx/pggat/lib/gat"
	"time"
)

type Stats struct {
	start time.Time

	xactCount  int
	queryCount int
	waitCount  int
	received   int
	sent       int
	xactTime   int
	queryTime  int
	waitTime   int
}

func newStats() *Stats {
	return &Stats{
		start: time.Now(),
	}
}

func (s *Stats) TotalXactCount() int {
	return s.xactCount
}

func (s *Stats) TotalQueryCount() int {
	return s.queryCount
}

func (s *Stats) TotalReceived() int {
	return s.received
}

func (s *Stats) TotalSent() int {
	return s.sent
}

func (s *Stats) TotalXactTime() int {
	return s.xactTime
}

func (s *Stats) TotalQueryTime() int {
	return s.queryTime
}

func (s *Stats) TotalWaitTime() int {
	return s.waitTime
}

func (s *Stats) totalTime() time.Duration {
	return time.Now().Sub(s.start)
}

func (s *Stats) AvgXactCount() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.xactCount) / seconds
}

func (s *Stats) AvgQueryCount() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.queryCount) / seconds
}

func (s *Stats) AvgRecv() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.received) / seconds
}

func (s *Stats) AvgSent() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.sent) / seconds
}

func (s *Stats) AvgXactTime() float64 {
	return float64(s.xactTime) / float64(s.xactCount)
}

func (s *Stats) AvgQueryTime() float64 {
	return float64(s.queryTime) / float64(s.queryCount)
}

func (s *Stats) AvgWaitTime() float64 {
	return float64(s.waitTime) / float64(s.waitCount)
}

var _ gat.PoolStats = (*Stats)(nil)
