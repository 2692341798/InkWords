package export

import (
	"encoding/json"
	shared "inkwords-backend/shared/kernel/textbook"
	"time"
)

// frozenBookQualityReport combines immutable chapter detector output with the
// dated review ledger for this build. It never synthesizes editorial scores.
func frozenBookQualityReport(build TextbookBookBuildExport, now time.Time) (json.RawMessage, error) {
	snapshot, err := shared.ReadBookQualitySnapshot(build.ManifestJSON)
	if err != nil {
		return nil, err
	}
	status := "available"
	var assessment *shared.BookQualityAssessment
	if snapshot == nil {
		status = "unavailable"
	} else {
		a := snapshot.Assess()
		assessment = &a
	}
	return json.Marshal(struct {
		Format                string                              `json:"format"`
		BuildID               string                              `json:"build_id"`
		ManifestHash          string                              `json:"manifest_hash"`
		ExportedAt            time.Time                           `json:"exported_at"`
		Scope                 string                              `json:"scope"`
		ChapterSnapshotStatus string                              `json:"chapter_snapshot_status"`
		ChapterSnapshot       *shared.BookQualitySnapshot         `json:"chapter_snapshot"`
		ChapterAssessment     *shared.BookQualityAssessment       `json:"chapter_assessment"`
		AutomatedChecks       []shared.AutomatedPublicationCheck  `json:"automated_checks"`
		HumanReviews          []shared.HumanPublicationReview     `json:"human_reviews"`
		DelegatedReviews      []shared.DelegatedPublicationReview `json:"delegated_reviews"`
		PublicationPreflight  shared.PublicationPreflightResult   `json:"publication_preflight"`
	}{
		Format: "inkwords.book-quality-report.v1", BuildID: build.BuildID.String(), ManifestHash: build.ManifestHash, ExportedAt: now,
		Scope:                 "章节自动质量报告冻结在构建中；审阅记录与出版预检为导出时当前构建的记录。自动门禁通过不代表整书审阅通过；委托 AI 审阅不是真人同行评审、独立冷读试学或出版认证。缺失阶段不得推定通过。",
		ChapterSnapshotStatus: status, ChapterSnapshot: snapshot, ChapterAssessment: assessment,
		AutomatedChecks: build.AutomatedChecks, HumanReviews: build.HumanReviews, DelegatedReviews: build.DelegatedReviews, PublicationPreflight: build.Preflight,
	})
}
