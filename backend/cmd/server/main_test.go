package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type shutdownCall struct {
	hasDeadline bool
	deadline    time.Time
	err         error
}

type fakeShutdownServer struct {
	calls chan shutdownCall
}

func (s *fakeShutdownServer) Shutdown(ctx context.Context) error {
	deadline, hasDeadline := ctx.Deadline()
	s.calls <- shutdownCall{
		hasDeadline: hasDeadline,
		deadline:    deadline,
		err:         ctx.Err(),
	}
	return nil
}

func TestNewHTTPServer_ConfiguresTimeoutsForStreaming(t *testing.T) {
	server := newHTTPServer(http.NewServeMux())

	require.Equal(t, ":8080", server.Addr)
	require.Equal(t, 15*time.Second, server.ReadTimeout)
	require.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	require.Zero(t, server.WriteTimeout)
	require.Equal(t, 60*time.Second, server.IdleTimeout)
}

func TestShutdownServerOnContextDone_UsesFreshTimeoutContext(t *testing.T) {
	signalContext, stop := context.WithCancel(context.Background())
	server := &fakeShutdownServer{calls: make(chan shutdownCall, 1)}

	done := make(chan struct{})
	go func() {
		shutdownServerOnContextDone(signalContext, server, 15*time.Second)
		close(done)
	}()

	stop()

	select {
	case call := <-server.calls:
		require.True(t, call.hasDeadline)
		require.NoError(t, call.err)
		remaining := time.Until(call.deadline)
		require.Greater(t, remaining, 0*time.Second)
		require.LessOrEqual(t, remaining, 15*time.Second)
	case <-time.After(time.Second):
		t.Fatal("shutdown was not triggered")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown helper did not return")
	}
}
