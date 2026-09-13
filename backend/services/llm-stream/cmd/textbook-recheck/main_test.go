package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/llm-stream/infra/rejecteddraft"
)

func TestRecheckRejectsMixedEditModesAndUnknownCorrectionFields(t *testing.T) {
	root := t.TempDir() + "/private"
	store, err := rejecteddraft.Open(root)
	require.NoError(t, err)
	hash, err := store.Save(context.Background(), []byte(`{}`))
	require.NoError(t, err)
	require.NoError(t, store.Close())
	for _, tc := range []struct {
		flags []string
		input string
	}{
		{[]string{"-markdown-stdin", "-correction-stdin"}, "{}"},
		{[]string{"-correction-stdin"}, `{"provider_name":"invented"}`},
		{[]string{"-correction-stdin"}, "{} {}"},
		{[]string{"-correction-stdin"}, strings.Repeat("x", (1<<20)+1)},
	} {
		var output bytes.Buffer
		err := run(append([]string{"-store", root, "-hash", hash}, tc.flags...), strings.NewReader(tc.input), &output)
		require.Error(t, err)
		require.Empty(t, output.String())
	}
}
