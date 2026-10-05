package parse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	crawldomain "inkwords-backend/services/parser-service/domain/crawl"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	parserinfra "inkwords-backend/shared/platform/parser"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
)

func TestOfficialWebV2PreservesAuditAndSeparatesLegacySnapshots(t *testing.T) {
	payload := sharedtextbook.OfficialWebImportTaskPayload{TaskVersion: 2, ParserVersion: sharedtextbook.OfficialWebParserVersion, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, ProjectID: "project", SourceID: "source", SnapshotID: "snapshot", SourceKind: sharedtextbook.SourceKindOfficialWeb, SourceRole: sharedtextbook.SourceRoleOfficial, EntryURL: "https://gin-gonic.com/en/docs/", AllowedPathPrefixes: []string{"/en/docs"}}
	payload.InputHash = sharedtextbook.OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes, payload.ParserVersion)
	policy, err := crawldomain.DefaultPolicy(payload.EntryURL)
	require.NoError(t, err)
	policy.AllowedPathPrefixes = payload.AllowedPathPrefixes
	body := []byte("<nav>Navigation noise</nav><main><h1>Gin</h1><pre data-language=go>package main\nfunc main() {}</pre></main>")
	digest := sha256.Sum256(body)
	page := crawldomain.PageManifest{URL: payload.EntryURL, CanonicalURL: payload.EntryURL, ContentHash: "sha256:" + hex.EncodeToString(digest[:]), ContentType: "text/html", StatusCode: 200, FetchedAt: time.Now().UTC()}
	manifest := crawldomain.Manifest{EntryURL: payload.EntryURL, Policy: policy, Status: "complete", TotalBytes: int64(len(body)), Pages: []crawldomain.PageManifest{page}, Decisions: []crawldomain.EntryDecision{{URL: "https://other.example/", Status: "skipped", Reason: "cross_origin"}}}
	manifest.SnapshotInputHash = crawldomain.SnapshotInputHash(manifest)
	crawler := &fakeOfficialWebCrawler{manifest: manifest, pages: []crawldomain.CapturedPage{{Manifest: page, Body: body}}}
	tasks := &fakeParseTaskService{}
	consumer := NewTaskConsumer(tasks, &stubParseTaskService{officialWebFunc: func(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
		require.False(t, request.LegacyOfficialWeb)
		require.True(t, request.PreserveOfficialWebTabs)
		return parserinfra.NewStructuredParser().ParseOfficialWebHTML(source, request)
	}}).WithOfficialWebCrawler(crawler)
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	workspaceID := uuid.New()
	message := sharedmq.ParseRequestedMessage{TaskID: uuid.New(), Kind: payload.TaskSubtype, WorkspaceID: &workspaceID, Payload: raw}
	require.NoError(t, consumer.HandleParseRequested(context.Background(), message))
	require.Equal(t, "succeeded", tasks.lastStatus, tasks.lastErrorMessage)
	var result sharedtextbook.OfficialWebImportTaskResult
	require.NoError(t, json.Unmarshal(tasks.lastResult, &result))
	require.NoError(t, result.ValidateAgainst(payload))
	require.NotEqual(t, manifest.SnapshotInputHash, result.ContentHash)
	require.Contains(t, string(result.CrawlManifest), "cross_origin")
	for _, chunk := range result.Chunks {
		require.NotContains(t, chunk.SearchText, "Navigation noise")
	}
	require.Equal(t, "package main\nfunc main() {}", result.Chunks[len(result.Chunks)-1].SearchText)
	for name, mutate := range map[string]func(*sharedtextbook.OfficialWebImportTaskResult){
		"missing audit": func(r *sharedtextbook.OfficialWebImportTaskResult) { r.CrawlManifest = nil },
		"oversized audit": func(r *sharedtextbook.OfficialWebImportTaskResult) {
			r.CrawlManifest = []byte(strings.Repeat(" ", (8<<20)+1))
		},
		"parser mismatch": func(r *sharedtextbook.OfficialWebImportTaskResult) { r.ParserVersion = "future" },
		"partial": func(r *sharedtextbook.OfficialWebImportTaskResult) {
			r.CrawlManifest = []byte(strings.Replace(string(r.CrawlManifest), `"complete"`, `"partial"`, 1))
		},
		"wrong boundary": func(r *sharedtextbook.OfficialWebImportTaskResult) {
			r.CrawlManifest = []byte(strings.Replace(string(r.CrawlManifest), `"/en/docs"`, `"/"`, 1))
		},
		"missing page": func(r *sharedtextbook.OfficialWebImportTaskResult) {
			r.Documents = append([]sharedtextbook.SourceDocument(nil), r.Documents...)
			r.Documents[0].CanonicalLocator += "missing"
		},
		"old content identity": func(r *sharedtextbook.OfficialWebImportTaskResult) {
			r.ContentHash = r.ManifestHash
			r.ResolvedVersion = r.ContentHash
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := result
			mutate(&changed)
			require.Error(t, changed.ValidateAgainst(payload))
		})
	}
	changed := payload
	changed.AllowedPathPrefixes = []string{"/"}
	require.Error(t, changed.Validate(), "input hash must freeze the boundary")
	legacy := payload
	legacy.ParserVersion = sharedtextbook.OfficialWebVisibleContentParserVersion
	legacy.InputHash = sharedtextbook.OfficialWebImportInputHash(legacy.ProjectID, legacy.SourceID, legacy.SnapshotID, legacy.EntryURL, legacy.AllowedPathPrefixes, legacy.ParserVersion)
	legacyRaw, err := json.Marshal(legacy)
	require.NoError(t, err)
	legacyMessage := message
	legacyMessage.Payload = legacyRaw
	legacyConsumer := NewTaskConsumer(tasks, &stubParseTaskService{officialWebFunc: func(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
		require.False(t, request.LegacyOfficialWeb)
		require.False(t, request.PreserveOfficialWebTabs, "frozen v2 tasks keep their original projection")
		return parserinfra.NewStructuredParser().ParseOfficialWebHTML(source, request)
	}}).WithOfficialWebCrawler(crawler)
	require.NoError(t, legacyConsumer.HandleParseRequested(context.Background(), legacyMessage))
	require.Equal(t, "succeeded", tasks.lastStatus)
	crawler.manifest.SnapshotInputHash = "sha256:" + strings.Repeat("a", 64)
	require.NoError(t, consumer.HandleParseRequested(context.Background(), message))
	require.Equal(t, "failed", tasks.lastStatus)
	require.Contains(t, tasks.lastErrorMessage, "manifest hash")
}
