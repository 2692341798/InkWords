package service

import (
	"context"
	"fmt"
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

func TestContinueGeneration_TaskOnlyMode_DoesNotUpdateBlogDirectly(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	blogID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         blogID,
		UserID:     userID,
		Title:      "旧标题",
		Content:    "旧内容",
		SourceType: "file",
		Status:     1,
	}).Error)

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	fakeLLM := newContinueGenerationStreamServer(t, "追加内容")
	defer fakeLLM.Close()

	service := &DecompositionService{
		llmClient: &llm.DeepSeekClient{
			APIKey: "test-key",
			APIURL: fakeLLM.URL,
			Client: fakeLLM.Client(),
		},
	}

	chunkChan := make(chan string, 8)
	errChan := make(chan error, 1)
	service.ContinueGeneration(context.Background(), userID, blogID, chunkChan, errChan)

	var appendedContent string
	for chunk := range chunkChan {
		appendedContent += chunk
	}
	for err := range errChan {
		require.NoError(t, err)
	}

	var blog model.Blog
	require.NoError(t, testDB.First(&blog, "id = ?", blogID).Error)
	require.Equal(t, "旧内容", blog.Content)

	result, err := service.BuildContinueTaskResult(context.Background(), userID, blogID, appendedContent)
	require.NoError(t, err)
	require.Equal(t, blogID.String(), result.BlogID)
	require.Equal(t, "追加内容", result.AppendedContent)
	require.Equal(t, "旧内容追加内容", result.FinalContent)
	require.Equal(t, len([]rune("追加内容"))*2, result.EstimatedTokens)
}

func newContinueGenerationStreamServer(t *testing.T, appendedContent string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", appendedContent)
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}
