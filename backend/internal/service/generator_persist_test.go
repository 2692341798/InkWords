package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

func TestGeneratorService_saveToDB_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	fakeLLM := newGeneratorPersistTestLLMServer(t)
	defer fakeLLM.Close()

	persistence := &generatorPersistenceRecorder{}
	userID := uuid.New()
	service := NewGeneratorServiceWithPersistence(
		nil,
		persistence,
	)
	service.llmClient = &llm.DeepSeekClient{
		APIKey: "test-key",
		APIURL: fakeLLM.URL,
		Client: fakeLLM.Client(),
	}

	err := service.saveToDB(context.Background(), userID, "file", "hello")
	require.NoError(t, err)
	require.Equal(t, 1, persistence.calls)
	require.Equal(t, userID, persistence.saved.UserID)
	require.Equal(t, "文件解析生成的博客", persistence.saved.Title)
	require.Equal(t, "file", persistence.saved.SourceType)
	require.Equal(t, "hello", persistence.saved.Content)
	require.Equal(t, 5, persistence.saved.WordCount)
	require.JSONEq(t, `["Go","Docker"]`, string(persistence.saved.TechStacks))
}

func TestGeneratorService_saveToDB_RollsBackWhenUserTokenUpdateFails(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	fakeLLM := newGeneratorPersistTestLLMServer(t)
	defer fakeLLM.Close()

	service := NewGeneratorService(nil)
	service.llmClient = &llm.DeepSeekClient{
		APIKey: "test-key",
		APIURL: fakeLLM.URL,
		Client: fakeLLM.Client(),
	}

	err = service.saveToDB(context.Background(), uuid.New(), "file", "hello world")
	require.Error(t, err)
	require.ErrorContains(t, err, "persist generated blog")

	var count int64
	require.NoError(t, testDB.Model(&model.Blog{}).Count(&count).Error)
	require.EqualValues(t, 0, count)
}

func TestGeneratorService_saveToDB_PersistsBlogAndUpdatesTokens(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	fakeLLM := newGeneratorPersistTestLLMServer(t)
	defer fakeLLM.Close()

	service := NewGeneratorService(nil)
	service.llmClient = &llm.DeepSeekClient{
		APIKey: "test-key",
		APIURL: fakeLLM.URL,
		Client: fakeLLM.Client(),
	}

	require.NoError(t, service.saveToDB(context.Background(), userID, "file", "hello"))

	var blog model.Blog
	require.NoError(t, testDB.First(&blog).Error)
	require.Equal(t, userID, blog.UserID)
	require.Equal(t, "file", blog.SourceType)
	require.Equal(t, "hello", blog.Content)
	require.Equal(t, 5, blog.WordCount)

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 10, user.TokensUsed)
}

func TestGenerateBlogStream_DoesNotPersistBlogDirectlyWhenTaskModeEnabled(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")

	svc := NewGeneratorService(nil)
	require.NotNil(t, svc)
	require.True(t, taskOnlyPersistenceMode())
}

func TestNewGeneratorServiceWithPersistence_FillsDefaultPersistence(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	svc := NewGeneratorServiceWithPersistence(nil, nil)
	require.NotNil(t, svc.persistence)
}

func TestGeneratorService_BuildGenerateSingleTaskResult_TaskOnlyModeDoesNotPersistAndReturnsTaskResult(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")

	fakeLLM := newGeneratorPersistTestLLMServer(t)
	defer fakeLLM.Close()

	persistence := &generatorPersistenceRecorder{}
	svc := NewGeneratorServiceWithPersistence(nil, persistence)
	svc.llmClient = &llm.DeepSeekClient{
		APIKey: "test-key",
		APIURL: fakeLLM.URL,
		Client: fakeLLM.Client(),
	}

	result, err := svc.BuildGenerateSingleTaskResult(context.Background(), "file", "# 标题\n\n正文")
	require.NoError(t, err)
	require.Zero(t, persistence.calls)
	require.NotEmpty(t, result.ResultJSON)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(result.ResultJSON, &decoded))
	require.Equal(t, "generation", decoded["task_type"])
	require.Equal(t, "generate_single", decoded["task_subtype"])
	require.Equal(t, "task_only", decoded["persistence_mode"])
}

func TestBuildSingleGenerateMessages_UsesResolvedPromptProfileRole(t *testing.T) {
	profile := prompt.ResolvePromptProfileKey(
		"psychology_communication_book",
		prompt.ScenarioModeEbookInterpretation,
	)

	messages := buildSingleGenerateMessages("源内容", "写作要求", profile)

	require.Len(t, messages, 2)
	require.Contains(t, messages[0].Content, "心理学")
	require.Contains(t, messages[0].Content, "项目源内容如下：\n源内容")
	require.NotContains(t, messages[0].Content, "高级全栈架构师")
	require.NotContains(t, messages[1].Content, "高级全栈架构师")
	require.Contains(t, messages[1].Content, "写作要求")
}

func newGeneratorPersistTestLLMServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[\"Go\",\"Docker\"]"}}]}`))
	}))
}

type generatorPersistenceRecorder struct {
	calls int
	saved blogcontracts.GeneratedBlogPersistenceInput
	err   error
}

func (r *generatorPersistenceRecorder) SaveGeneratedBlog(_ context.Context, input blogcontracts.GeneratedBlogPersistenceInput) error {
	r.calls++
	r.saved = input
	return r.err
}
