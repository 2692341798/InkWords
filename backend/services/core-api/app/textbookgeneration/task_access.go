package textbookgeneration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var (
	ErrTaskOutsideTextbookScope  = errors.New("task is outside the local textbook workspace scope")
	ErrRetryConfirmationRequired = errors.New("textbook generation retry confirmation is required")
)

type taskAccess interface {
	GetTextbookTask(context.Context, uuid.UUID, uuid.UUID) (coretask.JobTask, error)
	RetryTextbookTask(context.Context, uuid.UUID, uuid.UUID) (coretask.JobTask, error)
}

// TaskSnapshot is the safe browser projection of a local textbook task. Frozen
// task input and the bridge owner identity are deliberately not exposed.
type TaskSnapshot struct {
	ExecutionMode     string                       `json:"execution_mode,omitempty"`
	ID                uuid.UUID                    `json:"id"`
	TaskType          string                       `json:"task_type"`
	TaskSubtype       string                       `json:"task_subtype"`
	Status            coretask.JobTaskStatus       `json:"status"`
	Result            json.RawMessage              `json:"result,omitempty"`
	ErrorMessage      string                       `json:"error_message,omitempty"`
	RetryCount        int                          `json:"retry_count"`
	StartedAt         *time.Time                   `json:"started_at,omitempty"`
	FinishedAt        *time.Time                   `json:"finished_at,omitempty"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
	RetryConfirmation *GenerationRetryConfirmation `json:"retry_confirmation,omitempty"`
}

// GenerationRetryConfirmation exposes only the immutable cost/provenance
// summary needed for a person to authorize one more provider call. Source
// excerpts and the rest of the frozen task payload stay server-side.
type GenerationRetryConfirmation struct {
	ExecutionMode          string                                `json:"execution_mode,omitempty"`
	GenerationTarget       sharedtextbook.SampleGenerationTarget `json:"generation_target"`
	InputHash              string                                `json:"input_hash"`
	PromptSchemaVersion    string                                `json:"prompt_schema_version"`
	QualityContractVersion string                                `json:"quality_contract_version"`
	EstimatedInputTokens   int                                   `json:"estimated_input_tokens"`
	AllowedInputTokens     int                                   `json:"allowed_input_tokens"`
	ReservedOutputTokens   int                                   `json:"reserved_output_tokens"`
	EstimatedCostKnown     bool                                  `json:"estimated_cost_known"`
	RequiresConfirmation   bool                                  `json:"requires_confirmation"`
}

// TaskAccessService limits the local workspace route to the four versioned
// textbook task subtypes. Ownership is checked against job_tasks.workspace_id.
type TaskAccessService struct {
	tasks taskAccess
}

func NewTaskAccessService(tasks taskAccess) *TaskAccessService {
	return &TaskAccessService{tasks: tasks}
}

func (service *TaskAccessService) Get(ctx context.Context, workspaceID, taskID uuid.UUID) (TaskSnapshot, error) {
	task, err := service.getTextbookTask(ctx, workspaceID, taskID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	return snapshotTask(task), nil
}

func (service *TaskAccessService) Retry(ctx context.Context, workspaceID, taskID uuid.UUID, confirmedInputHash string) (TaskSnapshot, error) {
	task, err := service.getTextbookTask(ctx, workspaceID, taskID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if task.TaskSubtype == sharedtextbook.TextbookSampleGenerationTaskSubtype && task.Status == coretask.JobTaskStatusFailed {
		confirmation := generationRetryConfirmation(task)
		if confirmation == nil || strings.TrimSpace(confirmedInputHash) != confirmation.InputHash {
			return TaskSnapshot{}, ErrRetryConfirmationRequired
		}
	}
	retried, err := service.tasks.RetryTextbookTask(ctx, taskID, workspaceID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if !isTextbookTaskSubtype(retried.TaskSubtype) {
		return TaskSnapshot{}, ErrTaskOutsideTextbookScope
	}
	return snapshotTask(retried), nil
}

func (service *TaskAccessService) getTextbookTask(ctx context.Context, workspaceID, taskID uuid.UUID) (coretask.JobTask, error) {
	if service == nil || service.tasks == nil || workspaceID == uuid.Nil || taskID == uuid.Nil {
		return coretask.JobTask{}, errors.New("textbook task access is not configured")
	}
	task, err := service.tasks.GetTextbookTask(ctx, taskID, workspaceID)
	if err != nil {
		return coretask.JobTask{}, err
	}
	if !isTextbookTaskSubtype(task.TaskSubtype) {
		return coretask.JobTask{}, ErrTaskOutsideTextbookScope
	}
	return task, nil
}

func isTextbookTaskSubtype(subtype string) bool {
	switch subtype {
	case sharedtextbook.TextbookSampleGenerationTaskSubtype,
		sharedtextbook.TextbookSourceImportTaskSubtype,
		sharedtextbook.TextbookOfficialWebImportTaskSubtype,
		sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype:
		return true
	default:
		return false
	}
}

func snapshotTask(task coretask.JobTask) TaskSnapshot {
	mode := ""
	var payload sharedtextbook.SampleGenerationTaskPayload
	if task.TaskSubtype == sharedtextbook.TextbookSampleGenerationTaskSubtype && json.Unmarshal(task.PayloadJSON, &payload) == nil && payload.Correction != nil {
		mode = "automated_local_correction"
	}
	return TaskSnapshot{
		ExecutionMode: mode,
		ID:            task.ID, TaskType: task.TaskType, TaskSubtype: task.TaskSubtype, Status: task.Status,
		Result: append(json.RawMessage(nil), task.ResultJSON...), ErrorMessage: task.ErrorMessage,
		RetryCount: task.RetryCount, StartedAt: task.StartedAt, FinishedAt: task.FinishedAt,
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
		RetryConfirmation: generationRetryConfirmation(task),
	}
}

func generationRetryConfirmation(task coretask.JobTask) *GenerationRetryConfirmation {
	if task.Status != coretask.JobTaskStatusFailed || task.TaskSubtype != sharedtextbook.TextbookSampleGenerationTaskSubtype {
		return nil
	}
	var payload sharedtextbook.SampleGenerationTaskPayload
	if err := json.Unmarshal(task.PayloadJSON, &payload); err != nil || payload.Validate() != nil {
		return nil
	}
	if payload.Correction != nil {
		return &GenerationRetryConfirmation{ExecutionMode: "automated_local_correction", GenerationTarget: payload.GenerationTarget, InputHash: payload.InputHash, PromptSchemaVersion: payload.PromptSchemaVersion, QualityContractVersion: payload.QualityContractVersion, RequiresConfirmation: true}
	}
	estimatedInput, err := estimateSampleGenerationInput(payload)
	if err != nil {
		return nil
	}
	allowedInput := defaultSampleGenerationBudget.MaxInput - defaultSampleGenerationBudget.ReservedOutput
	return &GenerationRetryConfirmation{
		GenerationTarget:       payload.GenerationTarget,
		InputHash:              payload.InputHash,
		PromptSchemaVersion:    payload.PromptSchemaVersion,
		QualityContractVersion: payload.QualityContractVersion,
		EstimatedInputTokens:   estimatedInput,
		AllowedInputTokens:     allowedInput,
		ReservedOutputTokens:   defaultSampleGenerationBudget.ReservedOutput,
		EstimatedCostKnown:     false,
		RequiresConfirmation:   true,
	}
}
