package api

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	projectdomain "inkwords-backend/internal/domain/project"
	streamdomain "inkwords-backend/internal/domain/stream"
)

func TestAnalyzeRequest_HasSubDirField(t *testing.T) {
	rt := reflect.TypeOf(projectdomain.AnalyzeRequest{})
	field, ok := rt.FieldByName("SubDir")
	require.True(t, ok)
	assert.Equal(t, "sub_dir", field.Tag.Get("json"))
}

func TestGenerateRequest_HasSubDirField(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.GenerateRequest{})
	field, ok := rt.FieldByName("SubDir")
	require.True(t, ok)
	assert.Equal(t, "sub_dir", field.Tag.Get("json"))
}

func TestProjectAnalyze_UsesSubDirWhenFetchingRepo(t *testing.T) {
	data, err := os.ReadFile("../../../../domain/project/service.go")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "FetchWithSubDir("))
}

func TestPolishRequest_HasTitleAndContentFields(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.PolishRequest{})

	field, ok := rt.FieldByName("Title")
	require.True(t, ok)
	assert.Equal(t, "title", field.Tag.Get("json"))

	field, ok = rt.FieldByName("Content")
	require.True(t, ok)
	assert.Equal(t, "content", field.Tag.Get("json"))
}

func TestGenerateRequest_HasArticleStyleField(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.GenerateRequest{})
	field, ok := rt.FieldByName("ArticleStyle")
	require.True(t, ok)
	assert.Equal(t, "article_style", field.Tag.Get("json"))
}

func TestGenerateRequest_HasScenarioModeField(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.GenerateRequest{})
	field, ok := rt.FieldByName("ScenarioMode")
	require.True(t, ok)
	assert.Equal(t, "scenario_mode", field.Tag.Get("json"))
}

func TestGenerateRequest_HasPromptProfileFields(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.GenerateRequest{})

	field, ok := rt.FieldByName("PromptProfileKey")
	require.True(t, ok)
	assert.Equal(t, "prompt_profile_key", field.Tag.Get("json"))

	field, ok = rt.FieldByName("DocumentKind")
	require.True(t, ok)
	assert.Equal(t, "document_kind", field.Tag.Get("json"))
}

func TestStreamGenerate_UsesResolvedPromptProfileWhenCallingServices(t *testing.T) {
	data, err := os.ReadFile("../../../../domain/stream/service.go")
	require.NoError(t, err)

	source := string(data)
	assert.Contains(t, source, "ResolvePromptProfileKey(req.PromptProfileKey, scenarioMode)")
	assert.Contains(t, source, "GenerateSeriesWithProfile(")
	assert.Contains(t, source, "GenerateBlogStreamWithProfile(")
}
