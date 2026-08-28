package service

import (
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	mu         sync.RWMutex
	registered int
	processed  int
	reviewed   int
	failures   int
	lastEvent  time.Time
}

func (m *Metrics) RecordRegistration(now time.Time) {
	m.mu.Lock()
	m.registered++
	m.lastEvent = now
	m.mu.Unlock()
}
func (m *Metrics) RecordProcessing(now time.Time) {
	m.mu.Lock()
	m.processed++
	m.lastEvent = now
	m.mu.Unlock()
}
func (m *Metrics) RecordReview(now time.Time) {
	m.mu.Lock()
	m.reviewed++
	m.lastEvent = now
	m.mu.Unlock()
}
func (m *Metrics) RecordFailure() { m.mu.Lock(); m.failures++; m.mu.Unlock() }

type MetricsSnapshot struct {
	Registered int       `json:"registered"`
	Processed  int       `json:"processed"`
	Reviewed   int       `json:"reviewed"`
	Failures   int       `json:"failures"`
	LastEvent  time.Time `json:"last_event"`
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MetricsSnapshot{m.registered, m.processed, m.reviewed, m.failures, m.lastEvent}
}

type LatencyWindow struct {
	Samples  []time.Duration
	Capacity int
}

func (w *LatencyWindow) Add(value time.Duration) {
	if w.Capacity <= 0 {
		w.Capacity = 32
	}
	w.Samples = append(w.Samples, value)
	if len(w.Samples) > w.Capacity {
		w.Samples = w.Samples[len(w.Samples)-w.Capacity:]
	}
}
func (w *LatencyWindow) Percentile(percent float64) time.Duration {
	if len(w.Samples) == 0 {
		return 0
	}
	values := append([]time.Duration(nil), w.Samples...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	index := int(float64(len(values)-1) * percent)
	return values[index]
}
