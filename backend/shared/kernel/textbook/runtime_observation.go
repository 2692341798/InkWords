package textbook

import (
	"encoding/json"
	"fmt"
	"strings"
)

const RuntimeObservationOutputFormat = "inkwords.runtime-observation.v1"

// RuntimeObservationOutput separates direct tool output from explanatory
// prose. Observations are machine-readable facts; any interpretation must
// declare whether it is a source explanation or a bounded inference.
type RuntimeObservationOutput struct {
	Format          string                             `json:"format"`
	Observations    json.RawMessage                    `json:"observations"`
	Interpretations []RuntimeObservationInterpretation `json:"interpretations"`
}

type RuntimeObservationInterpretation struct {
	Statement       string   `json:"statement"`
	Basis           string   `json:"basis"`
	ObservationRefs []string `json:"observation_refs"`
}

func (output RuntimeObservationOutput) Validate() error {
	if output.Format != RuntimeObservationOutputFormat || len(output.Observations) == 0 || !json.Valid(output.Observations) {
		return fmt.Errorf("runtime observation output requires a structured observation payload")
	}
	for _, interpretation := range output.Interpretations {
		if strings.TrimSpace(interpretation.Statement) == "" || (interpretation.Basis != "source_explanation" && interpretation.Basis != "bounded_inference") || len(interpretation.ObservationRefs) == 0 {
			return fmt.Errorf("runtime interpretation requires a statement, declared basis, and observation references")
		}
		for _, reference := range interpretation.ObservationRefs {
			if strings.TrimSpace(reference) == "" {
				return fmt.Errorf("runtime interpretation observation reference is required")
			}
		}
	}
	return nil
}

func ParseRuntimeObservationOutput(value string) (RuntimeObservationOutput, error) {
	var output RuntimeObservationOutput
	if err := json.Unmarshal([]byte(value), &output); err != nil {
		return RuntimeObservationOutput{}, fmt.Errorf("decode runtime observation output: %w", err)
	}
	if err := output.Validate(); err != nil {
		return RuntimeObservationOutput{}, err
	}
	return output, nil
}

func MarshalRuntimeObservationOutput(observations any, interpretations []RuntimeObservationInterpretation) ([]byte, error) {
	observationJSON, err := json.Marshal(observations)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime observations: %w", err)
	}
	output := RuntimeObservationOutput{Format: RuntimeObservationOutputFormat, Observations: observationJSON, Interpretations: interpretations}
	if err := output.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(output)
}
