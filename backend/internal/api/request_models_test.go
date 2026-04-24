package api

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeRequest_HasSubDirField(t *testing.T) {
	rt := reflect.TypeOf(AnalyzeRequest{})
	field, ok := rt.FieldByName("SubDir")
	require.True(t, ok)
	assert.Equal(t, "sub_dir", field.Tag.Get("json"))
}

func TestGenerateRequest_HasSubDirField(t *testing.T) {
	rt := reflect.TypeOf(GenerateRequest{})
	field, ok := rt.FieldByName("SubDir")
	require.True(t, ok)
	assert.Equal(t, "sub_dir", field.Tag.Get("json"))
}

func TestProjectAnalyze_UsesSubDirWhenFetchingRepo(t *testing.T) {
	data, err := os.ReadFile("project.go")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "FetchWithSubDir("))
}
