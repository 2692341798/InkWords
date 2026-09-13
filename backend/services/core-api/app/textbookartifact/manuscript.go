package textbookartifact

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

// ManuscriptGoInput projects only the two explicitly labelled Go files in the
// saved candidate. It never repairs code or substitutes a passing fixture.
func ManuscriptGoInput(revision textbookdomain.ChapterRevision) (Input, error) {
	if revision.ID == uuid.Nil || revision.Kind != sharedtextbook.RevisionKindCandidate || revision.CreatedBy != textbookdomain.RevisionCreatorGeneration || fmt.Sprintf("%x", sha256.Sum256([]byte(revision.Markdown))) != revision.ContentHash {
		return Input{}, fmt.Errorf("teaching projection requires an intact generated candidate")
	}
	files, limitations, err := teachingartifact.ManuscriptGoFiles(revision.Markdown)
	if err != nil {
		return Input{}, err
	}
	return Input{
		ArtifactID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("inkwords:manuscript-go:v1:"+revision.ID.String()+":"+revision.ContentHash)),
		RevisionID: revision.ID, Language: "go", ToolchainVersion: "go1.26.0", Entrypoint: "main.go",
		Files: []teachingartifact.File{
			// Build metadata is operator-owned; no dependency downloads or
			// commands are taken from manuscript prose.
			{Path: "go.mod", Content: []byte("module example.com/inkwords/teaching\n\ngo 1.26.0\n")},
			{Path: "main.go", Content: files["main.go"]},
			{Path: "main_test.go", Content: files["main_test.go"]},
		},
		Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}}, Limitations: limitations,
	}, nil
}
