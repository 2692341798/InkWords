package projectcourse

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func TestValidateClaimLedgerRequiresVerifiedKnownEvidence(t *testing.T) {
	pack := EvidencePack{ChapterID: "chapter-1", SourceEvidence: []sharedkernel.EvidenceRef{{EvidenceID: "ev-1", CommitSHA: "0123456789abcdef0123456789abcdef01234567", Path: "main.go", StartLine: 1, EndLine: 2, ContentHash: "sha256:main"}}}
	claim := sharedkernel.Claim{ClaimID: "claim-1", Text: "main is the entrypoint", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{"ev-1"}, Status: sharedkernel.ClaimVerified}
	require.NoError(t, ValidateClaimLedger(pack, []sharedkernel.Claim{claim}))
	claim.Status = sharedkernel.ClaimUnsupported
	require.Error(t, ValidateClaimLedger(pack, []sharedkernel.Claim{claim}))
	claim.Status = sharedkernel.ClaimVerified
	claim.EvidenceIDs = []string{"missing"}
	require.Error(t, ValidateClaimLedger(pack, []sharedkernel.Claim{claim}))
}
