package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestApprovalGateInvalidatesSampleWhenContractChanges(t *testing.T) {
	gate := ApprovalGate{ContractHash: "contract-a", BlueprintHash: "blueprint-a", ApprovedBlueprintHash: "blueprint-a", ApprovedSampleHash: "sample-a", SampleContractHash: "contract-a", SampleBlueprintHash: "blueprint-a"}
	require.NoError(t, gate.CanGenerate())
	gate.ContractHash = "contract-b"
	require.ErrorContains(t, gate.CanGenerate(), "sample")
}
