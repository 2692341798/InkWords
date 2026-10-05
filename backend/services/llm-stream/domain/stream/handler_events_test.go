package stream

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func (recorder *closeNotifyRecorder) CloseNotify() <-chan bool { return recorder.closed }

func TestSSEStreamErrorNeverLogsOrEmitsProviderDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder(), closed: make(chan bool)}
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/stream", nil)
	chunks := make(chan string)
	errorsChannel := make(chan error, 1)
	errorsChannel <- errors.New("provider rejected: OPENAI_API_KEY=must-never-appear")
	close(errorsChannel)

	sseStreamBody(context, chunks, &errorsChannel, streamOperationGenerate)

	require.NotContains(t, recorder.Body.String(), "must-never-appear")
	require.Contains(t, recorder.Body.String(), "blog generation failed")
	require.NotContains(t, logs.String(), "must-never-appear")
	require.Contains(t, logs.String(), "blog generation failed")
}
