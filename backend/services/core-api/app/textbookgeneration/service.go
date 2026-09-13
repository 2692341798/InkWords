// Package textbookgeneration coordinates the immutable textbook-generation request boundary.
package textbookgeneration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var defaultSampleGenerationBudget = sharedgeneration.TokenBudget{MaxInput: sharedtextbook.SampleMaxInputTokens, ReservedOutput: sharedtextbook.SampleReservedOutputTokens}

func estimateSampleGenerationInput(payload sharedtextbook.SampleGenerationTaskPayload) (int, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal frozen textbook task for token estimate: %w", err)
	}
	return sharedgeneration.ConservativeTokenEstimate(string(rawPayload)) + sharedtextbook.SamplePreflightInstructionReserveTokens, nil
}

type payloadPreparer interface {
	PrepareSampleGeneration(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.SampleGenerationTarget) (sharedtextbook.SampleGenerationTaskPayload, error)
}

type generationTaskCreator interface {
	CreateGenerationTask(context.Context, coretask.CreateGenerationTaskInput) (coretask.JobTask, error)
}

// Service freezes a server-owned evidence pack before delegating only asynchronous execution.
// The browser therefore cannot submit arbitrary excerpts, contracts, or task owners.
type Service struct {
	preparer    payloadPreparer
	taskCreator generationTaskCreator
	target      sharedtextbook.SampleGenerationTarget
}

func NewService(preparer payloadPreparer, taskCreator generationTaskCreator, targets ...sharedtextbook.SampleGenerationTarget) *Service {
	target := sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}
	if len(targets) > 0 {
		target = targets[0]
	}
	return &Service{preparer: preparer, taskCreator: taskCreator, target: target}
}

// SampleGenerationTargetFromConfig maps shared runtime settings to an
// immutable task identity without exposing provider credentials to core-api.
func SampleGenerationTargetFromConfig(provider, model string) (sharedtextbook.SampleGenerationTarget, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.TrimSpace(model)
	if provider == "" || provider == "fake" {
		return sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}, nil
	}
	if provider != "deepseek" && provider != "openai" {
		return sharedtextbook.SampleGenerationTarget{}, fmt.Errorf("unsupported textbook generation provider %q", provider)
	}
	target := sharedtextbook.SampleGenerationTarget{ProviderName: provider, ModelName: model}
	if err := target.Validate(); err != nil {
		return sharedtextbook.SampleGenerationTarget{}, err
	}
	return target, nil
}

// CreateSampleGenerationTask creates an idempotent task keyed by the complete frozen input.
func (s *Service) prepare(ctx context.Context, workspaceID, projectID, chapterID uuid.UUID) (sharedtextbook.SampleGenerationTaskPayload, textbookdomain.SampleGenerationPreflight, error) {
	if s.preparer == nil || workspaceID == uuid.Nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, textbookdomain.SampleGenerationPreflight{}, errors.New("textbook generation is not configured")
	}
	payload, err := s.preparer.PrepareSampleGeneration(ctx, workspaceID, projectID, chapterID, s.target)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, textbookdomain.SampleGenerationPreflight{}, err
	}
	if err := payload.Validate(); err != nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, textbookdomain.SampleGenerationPreflight{}, fmt.Errorf("validate frozen textbook task: %w", err)
	}
	estimatedInput, err := estimateSampleGenerationInput(payload)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, textbookdomain.SampleGenerationPreflight{}, err
	}
	allowedInput := defaultSampleGenerationBudget.MaxInput - defaultSampleGenerationBudget.ReservedOutput
	preflight := textbookdomain.SampleGenerationPreflight{
		TaskVersion:            payload.TaskVersion,
		PromptSchemaVersion:    payload.PromptSchemaVersion,
		QualityContractVersion: payload.QualityContractVersion,
		GenerationTarget:       payload.GenerationTarget,
		InputHash:              payload.InputHash,
		EvidenceReferenceCount: len(payload.EvidencePack.Evidence),
		EstimatedInputTokens:   estimatedInput,
		AllowedInputTokens:     allowedInput,
		ReservedOutputTokens:   defaultSampleGenerationBudget.ReservedOutput,
		WithinBudget:           estimatedInput <= allowedInput,
		EstimateMethod:         "conservative_utf8_payload_plus_prompt_reserve_v1",
		CacheStatus:            "unknown_before_worker",
		EstimatedCostKnown:     false,
		RequiresConfirmation:   true,
	}
	return payload, preflight, nil
}

// PrepareSampleGeneration is read-only: it freezes and estimates the same input that create will verify.
func (s *Service) PrepareSampleGeneration(ctx context.Context, workspaceID, projectID, chapterID uuid.UUID) (textbookdomain.SampleGenerationPreflight, error) {
	_, preflight, err := s.prepare(ctx, workspaceID, projectID, chapterID)
	return preflight, err
}

// CreateSampleGenerationTask accepts only a confirmation for the exact current frozen input.
func (s *Service) CreateSampleGenerationTask(ctx context.Context, workspaceID, projectID, chapterID uuid.UUID, confirmedInputHash string) (textbookdomain.SampleGenerationTask, error) {
	if s.preparer == nil || s.taskCreator == nil || workspaceID == uuid.Nil {
		return textbookdomain.SampleGenerationTask{}, errors.New("textbook generation is not configured")
	}
	payload, preflight, err := s.prepare(ctx, workspaceID, projectID, chapterID)
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	if strings.TrimSpace(confirmedInputHash) == "" || confirmedInputHash != payload.InputHash {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: generation preflight confirmation is stale", textbookdomain.ErrInvalidState)
	}
	if !preflight.WithinBudget {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: estimated input %d exceeds allowed input %d", textbookdomain.ErrEvidenceBudgetExceeded, preflight.EstimatedInputTokens, preflight.AllowedInputTokens)
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("marshal frozen textbook task: %w", err)
	}
	task, err := s.taskCreator.CreateGenerationTask(ctx, coretask.CreateGenerationTaskInput{
		WorkspaceID:       workspaceID,
		TextbookChapterID: &chapterID,
		TaskSubtype:       sharedtextbook.TextbookSampleGenerationTaskSubtype,
		IdempotencyKey:    "textbook-sample:" + payload.InputHash,
		Payload:           rawPayload,
	})
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	return textbookdomain.SampleGenerationTask{ID: task.ID, Status: string(task.Status)}, nil
}
