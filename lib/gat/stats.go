package gat

import (
	"time"
)

type PoolStats struct {
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

func NewPoolStats() *PoolStats {
	return &PoolStats{
		start: time.Now(),
	}
}

func (s *PoolStats) TimeActive() time.Duration {
	return time.Now().Sub(s.start)
}

func (s *PoolStats) TotalXactCount() int {
	return s.xactCount
}

func (s *PoolStats) TotalQueryCount() int {
	return s.queryCount
}

func (s *PoolStats) TotalWaitCount() int {
	return s.waitCount
}

func (s *PoolStats) TotalReceived() int {
	return s.received
}

func (s *PoolStats) AddTotalReceived(amount int) {
	s.received += amount
}

func (s *PoolStats) TotalSent() int {
	return s.sent
}

func (s *PoolStats) AddTotalSent(amount int) {
	s.sent += amount
}

func (s *PoolStats) TotalXactTime() int {
	return s.xactTime
}

func (s *PoolStats) AddXactTime(time int) {
	s.xactCount += 1
	s.xactTime += time
}

func (s *PoolStats) TotalQueryTime() int {
	return s.queryTime
}

func (s *PoolStats) AddQueryTime(time int) {
	s.queryCount += 1
	s.queryTime += time
}

func (s *PoolStats) TotalWaitTime() int {
	return s.waitTime
}

func (s *PoolStats) AddWaitTime(time int) {
	s.waitCount += 1
	s.waitTime += time
}

func (s *PoolStats) AvgXactCount() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.xactCount) / seconds
}

func (s *PoolStats) AvgQueryCount() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.queryCount) / seconds
}

func (s *PoolStats) AvgRecv() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.received) / seconds
}

func (s *PoolStats) AvgSent() float64 {
	seconds := s.TimeActive().Seconds()
	return float64(s.sent) / seconds
}

func (s *PoolStats) AvgXactTime() float64 {
	if s.xactCount == 0 {
		return 0
	}
	return float64(s.xactTime) / float64(s.xactCount)
}

func (s *PoolStats) AvgQueryTime() float64 {
	if s.queryCount == 0 {
		return 0
	}
	return float64(s.queryTime) / float64(s.queryCount)
}

func (s *PoolStats) AvgWaitTime() float64 {
	if s.waitCount == 0 {
		return 0
	}
	return float64(s.waitTime) / float64(s.waitCount)
}
