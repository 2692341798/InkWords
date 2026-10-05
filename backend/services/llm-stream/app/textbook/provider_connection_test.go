package textbook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

type providerConnectionPort struct {
	request sharedgeneration.Request
	result  sharedgeneration.Result
	err     error
}

func (port *providerConnectionPort) Generate(_ context.Context, request sharedgeneration.Request) (sharedgeneration.Result, error) {
	port.request = request
	return port.result, port.err
}

func TestProviderConnectionTesterUsesOnlyFixedProbeAndReturnsSafeTelemetry(t *testing.T) {
	port := &providerConnectionPort{result: sharedgeneration.Result{Provider: "openai", Model: "gpt-test", Output: `{"ok":true}`, RequestID: "request-safe", Usage: sharedgeneration.Usage{Known: true}}}
	result, err := NewProviderConnectionTester(port, "openai", "gpt-test").Test(context.Background())
	require.NoError(t, err)
	require.Equal(t, ProviderConnectionResult{Provider: "openai", Model: "gpt-test", RequestID: "request-safe", UsageKnown: true}, result)
	require.Equal(t, "connection-probe", port.request.Evidence[0].ID)
	require.Contains(t, port.request.Evidence[0].Content, "no textbook")
	require.NotContains(t, port.request.TaskInstruction, "教材")
	require.Equal(t, 16, port.request.MaxOutputTokens)
}

func TestProviderConnectionTesterRejectsProviderFailuresAndInvalidAcknowledgements(t *testing.T) {
	port := &providerConnectionPort{err: errors.New("credential rejected: OPENAI_API_KEY=must-never-appear")}
	_, err := NewProviderConnectionTester(port, "openai", "gpt-test").Test(context.Background())
	require.ErrorContains(t, err, "connection request failed")
	require.NotContains(t, err.Error(), "must-never-appear")

	port.err = nil
	port.result = sharedgeneration.Result{Provider: "openai", Model: "gpt-test", Output: `{"ok":false}`}
	_, err = NewProviderConnectionTester(port, "openai", "gpt-test").Test(context.Background())
	require.ErrorContains(t, err, "invalid acknowledgement")

	port.result = sharedgeneration.Result{Provider: "other", Model: "gpt-test", Output: `{"ok":true}`}
	_, err = NewProviderConnectionTester(port, "openai", "gpt-test").Test(context.Background())
	require.ErrorContains(t, err, "mismatched provenance")
}
