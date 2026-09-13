package textbook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

func TestSampleBudgetRejectsOmittedMessageOverheadBeforeProviderCall(t *testing.T) {
	request := sampleGenerationRequest(t)
	providerRequest, err := providerRequestForSample("configured-model", request)
	require.NoError(t, err)
	// This input fits the former body-only check exactly. Schema, locators and
	// actual message encoding must prevent the call, without truncating evidence.
	parts := []string{providerRequest.SystemInstruction, providerRequest.TaskInstruction}
	for _, evidence := range providerRequest.Evidence {
		parts = append(parts, evidence.Content)
	}
	old, err := CheckBudget(DefaultSampleGenerationBudget(), parts)
	require.NoError(t, err)
	budget := TokenBudget{MaxInput: old.EstimatedInput + providerRequest.MaxOutputTokens, ReservedOutput: providerRequest.MaxOutputTokens}
	port := &capturedGenerationPort{}
	_, err = NewBudgetedPortSampleGenerator(port, "configured-model", budget).Generate(context.Background(), request)
	require.ErrorContains(t, err, "generation budget preflight failed")
	require.Zero(t, port.calls)
}

func TestSampleRequestEstimateUsesTheSharedMessageBudget(t *testing.T) {
	request := sampleGenerationRequest(t)
	providerRequest, err := providerRequestForSample("configured-model", request)
	require.NoError(t, err)
	report, err := sharedgeneration.CheckRequestBudget(DefaultSampleGenerationBudget(), providerRequest)
	require.NoError(t, err)
	require.False(t, report.RequiresCompression)
	port := &capturedGenerationPort{}
	budget := TokenBudget{MaxInput: report.EstimatedInput - 1 + providerRequest.MaxOutputTokens, ReservedOutput: providerRequest.MaxOutputTokens}
	_, err = NewBudgetedPortSampleGenerator(port, "configured-model", budget).Generate(context.Background(), request)
	require.ErrorContains(t, err, "generation budget preflight failed")
	require.Zero(t, port.calls)
	// The exact threshold admits the provider call. Its intentionally empty
	// result then fails provenance, proving the preflight did not reject it.
	budget.MaxInput++
	_, err = NewBudgetedPortSampleGenerator(port, "configured-model", budget).Generate(context.Background(), request)
	require.ErrorContains(t, err, "missing provider provenance")
	require.Equal(t, 1, port.calls)
}
