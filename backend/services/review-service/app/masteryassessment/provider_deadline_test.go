package masteryassessment

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdapterCancellationKeepsSafeClassification(t *testing.T) {
	for _, cause := range []error{context.DeadlineExceeded, context.Canceled} {
		t.Run(cause.Error(), func(t *testing.T) {
			input, output := fixture(t)
			port := &capturedPort{result: output, err: fmt.Errorf("private provider diagnostics: %w", cause)}
			result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
			require.ErrorIs(t, err, cause)
			require.NotContains(t, err.Error(), "private")
			require.Nil(t, result.Feedback)
			require.Equal(t, output.Usage, result.Usage)
			require.Equal(t, 1, port.calls)
		})
	}
}
