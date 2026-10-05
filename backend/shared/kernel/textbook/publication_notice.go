package textbook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const PublicationNoticeFormat = "inkwords.publication-notice.v1"
const CanonicalBookASTWithNoticesFormat = "inkwords.book-ast.v3"

// PublicationNoticeDraft is user-supplied distribution text, not a rights
// approval. SourceURL is an attribution link and must never trigger fetching.
type PublicationNoticeDraft struct {
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	SourceURL   string   `json:"source_url"`
	PreparedBy  string   `json:"prepared_by"`
	SubjectRefs []string `json:"subject_refs"`
}

// PublicationNotice freezes one distribution statement with exact text and
// stable identity. Its subjects must additionally belong to the frozen build.
type PublicationNotice struct {
	PublicationNoticeDraft
	Format      string `json:"format"`
	ID          string `json:"id"`
	ContentHash string `json:"content_hash"`
}

func noticeDigest(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }

func (d PublicationNoticeDraft) validate() error {
	u, err := url.Parse(d.SourceURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(d.SourceURL) > 2048 || strings.ContainsAny(d.SourceURL, "<>\" \t\r\n") {
		return fmt.Errorf("publication notice source must be an HTTPS attribution link")
	}
	if strings.TrimSpace(d.Title) == "" || len(d.Title) > 160 || strings.ContainsAny(d.Title, "\r\n\x00") || strings.TrimSpace(d.PreparedBy) == "" || len(d.PreparedBy) > 200 || strings.ContainsAny(d.PreparedBy, "\r\n\x00") || strings.TrimSpace(d.Text) == "" || len(d.Text) > 24000 || !utf8.ValidString(d.Text) || strings.ContainsRune(d.Text, 0) || len(d.SubjectRefs) == 0 || len(d.SubjectRefs) > 100 {
		return fmt.Errorf("publication notice is incomplete or exceeds budget")
	}
	seen := map[string]bool{}
	for _, subject := range d.SubjectRefs {
		kind, id, ok := strings.Cut(subject, ":")
		parsed, err := uuid.Parse(id)
		if !ok || (kind != "chapter-revision" && kind != "code-artifact") || err != nil || parsed == uuid.Nil || parsed.String() != id || seen[subject] {
			return fmt.Errorf("invalid publication notice subject")
		}
		seen[subject] = true
	}
	return nil
}

// FreezePublicationNotices computes immutable IDs without changing license text.
// Stable ordering gives retries the same book-build input hash.
func FreezePublicationNotices(drafts []PublicationNoticeDraft) ([]PublicationNotice, error) {
	if len(drafts) > 16 {
		return nil, fmt.Errorf("publication notice count exceeds budget")
	}
	var notices []PublicationNotice
	seen := map[string]bool{}
	total := 0
	for _, d := range drafts {
		if err := d.validate(); err != nil {
			return nil, err
		}
		total += len(d.Text)
		if total > 128000 {
			return nil, fmt.Errorf("publication notice text exceeds book budget")
		}
		d.SubjectRefs = append([]string(nil), d.SubjectRefs...)
		sort.Strings(d.SubjectRefs)
		raw, _ := json.Marshal(d)
		id := "notice-" + strings.TrimPrefix(noticeDigest(raw), "sha256:")
		if seen[id] {
			return nil, fmt.Errorf("duplicate publication notice")
		}
		seen[id] = true
		notices = append(notices, PublicationNotice{PublicationNoticeDraft: d, Format: PublicationNoticeFormat, ID: id, ContentHash: noticeDigest([]byte(d.Text))})
	}
	sort.Slice(notices, func(i, j int) bool { return notices[i].ID < notices[j].ID })
	return notices, nil
}

// Validate checks the frozen statement identity against its exact text and subjects.
func (n PublicationNotice) Validate() error {
	expected, err := FreezePublicationNotices([]PublicationNoticeDraft{n.PublicationNoticeDraft})
	if err != nil {
		return err
	}
	if n.Format != PublicationNoticeFormat || n.ID != expected[0].ID || n.ContentHash != expected[0].ContentHash {
		return fmt.Errorf("publication notice identity or text hash differs")
	}
	return nil
}
