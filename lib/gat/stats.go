package gat

import (
	"sync/atomic"
	"time"
)

type PoolStats struct {
	start time.Time

	xactCount  atomic.Int64
	queryCount atomic.Int64
	waitCount  atomic.Int64
	received   atomic.Int64
	sent       atomic.Int64
	xactTime   atomic.Int64
	queryTime  atomic.Int64
	waitTime   atomic.Int64
}

func NewPoolStats() *PoolStats {
	return &PoolStats{
		start: time.Now(),
	}
}

func (s *PoolStats) TimeActive() time.Duration {
	return time.Now().Sub(s.start)
}

func (s *PoolStats) TotalXactCount() int64 {
	return s.xactCount.Load()
}

func (s *PoolStats) TotalQueryCount() int64 {
	return s.queryCount.Load()
}

func (s *PoolStats) TotalWaitCount() int64 {
	return s.waitCount.Load()
}

func (s *PoolStats) TotalReceived() int64 {
	return s.received.Load()
}

func (s *PoolStats) AddTotalReceived(amount int64) {
	s.received.Add(amount)
}

func (s *PoolStats) TotalSent() int64 {
	return s.sent.Load()
}

func (s *PoolStats) AddTotalSent(amount int64) {
	s.sent.Add(amount)
}

func (s *PoolStats) TotalXactTime() int64 {
	return s.xactTime.Load()
}

func (s *PoolStats) AddXactTime(time int64) {
	s.xactCount.Add(1)
	s.xactTime.Add(time)
}

func (s *PoolStats) TotalQueryTime() int64 {
	return s.queryTime.Load()
}

func (s *PoolStats) AddQueryTime(time int64) {
	s.queryCount.Add(1)
	s.queryTime.Add(time)
}

func (s *PoolStats) TotalWaitTime() int64 {
	return s.waitTime.Load()
}

func (s *PoolStats) AddWaitTime(time int64) {
	s.waitCount.Add(1)
	s.waitTime.Add(time)
}

func (s *PoolStats) AvgXactCount() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.xactCount.Load()) / seconds
}

func (s *PoolStats) AvgQueryCount() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.queryCount.Load()) / seconds
}

func (s *PoolStats) AvgRecv() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.received.Load()) / seconds
}

func (s *PoolStats) AvgSent() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.sent.Load()) / seconds
}

func (s *PoolStats) AvgXactTime() float64 {
	xactCount := s.xactCount.Load()
	if xactCount == 0 {
		return 0
	}
	return float64(s.xactTime.Load()) / float64(xactCount)
}

func (s *PoolStats) AvgQueryTime() float64 {
	queryCount := s.queryCount.Load()
	if queryCount == 0 {
		return 0
	}
	return float64(s.queryTime.Load()) / float64(queryCount)
}

func (s *PoolStats) AvgWaitTime() float64 {
	waitCount := s.waitCount.Load()
	if waitCount == 0 {
		return 0
	}
	return float64(s.waitTime.Load()) / float64(waitCount)
}
