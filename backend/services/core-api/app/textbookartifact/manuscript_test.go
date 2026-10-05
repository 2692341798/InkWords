package textbookartifact

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

func manuscriptRevision(markdown string) textbookdomain.ChapterRevision {
	return textbookdomain.ChapterRevision{ID: uuid.New(), Kind: sharedtextbook.RevisionKindCandidate, CreatedBy: textbookdomain.RevisionCreatorGeneration, Markdown: markdown, ContentHash: fmt.Sprintf("%x", sha256.Sum256([]byte(markdown)))}
}

func TestManuscriptProjectionPreservesBytesAndChangesArtifactWithSource(t *testing.T) {
	first := manuscriptRevision(projectionManuscript("first-value"))
	first.Markdown = strings.Replace(first.Markdown, "func main()", "// func Test is a comment, not a declaration.\nfunc main()", 1)
	first.ContentHash = fmt.Sprintf("%x", sha256.Sum256([]byte(first.Markdown)))
	input, err := ManuscriptGoInput(first)
	require.NoError(t, err)
	require.Equal(t, "package main\n\n// func Test is a comment, not a declaration.\nfunc main() { println(\"first-value\") }\n", string(input.Files[1].Content))
	require.Contains(t, input.Limitations[0], "省略并发处理")
	require.Len(t, input.Commands, 1)
	require.Equal(t, "go_test", input.Commands[0].Kind)
	store := teachingartifact.NewStore(t.TempDir())
	staged, err := store.Stage(context.Background(), input.Files)
	require.NoError(t, err)
	again, err := ManuscriptGoInput(first)
	require.NoError(t, err)
	require.Equal(t, input, again)
	second := manuscriptRevision(projectionManuscript("different-value"))
	second.ID = first.ID
	changed, err := ManuscriptGoInput(second)
	require.NoError(t, err)
	require.NotEqual(t, input.ArtifactID, changed.ArtifactID)
	changedTree, err := store.Stage(context.Background(), changed.Files)
	require.NoError(t, err)
	require.NotEqual(t, staged.ArtifactHash, changedTree.ArtifactHash)
}

func TestManuscriptProjectionRejectsAmbiguousOrChangedSources(t *testing.T) {
	for name, mutate := range map[string]func(*textbookdomain.ChapterRevision){
		"empty":      func(r *textbookdomain.ChapterRevision) { r.Markdown = "" },
		"wrong hash": func(r *textbookdomain.ChapterRevision) { r.ContentHash = strings.Repeat("0", 64) },
		"manual":     func(r *textbookdomain.ChapterRevision) { r.CreatedBy = "manual" },
		"approved":   func(r *textbookdomain.ChapterRevision) { r.Kind = sharedtextbook.RevisionKindApproved },
		"missing origin": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "代码来源", "来源")
		},
		"upstream origin": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "教学实现", "上游源码讲解")
		},
		"wrong filename": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "（main.go）", "（../../main.go）")
		},
		"missing boundary": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "省略", "未实现")
		},
		"duplicate":     func(r *textbookdomain.ChapterRevision) { r.Markdown += "\n" + r.Markdown },
		"extra command": func(r *textbookdomain.ChapterRevision) { r.Markdown += "\n```sh\necho unsafe\n```\n" },
		"quoted": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = "> " + strings.ReplaceAll(r.Markdown, "\n", "\n> ")
		},
		"syntax": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "func main()", "func main(")
		},
		"comment-only test": func(r *textbookdomain.ChapterRevision) {
			r.Markdown = strings.ReplaceAll(r.Markdown, "func TestMainOutput", "// func TestMainOutput")
		},
	} {
		t.Run(name, func(t *testing.T) {
			revision := manuscriptRevision(projectionManuscript("private-answer-marker"))
			mutate(&revision)
			if name != "wrong hash" {
				revision.ContentHash = fmt.Sprintf("%x", sha256.Sum256([]byte(revision.Markdown)))
			}
			_, err := ManuscriptGoInput(revision)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private-answer-marker")
		})
	}
}

func TestFailedManuscriptProjectionNeverStagesFallbackOrLosesCandidate(t *testing.T) {
	writer := &recordingSampleWriter{}
	stager := &recordingProjectionStager{}
	for _, generated := range []*textbookdomain.GeneratedRevisionContext{nil, {Revision: manuscriptRevision("No generated code here.")}} {
		persister := NewSampleProjectionPersister(writer, fakeRevisionResolver{context: generated}, stager)
		require.NoError(t, persister.PersistTextbookSampleResult(context.Background(), uuid.New(), nil))
		require.True(t, writer.called)
		require.Equal(t, uuid.Nil, stager.input.ArtifactID)
	}
}
