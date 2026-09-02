package verification

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBubblewrapArgsDisableNetworkAndExposeOnlyWorkspace(t *testing.T) {
	args := buildBubblewrapArgs("/tmp/lab", "test -run TestCheckpoint")
	joined := strings.Join(args, " ")
	require.Contains(t, joined, "--unshare-all")
	require.Contains(t, joined, "--tmpfs /tmp")
	require.Contains(t, joined, "--bind /tmp/lab /workspace")
	require.Contains(t, joined, "--setenv GOCACHE /tmp/go-cache")
	require.Contains(t, joined, "go test ./... -run TestCheckpoint")
}
