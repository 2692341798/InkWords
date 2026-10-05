package export

import (
	"encoding/json"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"time"
)

// frozenBookVerificationSummary preserves old-build absence and exports both
// historical and dated current assessments without querying mutable evidence.
func frozenBookVerificationSummary(build TextbookBookBuildExport, now time.Time) (json.RawMessage, error) {
	snapshot, err := sharedtextbook.ReadBookVerificationSnapshot(build.ManifestJSON)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return json.RawMessage(`{"status":"unavailable","reason":"冻结构建未保存整书级运行验证快照；请在审校中逐项核验。"}`), nil
	}
	return json.Marshal(struct {
		Format       string                                    `json:"format"`
		BuildID      string                                    `json:"build_id"`
		ManifestHash string                                    `json:"manifest_hash"`
		Scope        string                                    `json:"scope"`
		Snapshot     *sharedtextbook.BookVerificationSnapshot  `json:"snapshot"`
		AtFreeze     sharedtextbook.BookVerificationAssessment `json:"at_freeze"`
		AtExport     sharedtextbook.BookVerificationAssessment `json:"at_export"`
	}{"inkwords.book-verification-summary.v1", build.BuildID.String(), build.ManifestHash, "仅冻结工件清单声明的命令；不代表安装全流程、未声明的运行行为或学习者掌握。", snapshot, snapshot.Assess(snapshot.CapturedAt), snapshot.Assess(now)})
}
