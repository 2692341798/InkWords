package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingTaskPublisher struct {
	generation []GenerationRequestedMessage
	export     []ExportRequestedMessage
}

func (p *recordingTaskPublisher) PublishGenerationRequested(_ context.Context, message GenerationRequestedMessage) error {
	p.generation = append(p.generation, message)
	return nil
}
func (p *recordingTaskPublisher) PublishParseRequested(context.Context, ParseRequestedMessage) error {
	return nil
}
func (p *recordingTaskPublisher) PublishExportRequested(_ context.Context, message ExportRequestedMessage) error {
	p.export = append(p.export, message)
	return nil
}

func TestProjectCourseTaskSubtypesUseExistingQueuesAndRemainIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&JobTask{}, &JobTaskEvent{}))
	publisher := &recordingTaskPublisher{}
	service := NewService(NewGormRepository(db), publisher, nil)
	owner := uuid.New()
	input := CreateProjectCourseTaskInput{RequestedBy: owner, IdempotencyKey: "course:1", Payload: []byte(`{"course_id":"course-1"}`)}
	analyze, err := service.CreateProjectCourseAnalyzeTask(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, ProjectCourseAnalyzeTaskSubtype, analyze.TaskSubtype)
	analyzeAgain, err := service.CreateProjectCourseAnalyzeTask(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, analyze.ID, analyzeAgain.ID)
	generate, err := service.CreateProjectCourseGenerateTask(context.Background(), CreateProjectCourseTaskInput{RequestedBy: owner, IdempotencyKey: "course:generate", Payload: input.Payload})
	require.NoError(t, err)
	require.Equal(t, ProjectCourseGenerateTaskSubtype, generate.TaskSubtype)
	pack, err := service.CreateProjectCoursePackageTask(context.Background(), CreateProjectCourseTaskInput{RequestedBy: owner, IdempotencyKey: "course:package", Payload: input.Payload})
	require.NoError(t, err)
	require.Equal(t, ProjectCoursePackageTaskSubtype, pack.TaskSubtype)
	require.Len(t, publisher.generation, 2)
	require.Len(t, publisher.export, 1)
}
