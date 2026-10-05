package masteryverification

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type lifecycleInputs struct {
	input sharedtextbook.LearnerVerificationInput
}

func (source lifecycleInputs) PrepareLearnerVerificationInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.LearnerRunnerIdentity) (sharedtextbook.LearnerVerificationInput, error) {
	return source.input, nil
}

type lifecycleRunner struct {
	capability sharedtextbook.LearnerVerificationCapability
	resolve    func(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error)
	block      bool
	calls      atomic.Int32
	reference  sharedtextbook.LearnerVerificationReference
}

func (runner *lifecycleRunner) Capability(context.Context) (sharedtextbook.LearnerVerificationCapability, error) {
	return runner.capability, nil
}
func (runner *lifecycleRunner) Execute(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationReport, error) {
	runner.calls.Add(1)
	runner.reference = reference
	input, err := runner.resolve(ctx, reference)
	if err != nil {
		return sharedtextbook.LearnerVerificationReport{}, err
	}
	if runner.block {
		<-ctx.Done()
		return sharedtextbook.LearnerVerificationReport{}, ctx.Err()
	}
	now, zero := time.Now().UTC(), 0
	return sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: input.Plan.SnapshotHash, ExecutionTreeHash: digest('f'), Runner: input.Plan.Runner, Profile: input.Plan.Profile, Policy: input.Plan.Policy, Status: sharedtextbook.LearnerVerificationPassed, ExecutionStarted: true, StartedAt: &now, CompletedAt: now, ExitCode: &zero}, nil
}

type memoryStore struct {
	mu        sync.Mutex
	jobs      map[uuid.UUID]Job
	claims    map[uuid.UUID]string
	byRequest map[uuid.UUID]uuid.UUID
}

func newMemoryStore() *memoryStore {
	return &memoryStore{jobs: map[uuid.UUID]Job{}, claims: map[uuid.UUID]string{}, byRequest: map[uuid.UUID]uuid.UUID{}}
}
func (store *memoryStore) FindRequest(_ context.Context, owner, request uuid.UUID) (*Job, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	id, ok := store.byRequest[request]
	if !ok || store.jobs[id].WorkspaceID != owner {
		return nil, nil
	}
	job := store.jobs[id]
	return &job, nil
}
func (store *memoryStore) Create(_ context.Context, job Job, claim string) (Job, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.jobs[job.ID], store.claims[job.ID], store.byRequest[job.RequestID] = job, claim, job.ID
	return job, true, nil
}
func (store *memoryStore) Resolve(_ context.Context, reference sharedtextbook.LearnerVerificationReference, claim string) (sharedtextbook.LearnerVerificationInput, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	id, err := uuid.Parse(reference.RunID)
	if err != nil || store.claims[id] != claim {
		return sharedtextbook.LearnerVerificationInput{}, ErrNotFound
	}
	job := store.jobs[id]
	if job.Status != "queued" {
		return sharedtextbook.LearnerVerificationInput{}, ErrNotFound
	}
	now := time.Now().UTC()
	job.Status, job.StartedAt = "running", &now
	store.jobs[id] = job
	return job.Input, nil
}
func (store *memoryStore) Finish(_ context.Context, id uuid.UUID, report sharedtextbook.LearnerVerificationReport, status, code string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	job := store.jobs[id]
	if job.Status == "cancelled" {
		return nil
	}
	now := time.Now().UTC()
	job.Status, job.Report, job.ErrorCode, job.CompletedAt = status, &report, code, &now
	store.jobs[id] = job
	return nil
}
func (store *memoryStore) Read(_ context.Context, owner, objective, id uuid.UUID) (Job, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	job, ok := store.jobs[id]
	if !ok || job.WorkspaceID != owner || job.ObjectiveID != objective {
		return Job{}, ErrNotFound
	}
	return job, nil
}
func (store *memoryStore) Latest(_ context.Context, owner, objective, attempt uuid.UUID) (*Job, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	var latest *Job
	for _, item := range store.jobs {
		item := item
		if item.WorkspaceID == owner && item.ObjectiveID == objective && item.AttemptID == attempt && (latest == nil || item.CreatedAt.After(latest.CreatedAt)) {
			latest = &item
		}
	}
	return latest, nil
}
func (store *memoryStore) Cancel(ctx context.Context, owner, objective, id uuid.UUID) (Job, error) {
	store.mu.Lock()
	job, ok := store.jobs[id]
	if !ok || job.WorkspaceID != owner || job.ObjectiveID != objective {
		store.mu.Unlock()
		return Job{}, ErrNotFound
	}
	now := time.Now().UTC()
	job.Status, job.CompletedAt = "cancelled", &now
	store.jobs[id] = job
	store.mu.Unlock()
	return store.Read(ctx, owner, objective, id)
}
func (store *memoryStore) InterruptRunning(context.Context) error { return nil }
func (store *memoryStore) Interrupt(_ context.Context, id uuid.UUID) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	job := store.jobs[id]
	now := time.Now().UTC()
	job.Status, job.CompletedAt = "interrupted", &now
	store.jobs[id] = job
	return nil
}

func TestStartUsesOneTimeCapabilityDeduplicatesAndPersistsReport(t *testing.T) {
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	runnerIdentity := sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('a'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	plan := sharedtextbook.LearnerVerificationPlan{Format: sharedtextbook.LearnerVerificationPlanFormat, Profile: sharedtextbook.LearnerGoTestProfile, WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), SnapshotHash: digest('b'), FilesHash: digest('c'), ExecutionFilesHash: digest('d'), Runner: runnerIdentity, Policy: sharedtextbook.DefaultLearnerGoTestPolicy(), InputHash: digest('e')}
	input := sharedtextbook.LearnerVerificationInput{Format: sharedtextbook.LearnerVerificationInputFormat, Plan: plan}
	runner := &lifecycleRunner{capability: sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: &runnerIdentity}}
	store := newMemoryStore()
	service := NewService(runner, lifecycleInputs{input: input}, store)
	runner.resolve = service.Resolve
	request := StartInput{RequestID: uuid.New(), ExpectedInputHash: plan.InputHash}
	job, err := service.Start(context.Background(), owner, objective, attempt, request)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, err := service.Read(context.Background(), owner, objective, job.ID)
		return err == nil && current.Status == "passed"
	}, time.Second, 10*time.Millisecond)
	repeated, err := service.Start(context.Background(), owner, objective, attempt, request)
	require.NoError(t, err)
	require.Equal(t, job.ID, repeated.ID)
	require.EqualValues(t, 1, runner.calls.Load())
	require.Len(t, runner.reference.ClaimToken, 43)
	encoded, err := json.Marshal(repeated)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), runner.reference.ClaimToken)
	changed := runner.reference
	changed.ClaimToken = strings.Repeat("B", 43)
	_, err = service.Resolve(context.Background(), changed)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCancelStopsInFlightRequestAndPreservesCancelledTerminalState(t *testing.T) {
	owner, objective, attempt := uuid.New(), uuid.New(), uuid.New()
	runnerIdentity := sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('a'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	plan := sharedtextbook.LearnerVerificationPlan{Format: sharedtextbook.LearnerVerificationPlanFormat, Profile: sharedtextbook.LearnerGoTestProfile, WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), SnapshotHash: digest('b'), FilesHash: digest('c'), ExecutionFilesHash: digest('d'), Runner: runnerIdentity, Policy: sharedtextbook.DefaultLearnerGoTestPolicy(), InputHash: digest('e')}
	runner := &lifecycleRunner{block: true, capability: sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: &runnerIdentity}}
	store := newMemoryStore()
	service := NewService(runner, lifecycleInputs{input: sharedtextbook.LearnerVerificationInput{Format: sharedtextbook.LearnerVerificationInputFormat, Plan: plan}}, store)
	runner.resolve = service.Resolve
	job, err := service.Start(context.Background(), owner, objective, attempt, StartInput{RequestID: uuid.New(), ExpectedInputHash: plan.InputHash})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, _ := store.Read(context.Background(), owner, objective, job.ID)
		return current.Status == "running"
	}, time.Second, 10*time.Millisecond)
	cancelled, err := service.Cancel(context.Background(), owner, objective, job.ID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", cancelled.Status)
	require.Eventually(t, func() bool {
		current, _ := store.Read(context.Background(), owner, objective, job.ID)
		return current.Status == "cancelled"
	}, time.Second, 10*time.Millisecond)
}
