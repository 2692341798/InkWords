package mastery

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memoryStore struct {
	objectives map[uuid.UUID]Objective
	attempts   map[uuid.UUID][]AttemptRecord
	schedules  map[uuid.UUID]Schedule
}

func newMemoryStore() *memoryStore {
	return &memoryStore{objectives: map[uuid.UUID]Objective{}, attempts: map[uuid.UUID][]AttemptRecord{}, schedules: map[uuid.UUID]Schedule{}}
}
func (store *memoryStore) CreateObjectiveAndInitialSchedule(_ context.Context, objective *Objective, schedule *Schedule) error {
	objective.ID = uuid.New()
	store.objectives[objective.ID] = *objective
	schedule.ObjectiveID = objective.ID
	store.schedules[objective.ID] = *schedule
	return nil
}

func TestServiceCreatesOneImmediatelyDueInitialTaskAndReusesItsChapterIdentity(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	workspaceID := uuid.New()
	input := validObjectiveInput([]Skill{Explain, Complete, Reproduce, Transfer, Diagnose, Retain})
	input.ChapterID = "chapter-1"

	first, err := service.CreateObjective(context.Background(), workspaceID, input)
	require.NoError(t, err)
	second, err := service.CreateObjective(context.Background(), workspaceID, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, store.objectives, 1)
	require.Equal(t, string(Explain), store.schedules[first.ID].NextSkill)

	tasks, err := service.Due(context.Background(), workspaceID, time.Now().UTC().Add(time.Second), 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, first.ID, tasks[0].ObjectiveID)
	require.Equal(t, Explain, tasks[0].Skill)
}
func (store *memoryStore) FindObjectiveByChapter(_ context.Context, workspaceID uuid.UUID, chapterID string) (Objective, bool, error) {
	for _, objective := range store.objectives {
		if objective.WorkspaceID == workspaceID && objective.ChapterID == chapterID {
			return objective, true, nil
		}
	}
	return Objective{}, false, nil
}
func (store *memoryStore) GetObjective(_ context.Context, workspaceID, objectiveID uuid.UUID) (Objective, error) {
	objective, ok := store.objectives[objectiveID]
	if !ok || objective.WorkspaceID != workspaceID {
		return Objective{}, errors.New("not found")
	}
	return objective, nil
}
func (store *memoryStore) ListAttempts(_ context.Context, objectiveID uuid.UUID) ([]AttemptRecord, error) {
	return append([]AttemptRecord(nil), store.attempts[objectiveID]...), nil
}
func (store *memoryStore) GetAttempt(_ context.Context, objectiveID, attemptID uuid.UUID) (AttemptRecord, error) {
	for _, attempt := range store.attempts[objectiveID] {
		if attempt.ID == attemptID {
			return attempt, nil
		}
	}
	return AttemptRecord{}, errors.New("not found")
}
func (store *memoryStore) AppendAttemptAndSchedule(_ context.Context, record *AttemptRecord, schedule *Schedule) error {
	record.ID = uuid.New()
	store.attempts[record.ObjectiveID] = append(store.attempts[record.ObjectiveID], *record)
	store.schedules[schedule.ObjectiveID] = *schedule
	return nil
}
func (store *memoryStore) ListDue(_ context.Context, workspaceID uuid.UUID, now time.Time, _ int) ([]DueObjective, error) {
	due := []DueObjective{}
	for id, schedule := range store.schedules {
		objective := store.objectives[id]
		if objective.WorkspaceID == workspaceID && !schedule.DueAt.After(now) {
			due = append(due, DueObjective{ObjectiveID: id, ChapterID: objective.ChapterID, Title: objective.Title, Skill: Skill(schedule.NextSkill), DueAt: schedule.DueAt, Reason: schedule.Reason})
		}
	}
	return due, nil
}

func TestServicePersistsAppendOnlyAttemptAndReturnsDueTaskWhenApplicationOpens(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	userID := uuid.New()
	objective, err := service.CreateObjective(context.Background(), userID, validObjectiveInput([]Skill{Explain, Transfer, Diagnose, Retain}))
	require.NoError(t, err)
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	due, err := service.RecordAttempt(context.Background(), userID, objective.ID, Attempt{Skill: Explain, Correct: false, Independent: false, HintCount: 1, Confidence: 2, ErrorKinds: []string{"missing_mechanism"}, At: now})
	require.NoError(t, err)
	require.Equal(t, Explain, due.Skill)
	require.Len(t, store.attempts[objective.ID], 1)
	require.Equal(t, algorithmVersion, store.schedules[objective.ID].AlgorithmVersion)

	tasks, err := service.Due(context.Background(), userID, now.Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, objective.ID, tasks[0].ObjectiveID)
	require.Equal(t, Explain, tasks[0].Skill)
}

func TestServiceRejectsAttemptOutsideObjectiveSkills(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	userID := uuid.New()
	objective, err := service.CreateObjective(context.Background(), userID, validObjectiveInput([]Skill{Explain, Diagnose}))
	require.NoError(t, err)

	_, err = service.RecordAttempt(context.Background(), userID, objective.ID, Attempt{
		Skill:       Transfer,
		Correct:     true,
		Independent: true,
		Confidence:  4,
		At:          time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
	})

	require.ErrorContains(t, err, "not required")
	require.Empty(t, store.attempts[objective.ID])
}

func TestServiceMigratesLegacyNoteOnceIntoWorkspaceObjective(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	workspaceID := uuid.New()
	input := validObjectiveInput(nil)
	input.ChapterID = "legacy-note:8e65bd81"
	input.EvidenceRefs = []string{"legacy-note:wiki/concepts/router.md"}

	first, created, err := service.EnsureLegacyNoteObjective(context.Background(), workspaceID, input)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, []Skill{Explain, Retain}, decodeSkills(t, first.RequiredSkills))

	second, created, err := service.EnsureLegacyNoteObjective(context.Background(), workspaceID, input)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, store.objectives, 1)
}

func decodeSkills(t *testing.T, raw []byte) []Skill {
	t.Helper()
	var skills []Skill
	require.NoError(t, json.Unmarshal(raw, &skills))
	return skills
}

func validObjectiveInput(skills []Skill) ObjectiveInput {
	return ObjectiveInput{ChapterID: "chapter-1", Title: "解释路由登记", Behavior: "独立说明路由如何登记", Skills: skills, Rubric: []string{"说明方法与路径"}, KeyPoints: []string{"路由树"}, EvidenceRefs: []string{"evidence:gin-routing"}}
}
