package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCourseMetricsSnapshotsStageRunsAndFailures(t *testing.T) {
	metrics := NewCourseMetrics()
	metrics.Observe("project_course_analyze", 10*time.Millisecond, true)
	metrics.Observe("project_course_generate", 20*time.Millisecond, false)
	snapshot := metrics.Snapshot()
	require.Equal(t, 1, snapshot.StageRuns["project_course_analyze"])
	require.Equal(t, 1, snapshot.CompletedStages["project_course_analyze"])
	require.Equal(t, 1, snapshot.StageFailures["project_course_generate"])
}

func TestCourseMetricsObservesGatesClaimsAndCoverage(t *testing.T) {
	metrics := NewCourseMetrics()
	metrics.ObserveCache(true)
	metrics.ObserveCache(false)
	metrics.ObserveResult([]byte(`{"quality_report":[{"name":"official_source","result":"hard_fail"}],"chapters":[{"document":{"claims":[{"status":"verified"},{"status":"unsupported"}]}}],"coverage":{"files":[{"covered":true},{"covered":false}]},"usage":{"prompt_tokens":11,"completion_tokens":7,"prompt_cache_hit_tokens":3,"prompt_cache_miss_tokens":8}}`))
	snapshot := metrics.Snapshot()
	require.Equal(t, 1, snapshot.CacheHits)
	require.Equal(t, 1, snapshot.CacheMisses)
	require.Equal(t, 11, snapshot.PromptTokens)
	require.Equal(t, 7, snapshot.CompletionTokens)
	require.Equal(t, 3, snapshot.PromptCacheHitTokens)
	require.Equal(t, 8, snapshot.PromptCacheMissTokens)
	require.Equal(t, 1, snapshot.GateFailures["official_source"])
	require.Equal(t, 1, snapshot.ClaimStatuses["verified"])
	require.Equal(t, 1, snapshot.ClaimStatuses["unsupported"])
	require.Equal(t, 1, snapshot.CoverageItems["files:covered"])
	require.Equal(t, 1, snapshot.CoverageItems["files:uncovered"])
}
