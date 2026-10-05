package masteryverification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var (
	ErrNotFound    = errors.New("代码验证任务不存在或不可访问")
	ErrConflict    = errors.New("代码、题目或运行环境已变化，请重新预检")
	ErrBusy        = errors.New("已有代码验证正在运行，请等待或取消后重试")
	ErrUnavailable = errors.New("当前隔离运行器不可用，代码保持未验证")
)

type CapabilitySource interface {
	Capability(context.Context) (sharedtextbook.LearnerVerificationCapability, error)
}

type InputSource interface {
	PrepareLearnerVerificationInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.LearnerRunnerIdentity) (sharedtextbook.LearnerVerificationInput, error)
}

type RunnerClient interface {
	CapabilitySource
	Execute(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationReport, error)
}

type Job struct {
	ID          uuid.UUID                                 `json:"id"`
	WorkspaceID uuid.UUID                                 `json:"-"`
	ObjectiveID uuid.UUID                                 `json:"objective_id"`
	AttemptID   uuid.UUID                                 `json:"attempt_id"`
	RequestID   uuid.UUID                                 `json:"request_id"`
	RetryOf     *uuid.UUID                                `json:"retry_of,omitempty"`
	Status      string                                    `json:"status"`
	Preview     Preview                                   `json:"preview"`
	Input       sharedtextbook.LearnerVerificationInput   `json:"-"`
	Report      *sharedtextbook.LearnerVerificationReport `json:"report,omitempty"`
	ErrorCode   string                                    `json:"error_code,omitempty"`
	CreatedAt   time.Time                                 `json:"created_at"`
	StartedAt   *time.Time                                `json:"started_at,omitempty"`
	CompletedAt *time.Time                                `json:"completed_at,omitempty"`
}

type StartInput struct {
	RequestID         uuid.UUID  `json:"request_id"`
	ExpectedInputHash string     `json:"expected_input_hash"`
	RetryOf           *uuid.UUID `json:"retry_of,omitempty"`
}

type Store interface {
	FindRequest(context.Context, uuid.UUID, uuid.UUID) (*Job, error)
	Create(context.Context, Job, string) (Job, bool, error)
	Resolve(context.Context, sharedtextbook.LearnerVerificationReference, string) (sharedtextbook.LearnerVerificationInput, error)
	Finish(context.Context, uuid.UUID, sharedtextbook.LearnerVerificationReport, string, string) error
	Read(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Job, error)
	Latest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*Job, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Job, error)
	InterruptRunning(context.Context) error
	Interrupt(context.Context, uuid.UUID) error
}

type Preview struct {
	Capability       sharedtextbook.LearnerVerificationCapability `json:"capability"`
	InputHash        string                                       `json:"input_hash,omitempty"`
	SnapshotHash     string                                       `json:"snapshot_hash,omitempty"`
	FilesHash        string                                       `json:"files_hash,omitempty"`
	ExecutionHash    string                                       `json:"execution_files_hash,omitempty"`
	DerivedFiles     []sharedtextbook.LearnerCodeFile             `json:"derived_files,omitempty"`
	RequiresExplicit bool                                         `json:"requires_explicit_start"`
}

type Service struct {
	capabilities RunnerClient
	inputs       InputSource
	store        Store
	mu           sync.Mutex
	running      map[uuid.UUID]context.CancelFunc
}

func NewService(capabilities RunnerClient, inputs InputSource, stores ...Store) *Service {
	var store Store
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Service{capabilities: capabilities, inputs: inputs, store: store, running: map[uuid.UUID]context.CancelFunc{}}
}

// Preview is read-only. When isolation is unavailable it returns that fact as
// normal data and never creates a task or stages learner files.
func (service *Service) Preview(ctx context.Context, owner, objective, attempt uuid.UUID) (Preview, error) {
	if service == nil || service.capabilities == nil || service.inputs == nil || owner == uuid.Nil || objective == uuid.Nil || attempt == uuid.Nil {
		return Preview{}, fmt.Errorf("invalid learner verification preview")
	}
	capability, err := service.capabilities.Capability(ctx)
	if err != nil {
		return Preview{}, err
	}
	if err := capability.Validate(); err != nil {
		return Preview{}, err
	}
	preview := Preview{Capability: capability, RequiresExplicit: true}
	if !capability.Available || capability.Runner == nil {
		return preview, nil
	}
	input, err := service.inputs.PrepareLearnerVerificationInput(ctx, owner, objective, attempt, *capability.Runner)
	if err != nil {
		return Preview{}, err
	}
	preview.InputHash = input.Plan.InputHash
	preview.SnapshotHash = input.Plan.SnapshotHash
	preview.FilesHash = input.Plan.FilesHash
	preview.ExecutionHash = input.Plan.ExecutionFilesHash
	preview.DerivedFiles = input.Plan.DerivedFiles
	return preview, nil
}

func (service *Service) Recover(ctx context.Context) error {
	if service == nil || service.store == nil {
		return nil
	}
	return service.store.InterruptRunning(ctx)
}

func (service *Service) Start(ctx context.Context, owner, objective, attempt uuid.UUID, request StartInput) (Job, error) {
	if service == nil || service.store == nil || request.RequestID == uuid.Nil || request.ExpectedInputHash == "" {
		return Job{}, ErrConflict
	}
	existing, err := service.store.FindRequest(ctx, owner, request.RequestID)
	if err != nil {
		return Job{}, err
	}
	if existing != nil {
		if existing.ObjectiveID != objective || existing.AttemptID != attempt || existing.Preview.InputHash != request.ExpectedInputHash || !sameOptionalID(existing.RetryOf, request.RetryOf) {
			return Job{}, ErrConflict
		}
		return *existing, nil
	}
	preview, err := service.Preview(ctx, owner, objective, attempt)
	if err != nil {
		return Job{}, err
	}
	if !preview.Capability.Available || preview.Capability.Runner == nil {
		return Job{}, ErrUnavailable
	}
	if preview.InputHash != request.ExpectedInputHash {
		return Job{}, ErrConflict
	}
	if request.RetryOf != nil {
		previous, err := service.store.Read(ctx, owner, objective, *request.RetryOf)
		if err != nil || previous.AttemptID != attempt || previous.Status == "queued" || previous.Status == "running" {
			return Job{}, ErrConflict
		}
	}
	input, err := service.inputs.PrepareLearnerVerificationInput(ctx, owner, objective, attempt, *preview.Capability.Runner)
	if err != nil || input.Plan.InputHash != preview.InputHash {
		return Job{}, ErrConflict
	}
	token, tokenHash, err := newClaimToken()
	if err != nil {
		return Job{}, err
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if len(service.running) > 0 {
		saved, findErr := service.store.FindRequest(ctx, owner, request.RequestID)
		if findErr == nil && saved != nil && saved.ObjectiveID == objective && saved.AttemptID == attempt && saved.Preview.InputHash == request.ExpectedInputHash && sameOptionalID(saved.RetryOf, request.RetryOf) {
			return *saved, nil
		}
		return Job{}, ErrBusy
	}
	job, created, err := service.store.Create(ctx, Job{ID: uuid.New(), WorkspaceID: owner, ObjectiveID: objective, AttemptID: attempt, RequestID: request.RequestID, RetryOf: request.RetryOf, Status: "queued", Preview: preview, Input: input, CreatedAt: time.Now().UTC()}, tokenHash)
	if err != nil || !created {
		return job, err
	}
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: job.ID.String(), WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), InputHash: input.Plan.InputHash, ClaimToken: token}
	runCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	service.running[job.ID] = cancel
	go service.execute(runCtx, job, reference)
	return job, nil
}

func (service *Service) execute(ctx context.Context, job Job, reference sharedtextbook.LearnerVerificationReference) {
	defer func() {
		if recover() != nil {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = service.store.Interrupt(saveCtx, job.ID)
			cancel()
		}
		service.mu.Lock()
		if cancel := service.running[job.ID]; cancel != nil {
			cancel()
		}
		delete(service.running, job.ID)
		service.mu.Unlock()
	}()
	report, err := service.capabilities.Execute(ctx, reference)
	status, code := string(report.Status), ""
	if err != nil || report.ValidateFor(reference, job.Input.Plan) != nil {
		now := time.Now().UTC()
		report = sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: job.Input.Plan.SnapshotHash, Runner: job.Input.Plan.Runner, Profile: job.Input.Plan.Profile, Policy: job.Input.Plan.Policy, Status: sharedtextbook.LearnerVerificationUnavailable, CompletedAt: now, Reason: "隔离运行器未返回可验证结果"}
		status, code = string(sharedtextbook.LearnerVerificationUnavailable), "runner_unavailable"
	}
	for attempt := 0; attempt < 3; attempt++ {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		saveErr := service.store.Finish(saveCtx, job.ID, report, status, code)
		cancel()
		if saveErr == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (service *Service) Resolve(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error) {
	if service == nil || service.store == nil || reference.Validate() != nil {
		return sharedtextbook.LearnerVerificationInput{}, ErrNotFound
	}
	return service.store.Resolve(ctx, reference, claimTokenHash(reference.ClaimToken))
}

func (service *Service) Read(ctx context.Context, owner, objective, id uuid.UUID) (Job, error) {
	job, err := service.store.Read(ctx, owner, objective, id)
	if err == nil && (job.Status == "queued" || job.Status == "running") {
		service.mu.Lock()
		local := service.running[id] != nil
		service.mu.Unlock()
		if !local {
			if err := service.store.Interrupt(ctx, id); err != nil {
				return Job{}, err
			}
			return service.store.Read(ctx, owner, objective, id)
		}
	}
	return job, err
}

func (service *Service) Latest(ctx context.Context, owner, objective, attempt uuid.UUID) (*Job, error) {
	job, err := service.store.Latest(ctx, owner, objective, attempt)
	if err != nil || job == nil {
		return job, err
	}
	current, err := service.Read(ctx, owner, objective, job.ID)
	return &current, err
}

// LatestAssessmentEvidence returns a validated stored input/report pair. It
// does not call capability, start a run, or fall back to an older result.
func (service *Service) LatestAssessmentEvidence(ctx context.Context, owner, objective, attempt uuid.UUID) (*sharedtextbook.LearnerVerificationInput, *sharedtextbook.LearnerVerificationReport, error) {
	job, err := service.Latest(ctx, owner, objective, attempt)
	if err != nil || job == nil || job.Report == nil {
		return nil, nil, err
	}
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: job.ID.String(), WorkspaceID: owner.String(), ObjectiveID: objective.String(), AttemptID: attempt.String(), InputHash: job.Input.Plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
	if job.Input.ValidateFor(reference) != nil || job.Report.ValidateFor(reference, job.Input.Plan) != nil {
		return nil, nil, ErrConflict
	}
	switch job.Report.Status {
	case sharedtextbook.LearnerVerificationPassed, sharedtextbook.LearnerVerificationFailed, sharedtextbook.LearnerVerificationTimedOut:
		input, report := job.Input, *job.Report
		return &input, &report, nil
	default:
		return nil, nil, nil
	}
}

func (service *Service) Cancel(ctx context.Context, owner, objective, id uuid.UUID) (Job, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	job, err := service.store.Cancel(ctx, owner, objective, id)
	if err == nil && job.Status == "cancelled" {
		if cancel := service.running[id]; cancel != nil {
			cancel()
		}
	}
	return job, err
}

func newClaimToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, claimTokenHash(token), nil
}

func claimTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sameOptionalID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
