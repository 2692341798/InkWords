package textbook

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"time"
)

// freezeBookVerification is called only after the workspace-owned project and
// approved revision list have been resolved inside the build transaction.
func freezeBookVerification(tx *gorm.DB, project Project, artifacts []bookBuildArtifactSource, at time.Time) (sharedtextbook.BookVerificationSnapshot, error) {
	bookHash, styleHash, err := frozenBookBuildContractHashes(tx, BookBuildRow{BookContractRevisionID: *project.ApprovedBookContractRevisionID, StyleSheetRevisionID: *project.ApprovedStyleSheetRevisionID})
	if err != nil {
		return sharedtextbook.BookVerificationSnapshot{}, err
	}
	snapshot := sharedtextbook.BookVerificationSnapshot{Format: sharedtextbook.BookVerificationSnapshotFormat, CapturedAt: at, BookContractHash: bookHash, StyleSheetHash: styleHash, Artifacts: make([]sharedtextbook.FrozenBookVerificationArtifact, 0, len(artifacts))}
	ids := make([]uuid.UUID, 0, len(artifacts))
	for _, a := range artifacts {
		ids = append(ids, a.ID)
	}
	var rows []RuntimeEvidenceRow
	if len(ids) > 0 {
		if err := tx.Where("code_artifact_id IN ?", ids).Order("code_artifact_id ASC, captured_at ASC, id ASC").Find(&rows).Error; err != nil {
			return snapshot, fmt.Errorf("freeze book verification receipts: %w", err)
		}
	}
	evidence := map[uuid.UUID][]sharedtextbook.RuntimeEvidence{}
	for _, row := range rows {
		evidence[row.CodeArtifactID] = append(evidence[row.CodeArtifactID], runtimeEvidenceContract(row))
	}
	for _, a := range artifacts {
		records := evidence[a.ID]
		if records == nil {
			records = []sharedtextbook.RuntimeEvidence{}
		}
		snapshot.Artifacts = append(snapshot.Artifacts, sharedtextbook.FrozenBookVerificationArtifact{ID: a.ID.String(), RevisionID: a.RevisionID.String(), ArtifactHash: a.ArtifactHash, ManifestHash: a.ManifestHash, Manifest: append(json.RawMessage(nil), a.ManifestJSON...), Evidence: records})
	}
	if err := snapshot.Validate(); err != nil {
		return snapshot, fmt.Errorf("validate frozen book verification: %w", err)
	}
	return snapshot, nil
}
