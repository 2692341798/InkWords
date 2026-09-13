package masteryassessment

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service owns bounded in-process execution; durable records own recovery.
type Service struct {
	source  InputSource
	engine  Engine
	store   Store
	mu      sync.Mutex
	running map[uuid.UUID]context.CancelFunc
}

// NewService does not start work or call a model. Bootstrap explicitly recovers old jobs.
func NewService(source InputSource, engine Engine, store Store) *Service {
	return &Service{source: source, engine: engine, store: store, running: map[uuid.UUID]context.CancelFunc{}}
}

// Recover marks interrupted requests for explicit retry; it never replays a call.
func (s *Service) Recover(ctx context.Context) error { return s.store.InterruptRunning(ctx) }

// Preview resolves immutable inputs and verifies budget without a model call.
func (s *Service) Preview(ctx context.Context, owner, objective, attempt uuid.UUID) (Preview, error) {
	input, err := s.source.PrepareAssessmentInput(ctx, owner, objective, attempt)
	if err != nil {
		return Preview{}, err
	}
	return s.engine.Preview(input)
}

// Start deduplicates retries before fetching sources and launching one bounded call.
func (s *Service) Start(ctx context.Context, owner, objective, attempt uuid.UUID, request StartInput) (Job, error) {
	if owner == uuid.Nil || objective == uuid.Nil || attempt == uuid.Nil || request.RequestID == uuid.Nil {
		return Job{}, ErrConflict
	}
	existing, err := s.store.FindRequest(ctx, owner, request.RequestID)
	if err != nil {
		return Job{}, err
	}
	if existing != nil {
		if existing.ObjectiveID != objective || existing.AttemptID != attempt || existing.Preview.InputHash != request.ExpectedInputHash || existing.Preview.RequestHash != request.ExpectedRequestHash {
			return Job{}, ErrConflict
		}
		return *existing, nil
	}
	input, err := s.source.PrepareAssessmentInput(ctx, owner, objective, attempt)
	if err != nil {
		return Job{}, err
	}
	preview, err := s.engine.Preview(input)
	if err != nil {
		return Job{}, err
	}
	if preview.InputHash != request.ExpectedInputHash || preview.RequestHash != request.ExpectedRequestHash {
		return Job{}, ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.running) > 0 {
		// A concurrent retry may have raced the first request's admission.
		saved, err := s.store.FindRequest(ctx, owner, request.RequestID)
		if err != nil {
			return Job{}, err
		}
		if saved != nil && saved.ObjectiveID == objective && saved.AttemptID == attempt && saved.Preview.InputHash == request.ExpectedInputHash && saved.Preview.RequestHash == request.ExpectedRequestHash {
			return *saved, nil
		}
		latest, err := s.store.Latest(ctx, owner, objective, attempt)
		if err != nil {
			return Job{}, err
		}
		if latest != nil && latest.Preview.InputHash == request.ExpectedInputHash && latest.Preview.RequestHash == request.ExpectedRequestHash && (latest.Status == "running" || latest.Status == "succeeded") {
			return *latest, nil
		}
		return Job{}, ErrBusy
	}
	job, created, err := s.store.Create(ctx, Job{ID: uuid.New(), WorkspaceID: owner, ObjectiveID: objective, AttemptID: attempt, RequestID: request.RequestID, RetryOf: request.RetryOf, Status: "running", Input: input, Preview: preview, CreatedAt: time.Now().UTC()})
	if err != nil || !created {
		return job, err
	}
	runCtx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	s.running[job.ID] = cancel
	go s.execute(runCtx, job)
	return job, nil
}

func (s *Service) execute(ctx context.Context, job Job) {
	defer func() {
		if recover() != nil {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = s.store.Interrupt(saveCtx, job.ID)
			cancel()
		}
		s.mu.Lock()
		if cancel := s.running[job.ID]; cancel != nil {
			cancel()
		}
		delete(s.running, job.ID)
		s.mu.Unlock()
	}()
	result, err := s.engine.Assess(ctx, job.Input)
	status, code := "succeeded", ""
	if err != nil {
		status, code = "failed", "provider_failed"
		switch {
		case errors.Is(err, context.Canceled):
			status, code = "cancelled", "cancelled"
		case errors.Is(err, context.DeadlineExceeded):
			code = "timeout"
		case errors.Is(err, ErrInvalidFeedback):
			code = "invalid_feedback"
		case errors.Is(err, ErrBusy):
			code = "busy"
		case errors.Is(err, ErrBudgetExceeded):
			code = "budget_exceeded"
		case errors.Is(err, ErrUnavailable):
			code = "unavailable"
		}
	}
	// Persistence retries do not re-execute the provider. If the database stays
	// unavailable, startup recovery leaves the call outcome explicitly unknown.
	for attempt := 0; attempt < 3; attempt++ {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		saveErr := s.store.Finish(saveCtx, job.ID, result, status, code)
		cancel()
		if saveErr == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Read restores status, immutable feedback and the append-only correction view.
func (s *Service) Read(ctx context.Context, owner, objective, job uuid.UUID) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.store.Read(ctx, owner, objective, job)
	if err == nil && result.Status == "running" && s.running[job] == nil {
		if err := s.store.Interrupt(ctx, job); err != nil {
			return Job{}, err
		}
		return s.store.Read(ctx, owner, objective, job)
	}
	return result, err
}

// Latest reads past work without launching a call.
func (s *Service) Latest(ctx context.Context, owner, objective, attempt uuid.UUID) (*Job, error) {
	job, err := s.store.Latest(ctx, owner, objective, attempt)
	if err != nil || job == nil {
		return job, err
	}
	current, err := s.Read(ctx, owner, objective, job.ID)
	return &current, err
}

// Cancel persists cancellation before interrupting local execution.
func (s *Service) Cancel(ctx context.Context, owner, objective, id uuid.UUID) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.store.Cancel(ctx, owner, objective, id)
	if err == nil && job.Status == "cancelled" {
		if cancel := s.running[id]; cancel != nil {
			cancel()
		}
	}
	return job, err
}

// Correct appends a workspace-attributed correction using the effective feedback hash.
func (s *Service) Correct(ctx context.Context, owner, objective, id uuid.UUID, input CorrectionInput) (Job, error) {
	return s.store.Correct(ctx, owner, objective, id, input)
}

// Apply uses a saved feedback snapshot without another model call.
func (s *Service) Apply(ctx context.Context, owner, objective, id uuid.UUID, input ApplyInput) (Job, error) {
	return s.store.Apply(ctx, owner, objective, id, input)
}
