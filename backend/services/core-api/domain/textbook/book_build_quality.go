package textbook

import (
	"encoding/json"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"time"
)

func freezeBookQuality(chapters []bookBuildChapterSource, bookContract, styleSheet string, at time.Time) (sharedtextbook.BookQualitySnapshot, error) {
	snapshot := sharedtextbook.BookQualitySnapshot{Format: sharedtextbook.BookQualitySnapshotFormat, CapturedAt: at, BookContractRevisionID: bookContract, StyleSheetRevisionID: styleSheet, Chapters: make([]sharedtextbook.FrozenChapterQuality, 0, len(chapters))}
	for _, chapter := range chapters {
		report := append(json.RawMessage(nil), chapter.QualityReportJSON...)
		if len(report) == 0 {
			report = json.RawMessage(`null`)
		}
		hash, err := sharedtextbook.BookQualityReportHash(report)
		if err != nil {
			return snapshot, err
		}
		snapshot.Chapters = append(snapshot.Chapters, sharedtextbook.FrozenChapterQuality{ChapterID: chapter.ChapterID.String(), RevisionID: chapter.RevisionID.String(), ContentHash: canonicalContractHash(chapter.ContentHash), BookContractRevisionID: chapter.BookContractRevisionID.String(), StyleSheetRevisionID: chapter.StyleSheetRevisionID.String(), ReportHash: hash, Report: report})
	}
	return snapshot, snapshot.Validate()
}
