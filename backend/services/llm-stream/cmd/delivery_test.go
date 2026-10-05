package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookapp "inkwords-backend/services/llm-stream/app/textbook"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

type fakeAcknowledger struct {
	ackErr  error
	nackErr error
	acked   int
	nacked  int
}

type providerConnectionFailurePort struct{ err error }

func (p providerConnectionFailurePort) Generate(context.Context, sharedgeneration.Request) (sharedgeneration.Result, error) {
	return sharedgeneration.Result{}, p.err
}

func (f *fakeAcknowledger) Ack(bool) error {
	f.acked++
	return f.ackErr
}

func (f *fakeAcknowledger) Nack(bool, bool) error {
	f.nacked++
	return f.nackErr
}

func TestAckDelivery_ReturnsWrappedFailure(t *testing.T) {
	ack := &fakeAcknowledger{ackErr: errors.New("channel closed")}
	err := ackDelivery(ack, "malformed generation message")
	require.ErrorContains(t, err, "malformed generation message")
	require.ErrorContains(t, err, "channel closed")
	require.Equal(t, 1, ack.acked)
}

func TestNackDelivery_ReturnsWrappedFailure(t *testing.T) {
	ack := &fakeAcknowledger{nackErr: errors.New("channel closed")}
	err := nackDelivery(ack, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.ErrorContains(t, err, "nack generation task")
	require.Equal(t, 1, ack.nacked)
}

func TestTextbookSampleRunnerFromEnvironmentDefaultsToFakeAndRejectsUnsafeConfiguration(t *testing.T) {
	t.Setenv("TEXTBOOK_GENERATION_PROVIDER", "")
	runner, err := textbookSampleRunnerFromEnvironment()
	require.NoError(t, err)
	require.NotNil(t, runner)

	t.Setenv("TEXTBOOK_GENERATION_PROVIDER", "openai")
	t.Setenv("TEXTBOOK_STANDARD_MODEL", "gpt-test")
	t.Setenv("OPENAI_API_KEY", "")
	_, err = textbookSampleRunnerFromEnvironment()
	require.ErrorContains(t, err, "key is not configured")

	t.Setenv("TEXTBOOK_GENERATION_PROVIDER", "unsupported")
	_, err = textbookSampleRunnerFromEnvironment()
	require.ErrorContains(t, err, "unsupported textbook generation provider")
}

func TestTextbookProviderPortFromEnvironmentSelectsConfiguredDeepSeekV4Flash(t *testing.T) {
	t.Setenv("TEXTBOOK_GENERATION_PROVIDER", "deepseek")
	t.Setenv("TEXTBOOK_STANDARD_MODEL", "deepseek-v4-flash")
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	port, provider, model, err := textbookProviderPortFromEnvironment()
	require.NoError(t, err)
	require.NotNil(t, port)
	require.Equal(t, "deepseek", provider)
	require.Equal(t, "deepseek-v4-flash", model)
}

func TestTextbookGenerationRequestTimeoutFromEnvironment(t *testing.T) {
	t.Run("defaults to a sample-sized bounded timeout", func(t *testing.T) {
		t.Setenv("TEXTBOOK_GENERATION_REQUEST_TIMEOUT", "")
		timeout, err := textbookGenerationRequestTimeoutFromEnvironment()
		require.NoError(t, err)
		require.Equal(t, 15*time.Minute, timeout)
	})

	t.Run("accepts an operator override", func(t *testing.T) {
		t.Setenv("TEXTBOOK_GENERATION_REQUEST_TIMEOUT", "8m")
		timeout, err := textbookGenerationRequestTimeoutFromEnvironment()
		require.NoError(t, err)
		require.Equal(t, 8*time.Minute, timeout)
	})

	for _, value := range []string{"invalid", "29s", "31m"} {
		t.Run("rejects_"+value, func(t *testing.T) {
			t.Setenv("TEXTBOOK_GENERATION_REQUEST_TIMEOUT", value)
			_, err := textbookGenerationRequestTimeoutFromEnvironment()
			require.Error(t, err)
		})
	}
}

func TestProviderConnectionHandlerOnlyReturnsSafeErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", providerConnectionHandler(func() (*textbookapp.ProviderConnectionTester, error) {
		return nil, errors.New("OPENAI_API_KEY=must-never-appear")
	}))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/test", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.NotContains(t, response.Body.String(), "must-never-appear")
	require.Contains(t, response.Body.String(), "PROVIDER_CONNECTION_UNAVAILABLE")
}

func TestProviderConnectionHandlerDoesNotExposeProviderDiagnostics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", providerConnectionHandler(func() (*textbookapp.ProviderConnectionTester, error) {
		return textbookapp.NewProviderConnectionTester(providerConnectionFailurePort{err: errors.New("provider body leaked OPENAI_API_KEY=must-never-appear")}, "openai", "gpt-test"), nil
	}))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/test", nil))
	require.Equal(t, http.StatusBadGateway, response.Code)
	require.Contains(t, response.Body.String(), "PROVIDER_CONNECTION_FAILED")
	require.NotContains(t, response.Body.String(), "must-never-appear")
}
