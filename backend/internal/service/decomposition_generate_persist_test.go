package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
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

func TestGenerateSeries_PersistsFinalChapterFromQualityPipeline(t *testing.T) {
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

	var callCount atomic.Int32
	var requestKinds []string
	var requestKindsMu sync.Mutex
	jsonResponses := []string{
		`{"chapter_goal":"解释请求流转","reader_questions":["请求如何进入 handler"],"must_explain":["路由树匹配"],"must_include_examples":["curl 请求"],"avoid_overlap":[],"bridge_context":{"from_previous":"上一章介绍启动","to_next":"下一章介绍中间件"}}`,
		`{"depth_issues":["需要补充中间件短路"],"example_issues":["需要 curl 示例"],"structure_issues":[],"revision_actions":["补充中间件短路说明","补充 curl 复现"],"scorecard":{"depth":4,"examples":4,"reproducibility":4,"clarity":4}}`,
		`["Gin"]`,
	}
	textResponses := []string{
		`{"draft_markdown":"## Gin 路由\n\n终稿前草稿","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"code","supports_claim":"说明路由注册"}]}`,
	}
	streamResponses := []string{"终稿正文", "系列导读正文"}
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var request struct {
			Stream         bool              `json:"stream"`
			ResponseFormat map[string]string `json:"response_format"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))

		callCount.Add(1)
		requestKindsMu.Lock()
		defer requestKindsMu.Unlock()

		switch {
		case request.Stream:
			require.NotEmpty(t, streamResponses)
			requestKinds = append(requestKinds, "stream")
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", streamResponses[0])
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			streamResponses = streamResponses[1:]
		case request.ResponseFormat["type"] == "json_object":
			require.NotEmpty(t, jsonResponses)
			requestKinds = append(requestKinds, "json")
			fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, jsonResponses[0])
			jsonResponses = jsonResponses[1:]
		default:
			require.NotEmpty(t, textResponses)
			requestKinds = append(requestKinds, "text")
			fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, textResponses[0])
			textResponses = textResponses[1:]
		}
	}))
	defer llmServer.Close()

	svc := NewDecompositionService(nil)
	svc.llmClient.APIURL = llmServer.URL
	svc.llmClient.Client = llmServer.Client()

	parentID := uuid.New()
	progressChan := make(chan string, 64)
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
	require.Equal(t, "终稿正文", children[0].Content)
	require.EqualValues(t, 1, children[0].Status)
	require.GreaterOrEqual(t, callCount.Load(), int32(6))
	require.True(t, slices.Equal([]string{"json", "text", "json", "stream"}, requestKinds[:4]))
}
