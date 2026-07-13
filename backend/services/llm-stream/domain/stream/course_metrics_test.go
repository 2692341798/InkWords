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
