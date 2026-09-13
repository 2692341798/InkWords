package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenerationCacheKeyInvalidatesForEvidenceAndAudience(t *testing.T) {
	key := GenerationCacheKey{SnapshotHash: "s", Audience: "foundation", BookContractHash: "b", StyleSheetHash: "style", Stage: "chapter", EvidenceHash: "e", PromptSchema: "v1", Provider: "fake", Model: "fixture"}
	original := key.Digest()
	key.EvidenceHash = "changed"
	require.NotEqual(t, original, key.Digest())
	key.EvidenceHash = "e"
	key.Audience = "stack_familiar"
	require.NotEqual(t, original, key.Digest())
}
