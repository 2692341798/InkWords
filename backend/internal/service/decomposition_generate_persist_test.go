package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

func TestDecompositionService_GenerateSeries_PersistsChildDraftBeforeStreaming(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	userID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)

	var streamCallCount atomic.Int32
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNumber := streamCallCount.Add(1)
		if callNumber <= 3 {
			http.Error(w, "chapter generation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"系列导读正文\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	}))
	defer llmServer.Close()

	svc := NewDecompositionService(nil)
	svc.llmClient.APIURL = llmServer.URL
	svc.llmClient.Client = llmServer.Client()

	parentID := uuid.New()
	progressChan := make(chan string, 32)
	errChan := make(chan error, 8)

	svc.GenerateSeries(
		context.Background(),
		userID,
		parentID,
		"《孙子兵法》- 原典逐章精读系列",
		[]Chapter{
			{Title: "始计第一", Summary: "逐章精读", Sort: 1},
		},
		"兵者，国之大事。",
		"file",
		"",
		prompt.ScenarioModeEbookInterpretation,
		string(prompt.ArticleStyleGeneral),
		progressChan,
		errChan,
	)

	var children []model.Blog
	require.NoError(t, testDB.Where("parent_id = ?", parentID).Order("chapter_sort ASC").Find(&children).Error)
	require.Len(t, children, 1)
	require.Equal(t, "始计第一", children[0].Title)
}
