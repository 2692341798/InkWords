package textbookartifact

import (
	"context"
	"log"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
)

type sampleResultWriter interface {
	PersistTextbookSampleResult(context.Context, uuid.UUID, map[string]any) error
}

type generatedRevisionResolver interface {
	GetGeneratedRevisionContext(context.Context, uuid.UUID) (*textbookdomain.GeneratedRevisionContext, error)
}

type artifactProjectionStager interface {
	StageAndRegister(context.Context, uuid.UUID, Input) (*textbookdomain.CodeArtifactRow, error)
}

// SampleProjectionPersister decorates the authoritative candidate write. A
// runtime artifact is a useful projection of a manuscript, not its parent:
// failure to stage it must never delete, overwrite, or invalidate the saved
// candidate revision.
type SampleProjectionPersister struct {
	writer      sampleResultWriter
	resolver    generatedRevisionResolver
	stager      artifactProjectionStager
	goToolchain *string
}

func NewSampleProjectionPersister(writer sampleResultWriter, resolver generatedRevisionResolver, stager artifactProjectionStager) *SampleProjectionPersister {
	return &SampleProjectionPersister{writer: writer, resolver: resolver, stager: stager}
}

func (persister *SampleProjectionPersister) PersistTextbookSampleResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	if err := persister.writer.PersistTextbookSampleResult(ctx, taskID, result); err != nil {
		return err
	}
	if persister.resolver == nil || persister.stager == nil {
		log.Printf("textbook artifact projection unavailable after sample task %s; candidate remains saved and unverified", taskID)
		return nil
	}
	generated, err := persister.resolver.GetGeneratedRevisionContext(ctx, taskID)
	if err != nil {
		log.Printf("resolve generated textbook revision after task %s: %v; candidate remains saved and unverified", taskID, err)
		return nil
	}
	if generated == nil {
		log.Printf("missing generated revision after task %s; candidate remains saved and unverified", taskID)
		return nil
	}
	input, err := ManuscriptGoInput(generated.Revision)
	if persister.goToolchain != nil {
		input, err = ManuscriptGoInputForToolchain(generated.Revision, *persister.goToolchain)
	}
	if err != nil {
		log.Printf("derive manuscript teaching artifact after task %s: %v; candidate remains saved and unverified", taskID, err)
		return nil
	}
	input.BookContractHash, input.StyleSheetHash = generated.BookContractHash, generated.StyleSheetHash
	if _, err := persister.stager.StageAndRegister(ctx, generated.WorkspaceID, input); err != nil {
		log.Printf("stage manuscript teaching artifact after task %s: %v; candidate remains saved and unverified", taskID, err)
	}
	return nil
}
