package stream

import (
	"sync"
	"time"
)

type CourseMetricsSnapshot struct {
	StageRuns       map[string]int   `json:"stage_runs"`
	StageFailures   map[string]int   `json:"stage_failures"`
	CompletedStages map[string]int   `json:"completed_stages"`
	TotalDurationMS map[string]int64 `json:"total_duration_ms"`
}

type CourseMetrics struct {
	mu       sync.Mutex
	snapshot CourseMetricsSnapshot
}

func NewCourseMetrics() *CourseMetrics {
	return &CourseMetrics{snapshot: CourseMetricsSnapshot{StageRuns: map[string]int{}, StageFailures: map[string]int{}, CompletedStages: map[string]int{}, TotalDurationMS: map[string]int64{}}}
}

func (m *CourseMetrics) Observe(stage string, duration time.Duration, completed bool) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.StageRuns[stage]++
	m.snapshot.TotalDurationMS[stage] += duration.Milliseconds()
	if completed {
		m.snapshot.CompletedStages[stage]++
	} else {
		m.snapshot.StageFailures[stage]++
	}
}

func (m *CourseMetrics) Snapshot() CourseMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := CourseMetricsSnapshot{StageRuns: map[string]int{}, StageFailures: map[string]int{}, CompletedStages: map[string]int{}, TotalDurationMS: map[string]int64{}}
	for key, value := range m.snapshot.StageRuns {
		clone.StageRuns[key] = value
	}
	for key, value := range m.snapshot.StageFailures {
		clone.StageFailures[key] = value
	}
	for key, value := range m.snapshot.CompletedStages {
		clone.CompletedStages[key] = value
	}
	for key, value := range m.snapshot.TotalDurationMS {
		clone.TotalDurationMS[key] = value
	}
	return clone
}
