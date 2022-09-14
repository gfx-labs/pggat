package gat

import "time"

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

func (s *PoolStats) TotalXactCount() int {
	return s.xactCount
}

func (s *PoolStats) IncXactCount() {
	s.xactCount += 1
}

func (s *PoolStats) TotalQueryCount() int {
	return s.queryCount
}

func (s *PoolStats) IncQueryCount() {
	s.queryCount += 1
}

func (s *PoolStats) TotalReceived() int {
	return s.received
}

func (s *PoolStats) TotalSent() int {
	return s.sent
}

func (s *PoolStats) TotalXactTime() int {
	return s.xactTime
}

func (s *PoolStats) TotalQueryTime() int {
	return s.queryTime
}

func (s *PoolStats) TotalWaitTime() int {
	return s.waitTime
}

func (s *PoolStats) totalTime() time.Duration {
	return time.Now().Sub(s.start)
}

func (s *PoolStats) AvgXactCount() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.xactCount) / seconds
}

func (s *PoolStats) AvgQueryCount() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.queryCount) / seconds
}

func (s *PoolStats) AvgRecv() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.received) / seconds
}

func (s *PoolStats) AvgSent() float64 {
	seconds := s.totalTime().Seconds()
	return float64(s.sent) / seconds
}

func (s *PoolStats) AvgXactTime() float64 {
	return float64(s.xactTime) / float64(s.xactCount)
}

func (s *PoolStats) AvgQueryTime() float64 {
	return float64(s.queryTime) / float64(s.queryCount)
}

func (s *PoolStats) AvgWaitTime() float64 {
	return float64(s.waitTime) / float64(s.waitCount)
}
