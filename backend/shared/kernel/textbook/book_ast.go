// Package textbook contains stable, service-neutral contracts for the
// textbook authoring pipeline.
package textbook

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const CanonicalBookASTFormat = "inkwords.book-ast.v2"

// CanonicalBookAST is the single frozen manuscript representation from which
// publication formats must be rendered. It keeps the source Markdown intact
// for editable exports while giving every chapter and asset a stable ID.
// Rendering-specific concerns (DOCX reference styles, PDF pagination) do not
// belong in this model.
type CanonicalBookAST struct {
	Format   string                 `json:"format"`
	Title    string                 `json:"title"`
	BuiltAt  time.Time              `json:"built_at"`
	Chapters []CanonicalBookChapter `json:"chapters"`
	Notices  []PublicationNotice    `json:"publication_notices,omitempty"`
}

type CanonicalBookChapter struct {
	Citations   []CanonicalBookCitation `json:"citations,omitempty"`
	ID          string                  `json:"id"`
	Order       int                     `json:"order"`
	Title       string                  `json:"title"`
	Markdown    string                  `json:"markdown"`
	ContentHash string                  `json:"content_hash"`
	Assets      []CanonicalBookAsset    `json:"assets"`
}

type CanonicalBookAsset struct {
	ID          string `json:"id"`
	StableRef   string `json:"stable_ref"`
	ContentHash string `json:"content_hash"`
	AltText     string `json:"alt_text"`
}

func (document CanonicalBookAST) Validate() error {
	if (document.Format != CanonicalBookASTFormat && document.Format != "inkwords.book-ast.v1" && document.Format != CanonicalBookASTWithNoticesFormat) || strings.TrimSpace(document.Title) == "" || document.BuiltAt.IsZero() || len(document.Chapters) == 0 {
		return fmt.Errorf("canonical book AST is incomplete")
	}
	previousOrder := 0
	seen := make(map[string]struct{}, len(document.Chapters))
	for _, chapter := range document.Chapters {
		if strings.TrimSpace(chapter.ID) == "" || chapter.Order <= previousOrder || strings.TrimSpace(chapter.Title) == "" || strings.TrimSpace(chapter.Markdown) == "" || !isSHA256Digest(chapter.ContentHash) {
			return fmt.Errorf("canonical book chapter is incomplete")
		}
		if _, exists := seen[chapter.ID]; exists {
			return fmt.Errorf("canonical book chapter IDs must be unique")
		}
		seen[chapter.ID] = struct{}{}
		previousOrder = chapter.Order
		if err := chapter.validateCitations(); err != nil {
			return err
		}
		for _, asset := range chapter.Assets {
			if strings.TrimSpace(asset.ID) == "" || strings.TrimSpace(asset.StableRef) == "" || !isSHA256Digest(asset.ContentHash) || strings.TrimSpace(asset.AltText) == "" {
				return fmt.Errorf("canonical book asset is incomplete")
			}
		}
	}
	if len(document.Notices) > 0 && document.Format != CanonicalBookASTWithNoticesFormat {
		return fmt.Errorf("publication notices require canonical book AST v3")
	}
	if len(document.Notices) > 16 {
		return fmt.Errorf("publication notice count exceeds budget")
	}
	seenNotices, noticeBytes := map[string]bool{}, 0
	for _, notice := range document.Notices {
		if err := notice.Validate(); err != nil {
			return err
		}
		if seenNotices[notice.ID] {
			return fmt.Errorf("duplicate publication notice")
		}
		seenNotices[notice.ID] = true
		noticeBytes += len(notice.Text)
	}
	if noticeBytes > 128000 {
		return fmt.Errorf("publication notice text exceeds book budget")
	}
	return nil
}

// NewCanonicalBookAST validates and sorts a frozen chapter collection once,
// so every renderer observes the same order instead of applying its own sort.
func NewCanonicalBookAST(title string, builtAt time.Time, chapters []CanonicalBookChapter) (CanonicalBookAST, error) {
	document := CanonicalBookAST{Format: CanonicalBookASTFormat, Title: strings.TrimSpace(title), BuiltAt: builtAt.UTC(), Chapters: append([]CanonicalBookChapter(nil), chapters...)}
	sort.Slice(document.Chapters, func(left, right int) bool { return document.Chapters[left].Order < document.Chapters[right].Order })
	if err := document.Validate(); err != nil {
		return CanonicalBookAST{}, err
	}
	return document, nil
}
