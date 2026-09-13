package textbook

import "fmt"

// ApprovalGate tracks human approvals against immutable upstream hashes.
type ApprovalGate struct {
	ContractHash          string
	BlueprintHash         string
	ApprovedBlueprintHash string
	ApprovedSampleHash    string
	SampleContractHash    string
	SampleBlueprintHash   string
}

func (gate ApprovalGate) CanGenerate() error {
	if gate.ContractHash == "" || gate.BlueprintHash == "" {
		return fmt.Errorf("missing contract or blueprint")
	}
	if gate.ApprovedBlueprintHash != gate.BlueprintHash {
		return fmt.Errorf("current blueprint is not human-approved")
	}
	if gate.ApprovedSampleHash == "" || gate.SampleContractHash != gate.ContractHash || gate.SampleBlueprintHash != gate.BlueprintHash {
		return fmt.Errorf("current sample is not human-approved")
	}
	return nil
}
