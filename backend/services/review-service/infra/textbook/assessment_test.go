package textbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestAssessmentSourceClientChecksOriginIdentityAndExcerptIntegrity(t *testing.T) {
	owner, chapter, revision := uuid.New(), uuid.New(), uuid.New()
	excerpt := "冻结来源原文\n```go\nrouter.GET(\"/ping\", handler)\n```"
	snapshot := sharedtextbook.SourceSnapshot{ID: "snapshot", SourceID: "source", Kind: sharedtextbook.SourceKindOfficialWeb, Role: sharedtextbook.SourceRoleOfficial, Locator: "https://gin-gonic.com/docs/", ResolvedVersion: "snapshot-2026-09-05", ContentHash: "sha256:source", CapturedAt: time.Now().UTC()}
	projection := sharedtextbook.PracticeEvidenceProjection{Format: "inkwords.practice-evidence.v1", WorkspaceID: owner.String(), ChapterID: chapter.String(), RevisionID: revision.String(), ContentHash: "sha256:revision", TaskID: "task/one?", Sources: []sharedtextbook.PracticeSourceExcerpt{{Reference: sharedtextbook.EvidenceRef{ID: "evidence-1", SnapshotID: snapshot.ID, DocumentID: "document", ChunkID: "chunk", Locator: sharedtextbook.EvidenceLocator{URL: snapshot.Locator}, ContentHash: "sha256:chunk", Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: snapshot.Role}, Snapshot: snapshot, Excerpt: excerpt, ExcerptHash: sharedtextbook.PracticeExcerptHash(excerpt)}}}
	status, oversized, redirected := 200, false, false
	malformed := false
	projection.Sources[0].Reference.Locator.HeadingPath = []string{"路由"}
	require.NoError(t, snapshot.Validate())
	require.NoError(t, projection.Sources[0].Reference.Validate())
	require.NoError(t, projection.Validate())
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/textbook-projects/chapters/"+chapter.String()+"/practice-evidence", r.URL.Path)
		require.Equal(t, revision.String(), r.URL.Query().Get("revision_id"))
		require.Equal(t, "task/one?", r.URL.Query().Get("task_id"))
		if status == 302 {
			w.Header().Set("Location", target.URL)
		}
		w.WriteHeader(status)
		if status != 200 {
			_, _ = w.Write([]byte("private source service error"))
			return
		}
		if oversized {
			_, _ = w.Write([]byte(strings.Repeat("x", maxProjectionBytes+1)))
			return
		}
		if malformed {
			_, _ = w.Write([]byte(`{"code":0,"data":`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": projection})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	require.NoError(t, err)
	read := func() error {
		_, err := client.LoadApprovedAssessmentEvidence(context.Background(), owner, chapter, revision, "task/one?")
		return err
	}
	require.NoError(t, read(), "official-only task evidence is valid without adding unrelated primary text")
	projection.RevisionID = uuid.New().String()
	require.ErrorContains(t, read(), "mismatch")
	projection.RevisionID = revision.String()
	projection.Sources[0].Excerpt = "changed"
	require.ErrorContains(t, read(), "mismatch")
	projection.Sources[0].Excerpt = excerpt
	malformed = true
	require.ErrorContains(t, read(), "response")
	malformed = false
	oversized = true
	require.ErrorContains(t, read(), "size")
	oversized = false
	status = 500
	require.NotContains(t, read().Error(), "private")
	status = 302
	require.Error(t, read())
	require.False(t, redirected)
}
