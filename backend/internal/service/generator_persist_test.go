package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/model"
)

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

	service := &GeneratorService{
		llmClient: &llm.DeepSeekClient{
			APIKey: "test-key",
			APIURL: fakeLLM.URL,
			Client: fakeLLM.Client(),
		},
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

	service := &GeneratorService{
		llmClient: &llm.DeepSeekClient{
			APIKey: "test-key",
			APIURL: fakeLLM.URL,
			Client: fakeLLM.Client(),
		},
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

func newGeneratorPersistTestLLMServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[\"Go\",\"Docker\"]"}}]}`))
	}))
}
