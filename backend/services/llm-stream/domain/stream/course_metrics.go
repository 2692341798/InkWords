package stream

import (
	"encoding/json"
	"sync"
	"time"
)

type CourseMetricsSnapshot struct {
	StageRuns       map[string]int   `json:"stage_runs"`
	StageFailures   map[string]int   `json:"stage_failures"`
	CompletedStages map[string]int   `json:"completed_stages"`
	TotalDurationMS map[string]int64 `json:"total_duration_ms"`
	GateFailures    map[string]int   `json:"gate_failures"`
	ClaimStatuses   map[string]int   `json:"claim_statuses"`
	CoverageItems   map[string]int   `json:"coverage_items"`
}

type CourseMetrics struct {
	mu       sync.Mutex
	snapshot CourseMetricsSnapshot
}

func NewCourseMetrics() *CourseMetrics {
	return &CourseMetrics{snapshot: CourseMetricsSnapshot{StageRuns: map[string]int{}, StageFailures: map[string]int{}, CompletedStages: map[string]int{}, TotalDurationMS: map[string]int64{}, GateFailures: map[string]int{}, ClaimStatuses: map[string]int{}, CoverageItems: map[string]int{}}}
}

func (m *CourseMetrics) ObserveResult(result []byte) {
	if m == nil {
		return
	}
	var payload struct {
		QualityReport []struct {
			Name   string `json:"name"`
			Result string `json:"result"`
		} `json:"quality_report"`
		Chapters []struct {
			Document *struct {
				Claims []struct {
					Status string `json:"status"`
				} `json:"claims"`
			} `json:"document"`
		} `json:"chapters"`
		Coverage map[string][]struct {
			Covered bool `json:"covered"`
		} `json:"coverage"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, gate := range payload.QualityReport {
		if gate.Result == "hard_fail" || gate.Result == "soft_fail" {
			m.snapshot.GateFailures[gate.Name]++
		}
	}
	for _, chapter := range payload.Chapters {
		if chapter.Document == nil {
			continue
		}
		for _, claim := range chapter.Document.Claims {
			m.snapshot.ClaimStatuses[claim.Status]++
		}
	}
	for kind, items := range payload.Coverage {
		for _, item := range items {
			key := kind + ":covered"
			if !item.Covered {
				key = kind + ":uncovered"
			}
			m.snapshot.CoverageItems[key]++
		}
	}
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
	clone := CourseMetricsSnapshot{StageRuns: map[string]int{}, StageFailures: map[string]int{}, CompletedStages: map[string]int{}, TotalDurationMS: map[string]int64{}, GateFailures: map[string]int{}, ClaimStatuses: map[string]int{}, CoverageItems: map[string]int{}}
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
	for key, value := range m.snapshot.GateFailures {
		clone.GateFailures[key] = value
	}
	for key, value := range m.snapshot.ClaimStatuses {
		clone.ClaimStatuses[key] = value
	}
	for key, value := range m.snapshot.CoverageItems {
		clone.CoverageItems[key] = value
	}
	return clone
}
