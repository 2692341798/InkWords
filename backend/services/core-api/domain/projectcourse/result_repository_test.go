package projectcourse

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func TestPersistProjectCourseResultTransitionsAnalyzingCourse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:project-course-result-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ProjectCourse{}))
	repo := NewGormRepository(db)
	course := &ProjectCourse{UserID: uuid.New(), RepositoryURL: "https://github.com/example/repo", RequestedRef: "main", AudienceLevel: "programming", Status: StatusAnalyzing, BlueprintVersion: 1, BlueprintJSON: []byte(`{}`), CoverageJSON: []byte(`{}`), QualityReportJSON: []byte(`{}`)}
	require.NoError(t, repo.Create(context.Background(), course))

	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: course.RepositoryURL, RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0).UTC()}
	blueprint := sharedkernel.Blueprint{CourseID: course.ID.String(), BlueprintVersion: 1, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: sharedkernel.AudienceProgramming}
	result := map[string]any{"course_id": course.ID.String(), "snapshot": snapshot, "blueprint": blueprint, "coverage": sharedkernel.CoverageMatrix{}}
	require.NoError(t, repo.PersistProjectCourseResult(context.Background(), result))

	updated, err := repo.GetByID(context.Background(), course.UserID, course.ID)
	require.NoError(t, err)
	require.Equal(t, StatusAwaitingApproval, updated.Status)
	require.Equal(t, snapshot.ResolvedCommitSHA, updated.ResolvedCommitSHA)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(updated.BlueprintJSON, &stored))
	require.Equal(t, course.ID.String(), stored["course_id"])
}
