package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

func openDecompositionPersistTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(seriesPersistTestDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))
	return testDB
}

func TestOpenDecompositionPersistTestDB_IsolatesUserFixtures(t *testing.T) {
	firstDB := openDecompositionPersistTestDB(t)
	secondDB := openDecompositionPersistTestDB(t)

	firstUserID := uuid.New()
	require.NoError(t, firstDB.Create(&model.User{
		ID:       firstUserID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)

	secondUserID := uuid.New()
	require.NoError(t, secondDB.Create(&model.User{
		ID:       secondUserID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
}

func TestDecompositionService_GenerateSeries_PersistsChildDraftBeforeStreaming(t *testing.T) {
	testDB := openDecompositionPersistTestDB(t)

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
		[]blogcontracts.Chapter{
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

func TestDecompositionService_GenerateSeries_RollsBackPreflightWhenDraftCreationFails(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(seriesPersistTestDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	userID := uuid.New()
	parentID := uuid.New()
	obsoleteChildID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         parentID,
		UserID:     userID,
		Title:      "旧系列",
		Content:    "旧导读",
		SourceType: "file",
		IsSeries:   true,
		Status:     1,
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:          obsoleteChildID,
		UserID:      userID,
		ParentID:    &parentID,
		ChapterSort: 99,
		Title:       "旧章节",
		Content:     "旧内容",
		SourceType:  "file",
		Status:      1,
	}).Error)

	// Why: 这里故意让“创建新章节草稿”失败，验证前置阶段必须整体回滚，
	// 不能出现旧子节点被删掉、但新草稿又没建出来的半成品状态。
	callbackName := "test:fail_series_draft_create"
	require.NoError(t, testDB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		blog, ok := tx.Statement.Dest.(*model.Blog)
		if !ok || blog.ParentID == nil {
			return
		}
		if blog.Content == "正在生成章节内容..." {
			tx.AddError(fmt.Errorf("forced draft create failure"))
		}
	}))
	defer func() {
		testDB.Callback().Create().Remove(callbackName)
	}()

	svc := NewDecompositionService(nil)
	progressChan := make(chan string, 8)
	errChan := make(chan error, 8)

	svc.GenerateSeries(
		context.Background(),
		userID,
		parentID,
		"新系列",
		[]blogcontracts.Chapter{
			{Title: "新章节", Summary: "新摘要", Sort: 1},
		},
		"新内容",
		"file",
		"",
		prompt.ScenarioModeEbookInterpretation,
		string(prompt.ArticleStyleGeneral),
		progressChan,
		errChan,
	)

	var reportedErrs []error
	for err := range errChan {
		if err != nil {
			reportedErrs = append(reportedErrs, err)
		}
	}
	require.NotEmpty(t, reportedErrs)
	require.ErrorContains(t, reportedErrs[0], "forced draft create failure")

	var children []model.Blog
	require.NoError(t, testDB.Where("parent_id = ?", parentID).Order("chapter_sort ASC").Find(&children).Error)
	require.Len(t, children, 1)
	require.Equal(t, obsoleteChildID, children[0].ID)
	require.Equal(t, "旧章节", children[0].Title)
}

func TestDecompositionService_GenerateSeries_ReportsTokenUpdateFailureAndKeepsDraft(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(seriesPersistTestDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBody, _ := io.ReadAll(r.Body)
		body := string(requestBody)

		if strings.Contains(body, "\"stream\":true") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"章节正文\"},\"finish_reason\":null}]}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		var content string
		switch {
		case strings.Contains(body, "当前阶段：章节理解"):
			content = "{\"chapter_goal\":\"目标\",\"reader_questions\":[],\"must_explain\":[\"机制\"],\"must_include_examples\":[\"示例\"],\"avoid_overlap\":[],\"bridge_context\":{\"from_previous\":\"\",\"to_next\":\"\"}}"
		case strings.Contains(body, "当前阶段：章节写作"):
			content = "{\"draft_markdown\":\"# 草稿\\n\\n内容\",\"coverage_check\":{\"goal_covered\":true,\"mechanism_explained\":true,\"examples_present\":true,\"repro_present\":true,\"edge_cases_present\":true},\"example_inventory\":[{\"example_type\":\"code\",\"supports_claim\":\"ok\"}]}"
		case strings.Contains(body, "当前阶段：章节审稿"):
			content = "{\"depth_issues\":[],\"example_issues\":[],\"structure_issues\":[],\"revision_actions\":[\"补强\"],\"scorecard\":{\"depth\":5,\"examples\":5,\"reproducibility\":5,\"clarity\":5}}"
		case strings.Contains(body, "提取出涉及的核心技术栈名称"):
			content = "[\"Go\"]"
		default:
			content = "OK"
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, "{\"choices\":[{\"message\":{\"content\":%q}}]}", content)
	}))
	defer llmServer.Close()

	svc := NewDecompositionService(nil)
	svc.llmClient.APIURL = llmServer.URL
	svc.llmClient.Client = llmServer.Client()

	nonexistentUserID := uuid.New()
	parentID := uuid.New()
	progressChan := make(chan string, 64)
	errChan := make(chan error, 8)

	svc.GenerateSeries(
		context.Background(),
		nonexistentUserID,
		parentID,
		"《孙子兵法》- 原典逐章精读系列",
		[]blogcontracts.Chapter{
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

	progressPayloads := collectSeriesProgressPayloads(t, progressChan)
	require.Empty(t, drainSeriesErrors(errChan))

	var chapterPayloads []map[string]interface{}
	for _, payload := range progressPayloads {
		if payload["chapter_sort"] == float64(1) {
			chapterPayloads = append(chapterPayloads, payload)
		}
	}
	require.NotEmpty(t, chapterPayloads)

	foundChapterError := false
	for _, payload := range chapterPayloads {
		if payload["status"] == "error" && strings.Contains(fmt.Sprint(payload["message"]), "update user tokens") {
			foundChapterError = true
			break
		}
	}
	require.True(t, foundChapterError, "expected chapter error progress payload when token update fails")

	var child model.Blog
	require.NoError(t, testDB.Where("parent_id = ?", parentID).First(&child).Error)
	require.Equal(t, "始计第一", child.Title)
	require.NotEqualValues(t, 1, child.Status)
	require.Equal(t, "正在生成章节内容...", child.Content)
}

func TestHandleSeriesChapterCompletion_TaskOnlyMode_CollectsChapterResultWithoutDirectPersistence(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")

	testDB := openDecompositionPersistTestDB(t)
	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	userID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:         parentID,
		UserID:     userID,
		Title:      "系列父稿",
		Content:    "正在生成系列导读...",
		SourceType: "file",
		IsSeries:   true,
		Status:     0,
	}).Error)
	require.NoError(t, testDB.Create(&model.Blog{
		ID:          childID,
		UserID:      userID,
		ParentID:    &parentID,
		ChapterSort: 1,
		Title:       "第 1 章",
		Content:     "正在生成章节内容...",
		SourceType:  "file",
		Status:      0,
	}).Error)

	svc := NewDecompositionService(nil)
	collector := newSeriesTaskResultCollector(parentID.String(), "系列父稿")
	err := svc.handleSeriesChapterCompletion(
		context.Background(),
		userID,
		parentID,
		"file",
		blogcontracts.Chapter{
			ID:    childID.String(),
			Title: "第 1 章",
			Sort:  1,
		},
		"章节终稿",
		4,
		[]string{"Go"},
		collector,
	)
	require.NoError(t, err)
	require.Len(t, collector.Chapters, 1)
	require.Equal(t, "succeeded", collector.Chapters[0].Status)
	require.Equal(t, "章节终稿", collector.Chapters[0].Content)

	var child model.Blog
	require.NoError(t, testDB.First(&child, "id = ?", childID).Error)
	require.Equal(t, "正在生成章节内容...", child.Content)
	require.EqualValues(t, 0, child.Status)

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 0, user.TokensUsed)
}

func TestDecompositionService_HandleSeriesChapterCompletion_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	persistence := &seriesPersistenceRecorder{}
	svc := NewDecompositionServiceWithSeriesPersistence(nil, persistence)

	userID := uuid.New()
	parentID := uuid.New()
	chapterID := uuid.New()
	err := svc.handleSeriesChapterCompletion(
		context.Background(),
		userID,
		parentID,
		"file",
		blogcontracts.Chapter{
			ID:    chapterID.String(),
			Title: "第 1 章",
			Sort:  1,
		},
		"章节终稿",
		4,
		[]string{"Go"},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, 1, persistence.saveChapterCalls)
	require.Equal(t, userID, persistence.savedChapter.UserID)
	require.Equal(t, parentID, persistence.savedChapter.ParentID)
	require.Equal(t, chapterID, persistence.savedChapter.BlogID)
	require.Equal(t, "章节终稿", persistence.savedChapter.Content)
	require.JSONEq(t, `["Go"]`, string(persistence.savedChapter.TechStacks))
}

func TestNewDecompositionServiceWithPersistences_FillsMissingDefaultAdapters(t *testing.T) {
	testDB := openDecompositionPersistTestDB(t)

	previousDB := db.DB
	db.DB = testDB
	defer func() {
		db.DB = previousDB
	}()

	svc := NewDecompositionServiceWithPersistences(nil, nil, nil)
	require.NotNil(t, svc.seriesPersistence)
	require.NotNil(t, svc.continuePersistence)
}

func TestDecompositionService_GenerateSeriesIntro_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"系列导读正文\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer llmServer.Close()

	persistence := &seriesPersistenceRecorder{}
	svc := NewDecompositionServiceWithSeriesPersistence(nil, persistence)
	svc.llmClient.APIURL = llmServer.URL
	svc.llmClient.Client = llmServer.Client()

	progressChan := make(chan string, 32)
	errChan := make(chan error, 4)
	parentID := uuid.New()
	userID := uuid.New()

	svc.generateSeriesIntro(
		context.Background(),
		userID,
		parentID,
		"Go 入门系列",
		[]blogcontracts.Chapter{{Title: "第 1 章", Summary: "基础", Sort: 1}},
		prompt.ScenarioModeEbookInterpretation,
		prompt.ArticleStyleGeneral,
		prompt.PromptProfile{},
		nil,
		progressChan,
		errChan,
	)

	close(progressChan)
	close(errChan)
	require.Empty(t, drainSeriesErrors(errChan))
	_ = collectSeriesProgressPayloads(t, progressChan)
	require.Equal(t, 1, persistence.saveIntroCalls)
	require.Equal(t, userID, persistence.savedIntroUserID)
	require.Equal(t, parentID, persistence.savedIntroParentID)
	require.Equal(t, "系列导读正文", persistence.savedIntroContent)
}

func TestDecompositionService_ResolveSeriesOldContent_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	blogID := uuid.New()
	userID := uuid.New()
	persistence := &seriesPersistenceRecorder{
		loadedOldContent: strings.Repeat("旧内容", 10),
	}
	svc := NewDecompositionServiceWithSeriesPersistence(nil, persistence)

	oldContent := svc.resolveSeriesOldContent(context.Background(), userID, blogcontracts.Chapter{
		ID:     blogID.String(),
		Action: "regenerate",
	})

	require.Equal(t, 1, persistence.loadOldContentCalls)
	require.Equal(t, userID, persistence.loadedOldContentUserID)
	require.Equal(t, blogID, persistence.loadedOldContentBlogID)
	require.Equal(t, strings.Repeat("旧内容", 10), oldContent)
}

func TestDecompositionService_HandleSkippedSeriesChapter_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	userID := uuid.New()
	blogID := uuid.New()
	persistence := &seriesPersistenceRecorder{}
	svc := NewDecompositionServiceWithSeriesPersistence(nil, persistence)

	err := svc.handleSkippedSeriesChapter(context.Background(), userID, blogcontracts.Chapter{
		ID:    blogID.String(),
		Title: "跳过章节",
		Sort:  3,
	})

	require.NoError(t, err)
	require.Equal(t, 1, persistence.updateSkippedMetaCalls)
	require.Equal(t, userID, persistence.updatedSkippedMetaUserID)
	require.Equal(t, blogID, persistence.updatedSkippedMetaBlogID)
	require.Equal(t, "跳过章节", persistence.updatedSkippedMetaTitle)
	require.Equal(t, 3, persistence.updatedSkippedMetaSort)
}

func TestDecompositionService_EnsureSeriesParentAndDrafts_UsesInjectedPersistence(t *testing.T) {
	previousDB := db.DB
	db.DB = nil
	defer func() {
		db.DB = previousDB
	}()

	userID := uuid.New()
	parentID := uuid.New()
	persistence := &seriesPersistenceRecorder{
		preflightResult: []blogcontracts.Chapter{
			{ID: uuid.NewString(), Title: "第 1 章", Sort: 1},
		},
	}
	svc := NewDecompositionServiceWithSeriesPersistence(nil, persistence)

	updatedOutline, err := svc.ensureSeriesParentAndDrafts(
		context.Background(),
		userID,
		parentID,
		"系列标题",
		"file",
		"https://github.com/example/repo",
		[]blogcontracts.Chapter{{Title: "第 1 章", Sort: 1}},
	)

	require.NoError(t, err)
	require.Len(t, updatedOutline, 1)
	require.Equal(t, persistence.preflightResult[0].ID, updatedOutline[0].ID)
	require.Equal(t, 1, persistence.preflightCalls)
	require.Equal(t, userID, persistence.preflightInput.UserID)
	require.Equal(t, parentID, persistence.preflightInput.ParentID)
	require.Equal(t, "系列标题", persistence.preflightInput.ParentTitle)
	require.Equal(t, "file", persistence.preflightInput.SourceType)
	require.Equal(t, "https://github.com/example/repo", persistence.preflightInput.GitURL)
	require.Len(t, persistence.preflightInput.Outline, 1)
	require.Equal(t, "第 1 章", persistence.preflightInput.Outline[0].Title)
}

func collectSeriesProgressPayloads(t *testing.T, progressChan <-chan string) []map[string]interface{} {
	t.Helper()

	var payloads []map[string]interface{}
	for raw := range progressChan {
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		payloads = append(payloads, payload)
	}

	return payloads
}

func drainSeriesErrors(errChan <-chan error) []error {
	var errs []error
	for err := range errChan {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func seriesPersistTestDSN() string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
}

type seriesPersistenceRecorder struct {
	saveChapterCalls       int
	saveIntroCalls         int
	loadOldContentCalls    int
	updateSkippedMetaCalls int
	preflightCalls         int

	savedChapter             blogcontracts.SeriesChapterPersistenceInput
	savedIntroUserID         uuid.UUID
	savedIntroParentID       uuid.UUID
	savedIntroContent        string
	loadedOldContentUserID   uuid.UUID
	loadedOldContentBlogID   uuid.UUID
	loadedOldContent         string
	updatedSkippedMetaUserID uuid.UUID
	updatedSkippedMetaBlogID uuid.UUID
	updatedSkippedMetaTitle  string
	updatedSkippedMetaSort   int
	preflightInput           blogcontracts.SeriesDraftPreflightInput
	preflightResult          []blogcontracts.Chapter
}

func (r *seriesPersistenceRecorder) SaveSeriesChapter(_ context.Context, input blogcontracts.SeriesChapterPersistenceInput) error {
	r.saveChapterCalls++
	r.savedChapter = input
	return nil
}

func (r *seriesPersistenceRecorder) MarkSeriesChapterFailed(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (r *seriesPersistenceRecorder) SaveSeriesIntro(_ context.Context, userID uuid.UUID, parentID uuid.UUID, content string) error {
	r.saveIntroCalls++
	r.savedIntroUserID = userID
	r.savedIntroParentID = parentID
	r.savedIntroContent = content
	return nil
}

func (r *seriesPersistenceRecorder) MarkSeriesIntroFailed(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (r *seriesPersistenceRecorder) LoadSeriesOldContent(_ context.Context, userID uuid.UUID, blogID uuid.UUID) (string, error) {
	r.loadOldContentCalls++
	r.loadedOldContentUserID = userID
	r.loadedOldContentBlogID = blogID
	return r.loadedOldContent, nil
}

func (r *seriesPersistenceRecorder) UpdateSkippedSeriesChapterMeta(_ context.Context, userID uuid.UUID, blogID uuid.UUID, chapter blogcontracts.Chapter) error {
	r.updateSkippedMetaCalls++
	r.updatedSkippedMetaUserID = userID
	r.updatedSkippedMetaBlogID = blogID
	r.updatedSkippedMetaTitle = chapter.Title
	r.updatedSkippedMetaSort = chapter.Sort
	return nil
}

func (r *seriesPersistenceRecorder) EnsureSeriesParentAndDrafts(_ context.Context, input blogcontracts.SeriesDraftPreflightInput) ([]blogcontracts.Chapter, error) {
	r.preflightCalls++
	r.preflightInput = input
	return append([]blogcontracts.Chapter(nil), r.preflightResult...), nil
}
