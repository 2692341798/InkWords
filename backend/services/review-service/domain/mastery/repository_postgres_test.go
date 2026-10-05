package mastery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"inkwords-backend/shared/platform/postgres/migrations"
)

func TestPostgresPracticeBindingAndConcurrentAppendPreserveEvidence(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("practice_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("test-only-password"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NoError(t, db.Exec(`CREATE TABLE review_sessions(id UUID PRIMARY KEY,user_id UUID NOT NULL,note_path TEXT NOT NULL,created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,deleted_at TIMESTAMPTZ)`).Error)
	provider, err := migrations.NewReviewProvider(sqlDB)
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 28)
	require.NoError(t, err)
	basis := practiceBasisFixture()
	store, err := NewGormStore(db)
	require.NoError(t, err)
	service := NewService(store).WithPracticeSource(fixedPracticeSource{basis})
	objective, err := service.CreateObjective(ctx, basis.WorkspaceID, ObjectiveInput{ChapterID: "approved-revision:" + basis.Learning.ChapterID + ":" + basis.Learning.RevisionID})
	require.NoError(t, err)
	stored, err := store.GetObjective(ctx, basis.WorkspaceID, objective.ID)
	require.NoError(t, err)
	projection, err := stored.practiceProjection()
	require.NoError(t, err)
	require.Equal(t, basis.Learning, *projection)
	require.Error(t, db.Exec("UPDATE mastery_objectives SET practice_projection = '{}'::jsonb WHERE id = ?", objective.ID).Error, "missing JSON keys cannot evade a SQL CHECK through NULL")
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "frozen mastery practice evidence")
	_, err = provider.UpTo(ctx, 32)
	require.NoError(t, err)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	sessions := []PracticeSessionView{}
	for _, skill := range []Skill{Explain, Complete} {
		session, err := service.BeginPracticeSession(ctx, basis.WorkspaceID, objective.ID, skill)
		require.NoError(t, err)
		sessions = append(sessions, session)
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func(i int) {
			<-start
			session := sessions[i]
			record := &AttemptRecord{ObjectiveID: objective.ID, Skill: string(session.Skill), Answer: "按方法和路径查找", PracticeSessionID: &session.ID, SubmissionHash: "sha256:" + strings.Repeat("a", 64), PracticeTaskID: session.PracticeTaskID, PracticeContentHash: basis.Learning.ContentHash, Confidence: 4, ErrorKinds: []byte(`[]`), AttemptedAt: now}
			schedule := &Schedule{ObjectiveID: objective.ID, NextSkill: string(Complete), DueAt: now.Add(time.Hour), AlgorithmVersion: algorithmVersion}
			results <- store.AppendAttemptAndSchedule(ctx, record, schedule)
		}(i)
	}
	close(start)
	first, second := <-results, <-results
	require.True(t, (first == nil) != (second == nil), "only one stale-history writer may succeed")
	if first != nil {
		require.ErrorContains(t, first, "history changed")
	}
	if second != nil {
		require.ErrorContains(t, second, "history changed")
	}
	records, err := store.ListAttempts(ctx, objective.ID)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Contains(t, []string{"task-explain", "task-complete"}, records[0].PracticeTaskID)
	require.Equal(t, basis.Learning.ContentHash, records[0].PracticeContentHash)
	var plan []string
	rows, err := sqlDB.QueryContext(ctx, "EXPLAIN (FORMAT JSON) SELECT count(*) FROM mastery_attempts WHERE objective_id = $1", objective.ID)
	require.NoError(t, err)
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan = append(plan, line)
	}
	require.NoError(t, rows.Close())
	require.Contains(t, strings.Join(plan, ""), "mastery_attempts")
	t.Log("append guard query plan:", strings.Join(plan, ""))
	assertPersistentPracticeSessions(t, ctx, service, store, objective, basis, now)
	// v32/v31/v30 contain no code or grading evidence in this fixture.
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "practice sessions or disclosure evidence")
	var preserved json.RawMessage
	require.NoError(t, db.Raw("SELECT practice_projection FROM mastery_objectives WHERE id = ?", objective.ID).Row().Scan(&preserved))
	require.NotEmpty(t, preserved)
	_, err = store.GetObjective(ctx, uuid.New(), objective.ID)
	require.Error(t, err)
}
