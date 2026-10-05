package textbookartifact

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type dependencyProjectionResolver struct {
	selectionResolver
	generated *textbookdomain.GeneratedRevisionContext
}

func (r *dependencyProjectionResolver) GetGeneratedRevisionContext(context.Context, uuid.UUID) (*textbookdomain.GeneratedRevisionContext, error) {
	return r.generated, nil
}

type projectionCatalog struct {
	selected *SelectedOfflineGoBundle
	hash     string
	loads    int
}

func (c *projectionCatalog) Resolve(_ context.Context, workspace, chapter uuid.UUID, id string) (*SelectedOfflineGoBundle, string, error) {
	c.loads++
	if id != "prepared" || c.selected.selection.WorkspaceID != workspace.String() || c.selected.selection.ChapterID != chapter.String() {
		return nil, "", textbookdomain.ErrNotFound
	}
	return c.selected, c.hash, nil
}

func TestDependencyProjectionRequiresFreshExactConfirmationBeforeStaging(t *testing.T) {
	bundle := projectionDependencyBundle(t, "prepared dependency")
	w, p, ch, task := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	selection := sharedtextbook.TeachingDependencySelection{WorkspaceID: w.String(), ProjectID: p.String(), ChapterID: ch.String(), Snapshot: sharedtextbook.SourceSnapshot{ID: uuid.NewString(), SourceID: uuid.NewString(), Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://example.com/library", ResolvedVersion: bundle.Origin().Commit, ContentHash: "sha256:" + strings.Repeat("d", 64), CapturedAt: time.Now().UTC()}, Module: bundle.Origin().Module, Version: bundle.Origin().Version, Toolchain: bundle.Toolchain(), DependencyManifestHash: bundle.Hash()}
	revision := manuscriptRevision(projectionManuscript("candidate"))
	revision.ChapterID = ch
	resolver := &dependencyProjectionResolver{selectionResolver: selectionResolver{snapshot: selection.Snapshot}, generated: &textbookdomain.GeneratedRevisionContext{WorkspaceID: w, Revision: revision, BookContractHash: "sha256:" + strings.Repeat("b", 64), StyleSheetHash: "sha256:" + strings.Repeat("c", 64)}}
	catalog := &projectionCatalog{selected: &SelectedOfflineGoBundle{selection: selection, bundle: bundle}, hash: "sha256:" + strings.Repeat("e", 64)}
	stager := &recordingProjectionStager{}
	service := NewDependencyProjectionService(resolver, catalog, stager)
	request := DependencyProjectionRequest{OptionID: "prepared", TaskID: task, RevisionID: revision.ID, ContentHash: revision.ContentHash}
	preview, err := service.Preview(t.Context(), w, ch, request)
	require.NoError(t, err)
	require.NotEmpty(t, preview.ConfirmationHash)
	require.Equal(t, uuid.Nil, stager.input.ArtifactID)
	require.Equal(t, selection, preview.Selection)
	_, err = service.Apply(t.Context(), w, ch, request, "")
	require.ErrorIs(t, err, ErrDependencyConfirmationChanged)
	require.Equal(t, uuid.Nil, stager.input.ArtifactID)
	catalog.hash = "sha256:" + strings.Repeat("f", 64)
	_, err = service.Apply(t.Context(), w, ch, request, preview.ConfirmationHash)
	require.ErrorIs(t, err, ErrDependencyConfirmationChanged)
	require.Equal(t, uuid.Nil, stager.input.ArtifactID)
	preview, err = service.Preview(t.Context(), w, ch, request)
	require.NoError(t, err)
	_, err = service.Apply(t.Context(), w, ch, request, preview.ConfirmationHash)
	require.NoError(t, err)
	require.Equal(t, preview.ArtifactID, stager.input.ArtifactID)
	require.GreaterOrEqual(t, catalog.loads, 4)

	stager.input = Input{}
	_, err = service.Preview(t.Context(), uuid.New(), ch, request)
	require.Error(t, err)
	request.ContentHash = strings.Repeat("0", 64)
	_, err = service.Preview(t.Context(), w, ch, request)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, stager.input.ArtifactID)
	request.ContentHash = revision.ContentHash
	resolver.snapshot.ContentHash = "sha256:" + strings.Repeat("0", 64)
	_, err = service.Apply(t.Context(), w, ch, request, preview.ConfirmationHash)
	require.Error(t, err)
	require.Equal(t, uuid.Nil, stager.input.ArtifactID)
}
