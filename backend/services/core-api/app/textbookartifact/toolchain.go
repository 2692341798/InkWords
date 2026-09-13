package textbookartifact

import (
	"fmt"
	"github.com/google/uuid"
	"go/version"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	"regexp"
)

var exactGoToolchain = regexp.MustCompile(`^go[0-9]+\.[0-9]+\.[0-9]+$`)

// ManuscriptGoInputForToolchain binds a new immutable manifest to the operator's
// exact runner version. go.mod still declares the language's minimum version.
func ManuscriptGoInputForToolchain(revision textbookdomain.ChapterRevision, toolchain string) (Input, error) {
	if !exactGoToolchain.MatchString(toolchain) || !version.IsValid(toolchain) || version.Compare(toolchain, "go1.26.0") < 0 {
		return Input{}, fmt.Errorf("exact teaching Go toolchain is not configured")
	}
	input, err := ManuscriptGoInput(revision)
	if err != nil {
		return Input{}, err
	}
	input.ToolchainVersion = toolchain
	// Build metadata is part of the content-addressed tree. Pinning the
	// toolchain creates a new artifact without changing either teaching file.
	input.Files[0].Content = append(input.Files[0].Content, []byte("\ntoolchain "+toolchain+"\n")...)
	input.ArtifactID = uuid.NewSHA1(uuid.NameSpaceURL, []byte("inkwords:manuscript-go:v2:"+revision.ID.String()+":"+revision.ContentHash+":"+toolchain))
	return input, nil
}

// WithGoToolchainVersion selects an operator-owned runtime expectation. A
// mismatch with the actual runner remains stale, never an implicit pass.
func (persister *SampleProjectionPersister) WithGoToolchainVersion(toolchain string) *SampleProjectionPersister {
	persister.goToolchain = &toolchain
	return persister
}
