package generation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
)

func TestAnalyzeStreamRejectsMissingRepositoryInput(t *testing.T) {
	svc := NewDecompositionService(nil, nil, nil)
	svc.gitFetcher = nil
	progress := make(chan string, 4)
	errs := make(chan error, 1)
	svc.AnalyzeStream(context.Background(), uuid.New(), "", nil, prompt.ScenarioMode("invalid"), progress, errs)
	require.Empty(t, progress)
	require.ErrorContains(t, <-errs, "git url is required")
}

func TestTaskOnlyPersistenceModeDefaultsToTaskOnly(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "")
	require.True(t, taskOnlyPersistenceMode())
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "legacy")
	require.False(t, taskOnlyPersistenceMode())
}

func TestDecodeGeneratedOutlineAcceptsOutlineAlias(t *testing.T) {
	outline, err := decodeGeneratedOutline(`{"title":"脚本工具系列","outline":[{"title":"构建脚本","summary":"说明构建流程","sort":1}]}`)
	require.NoError(t, err)
	require.Equal(t, "脚本工具系列", outline.SeriesTitle)
	require.Len(t, outline.Chapters, 1)
}

func TestSeriesPureHelpersCoverBoundaries(t *testing.T) {
	require.Contains(t, buildSeriesReaderProfile(prompt.ScenarioModeBeginnerWalkthrough), "零基础")
	require.Contains(t, buildSeriesReaderProfile(prompt.ScenarioModeOpenBookExamReview), "复习")
	require.Contains(t, buildSeriesReaderProfile(prompt.ScenarioModeEbookInterpretation), "原理")

	t.Setenv("LLM_PRO_CONCURRENCY", "2")
	t.Setenv("LLM_FLASH_CONCURRENCY", "4")
	require.Equal(t, 2, maxWorkersForModel("deepseek-pro", 10))
	require.Equal(t, 3, maxWorkersForModel("deepseek-flash", 3))
	require.Equal(t, 1, maxWorkersFromEnv(1))

	require.Equal(t, "short", truncateSeriesContent("short", 10))
	require.Contains(t, truncateSeriesContent("这是很长的内容", 3), seriesContentTruncatedSuffix)
	require.Equal(t, "fallback", resolveSeriesChapterSourceContent("file", "", "fallback", sharedblog.Chapter{}))
	require.Empty(t, decodeTechStacksJSON(nil))
	require.Empty(t, decodeTechStacksJSON(json.RawMessage(`not-json`)))
	require.Equal(t, []string{"Go"}, decodeTechStacksJSON(json.RawMessage(`["Go"]`)))

	extra := buildSeriesChapterExtraRequirements("https://example/repo", []sharedblog.Chapter{{Title: "当前"}, {Title: "下一章"}}, 0)
	require.Contains(t, extra, "源码仓库引用")
	require.Contains(t, extra, "下期预告")

	var builder strings.Builder
	appendSeriesFileSource(&builder, t.TempDir(), "missing.go")
	require.Empty(t, builder.String())
}

func TestQuotaServiceAllowsWithinLimitAndRejectsMissingOrExhausted(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id text primary key, tokens_used integer, token_limit integer)`).Error)
	svc := NewQuotaService(db)
	uid := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, tokens_used, token_limit) VALUES (?, ?, ?)`, uid, 3, 10).Error)
	require.NoError(t, svc.CheckQuota(uid))
	require.ErrorContains(t, svc.CheckQuota(uuid.New()), "user not found")
	require.NoError(t, db.Exec(`UPDATE users SET tokens_used = token_limit WHERE id = ?`, uid).Error)
	require.ErrorContains(t, svc.CheckQuota(uid), "额度已耗尽")
}
