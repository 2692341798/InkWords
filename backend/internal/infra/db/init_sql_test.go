package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Why: CREATE DATABASE 不能运行在事务块里；这个测试直接锁定 review
// 初始化脚本必须使用顶层语句，避免 CI 在空 pgdata 首次启动时把 db 容器拉成 unhealthy。
//nolint:gosec
func TestCreateReviewDatabaseScript_UsesTopLevelStatement(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}

	scriptPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "db", "init", "00-create-review-db.sql")
	contentBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read init sql: %v", err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "CREATE DATABASE inkwords_review_db;") {
		t.Fatalf("expected top-level CREATE DATABASE statement")
	}
	if strings.Contains(content, "DO $$") {
		t.Fatalf("CREATE DATABASE must not be wrapped in DO $$ transaction block")
	}
}
