package bootstrap

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestLearnerCapabilityRequiresBothTrustedIdentityAndIsolation(t *testing.T) {
	identity := &sharedtextbook.LearnerRunnerIdentity{ImageDigest: "sha256:" + strings.Repeat("a", 64), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	available := learnerCapability(identity, nil)
	require.True(t, available.Available)
	require.NoError(t, available.Validate())

	unavailable := learnerCapability(identity, errors.New("namespace preflight failed"))
	require.False(t, unavailable.Available)
	require.Contains(t, unavailable.Reason, "preflight")
	require.NoError(t, unavailable.Validate())

	missing := learnerCapability(nil, errors.New("runner identity missing"))
	require.False(t, missing.Available)
	require.NoError(t, missing.Validate())
}

func TestFileSHA256DigestReadsTheMountedProfileInsteadOfTrustingAClaim(t *testing.T) {
	path := t.TempDir() + "/profile.json"
	require.NoError(t, os.WriteFile(path, []byte("reviewed profile"), 0o600))
	digest, err := fileSHA256Digest(path)
	require.NoError(t, err)
	require.Equal(t, "sha256:a7e4a36b2e6dc6ae71d17ef3db670b106ef395ae3674e9b4620242c3be72cec5", digest)
}
