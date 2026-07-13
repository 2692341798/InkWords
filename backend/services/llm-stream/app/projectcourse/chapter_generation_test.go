package projectcourse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	"inkwords-backend/shared/platform/llm"
)

func TestJSONChapterGeneratorBuildsEvidenceBoundChapterFromLocalMock(t *testing.T) {
	content := "package main\n\nfunc main() {}\n"
	contentHash := contentHash([]byte(content))
	snapshot := sharedkernel.SourceSnapshot{
		RepositoryURL:     "https://github.com/example/project",
		RequestedRef:      "main",
		ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567",
		CapturedAt:        time.Unix(1, 0).UTC(),
	}
	file := InventoryEntry{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: contentHash, Content: content, Size: len(content)}
	evidenceID := "evidence-" + stableID(file.Path+"\x00"+file.ContentHash)
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: []InventoryEntry{file}}
	pack, err := BuildEvidencePack(snapshot, "chapter-1", []string{evidenceID}, graph, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": fmt.Sprintf(`{"markdown":"# 主链路\n## 模块职责\n负责启动。\n## 源码证据\n%s\n## 练习\n说明入口。","claims":[{"claim_id":"claim-1","text":"main.go 提供启动入口","claim_type":"project_fact","confidence":"observed","evidence_ids":["%s"],"status":"verified"}]}`, evidenceID, evidenceID)}}},
		}
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	generator := JSONChapterGenerator{Client: &llm.DeepSeekClient{APIKey: "local-test", APIURL: server.URL, Client: server.Client()}, Model: "local-mock"}
	document, err := generator.Generate(context.Background(), sharedkernel.Chapter{ID: "chapter-1", Title: "主链路", Type: sharedkernel.ChapterModuleDeepDive}, pack, sharedkernel.AudienceProgramming)
	require.NoError(t, err)
	require.Equal(t, "chapter-1", document.ChapterID)
	require.Contains(t, document.Markdown, "模块职责")
	require.Len(t, document.Claims, 1)
	require.Equal(t, evidenceID, document.Claims[0].EvidenceIDs[0])
	require.NoError(t, document.ValidateContract())
}
