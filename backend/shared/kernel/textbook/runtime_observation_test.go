package textbook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeObservationOutputSeparatesToolOutputFromInterpretation(t *testing.T) {
	encoded, err := MarshalRuntimeObservationOutput(map[string]any{"exit_code": 0, "output": "ok"}, nil)
	require.NoError(t, err)
	var output RuntimeObservationOutput
	require.NoError(t, json.Unmarshal(encoded, &output))
	require.NoError(t, output.Validate())
	require.Empty(t, output.Interpretations)

	output.Interpretations = []RuntimeObservationInterpretation{{Statement: "该调用链可能经过中间件。", Basis: "bounded_inference", ObservationRefs: []string{"commands[0].stdout"}}}
	require.NoError(t, output.Validate())
	output.Interpretations[0].Basis = "observed"
	require.ErrorContains(t, output.Validate(), "declared basis")

	_, err = ParseRuntimeObservationOutput(`{"format":"inkwords.runtime-observation.v1","observations":{"exit_code":0},"interpretations":[]}`)
	require.NoError(t, err)
}
