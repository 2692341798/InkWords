package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"inkwords-backend/shared/kernel/httpx"
)

func TestNewHTTPServer_ConfiguresTimeoutsForStreaming(t *testing.T) {
	server := httpx.NewServer(http.NewServeMux())

	require.Equal(t, ":8080", server.Addr)
	require.Equal(t, 15*time.Second, server.ReadTimeout)
	require.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	require.Zero(t, server.WriteTimeout)
	require.Equal(t, 60*time.Second, server.IdleTimeout)
}
