package service

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompositionService_AnalyzeStreamSignatureIncludesSubDir(t *testing.T) {
	rt := reflect.TypeOf(&DecompositionService{})
	m, ok := rt.MethodByName("AnalyzeStream")
	require.True(t, ok)

	require.Equal(t, 6, m.Type.NumIn())
	assert.True(t, m.Type.In(1).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()))
	assert.Equal(t, reflect.TypeOf(""), m.Type.In(2))
	assert.Equal(t, reflect.TypeOf(""), m.Type.In(3))
	assert.Equal(t, reflect.TypeOf((chan<- string)(nil)), m.Type.In(4))
	assert.Equal(t, reflect.TypeOf((chan<- error)(nil)), m.Type.In(5))
}

func TestDecompositionService_AnalyzeStreamUsesFetchWithSubDir(t *testing.T) {
	data, err := os.ReadFile("decomposition.go")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "FetchWithSubDir("))
}

func TestDecompositionService_UsesRateLimiter(t *testing.T) {
	data, err := os.ReadFile("decomposition.go")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "rate.NewLimiter("))
	assert.True(t, strings.Contains(string(data), ".Wait("))
}

func TestDecompositionService_UsesEnvMaxWorkers(t *testing.T) {
	data, err := os.ReadFile("decomposition.go")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "MAX_CONCURRENT_WORKERS"))
}
