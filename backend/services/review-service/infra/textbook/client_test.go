package textbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func clientProjectionFixture() sharedtextbook.LearningProjection {
	arc := sharedtextbook.DefaultLearningArc()
	for i := range arc.Stages {
		arc.Stages[i].Objective = "可观察目标"
		arc.Stages[i].SuccessEvidence = []string{"source-1"}
		arc.Stages[i].RecoveryRoute = "回到上一步"
	}
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	modes := []sharedtextbook.LearningTaskMode{"explain", "complete", "reproduce", "transfer", "diagnose", "retain"}
	for _, mode := range modes {
		task := sharedtextbook.PracticeTask{ID: string(mode), Mode: mode, Prompt: "题目 " + string(mode), Variation: "更换路径", ExpectedAnswer: "按条件查找", EvidenceIDs: []string{"source-1"}, Hints: []sharedtextbook.PracticeHint{{Level: 1, Text: "回忆条件"}, {Level: 2, Text: "检查方法"}, {Level: 3, Text: "核对路径"}}, Rubric: sharedtextbook.PracticeRubricDimensions(mode)}
		for i := range task.Rubric {
			task.Rubric[i].Description = "核对 " + task.Rubric[i].ID
		}
		if mode == sharedtextbook.LearningTaskRetain {
			task.MinDelayHours = 24
		}
		set.Tasks = append(set.Tasks, task)
	}
	chapter := uuid.New().String()
	return sharedtextbook.LearningProjection{Format: "inkwords.learning-projection.v2", ChapterID: chapter, RevisionID: uuid.New().String(), ContentHash: "sha256:" + strings.Repeat("a", 64), LearningArc: arc, PracticeSet: &set, EvidenceIDs: []string{"source-1"}, Objectives: []sharedtextbook.LearningObjective{{ID: "one", ChapterID: chapter, Text: "完成练习", RequiredModes: modes}}}
}

func TestClientPinsRevisionAndWorkspaceAndRejectsInvalidResponses(t *testing.T) {
	projection := clientProjectionFixture()
	require.NoError(t, projection.Validate())
	owner := uuid.New()
	responseOwner := owner
	status := http.StatusOK
	oversized := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/textbook-projects/chapters/"+projection.ChapterID+"/projections", r.URL.Path)
		require.Equal(t, projection.RevisionID, r.URL.Query().Get("revision_id"))
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte("sensitive internal response"))
			return
		}
		if oversized {
			_, _ = w.Write([]byte(strings.Repeat("x", maxProjectionBytes+1)))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"workspace_id": responseOwner, "blog": map[string]string{"title": "冻结题目"}, "learning": projection}})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	require.NoError(t, err)
	load := func() error {
		_, err := client.LoadApprovedPractice(context.Background(), owner, uuid.MustParse(projection.ChapterID), uuid.MustParse(projection.RevisionID))
		return err
	}
	require.NoError(t, load())
	responseOwner = uuid.New()
	require.ErrorContains(t, load(), "mismatch")
	responseOwner = owner
	status = http.StatusInternalServerError
	require.NotContains(t, load().Error(), "sensitive")
	status = http.StatusOK
	oversized = true
	require.ErrorContains(t, load(), "size")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.LoadApprovedPractice(ctx, owner, uuid.MustParse(projection.ChapterID), uuid.MustParse(projection.RevisionID))
	require.Error(t, err)
}

func TestClientDoesNotFollowRedirectsOrAcceptCredentialURLs(t *testing.T) {
	for _, raw := range []string{"file:///tmp/source", "http://user:password@localhost", "http://localhost/path", "http://localhost?redirect=1"} {
		_, err := NewClient(raw)
		require.Error(t, err)
	}
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer server.Close()
	client, err := NewClient(server.URL)
	require.NoError(t, err)
	_, err = client.LoadApprovedPractice(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.Error(t, err)
	require.Zero(t, calls)
}
