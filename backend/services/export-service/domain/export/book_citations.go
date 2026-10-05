package export

import (
	"fmt"
	"net/url"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func renderChapterCitations(chapter sharedtextbook.CanonicalBookChapter, chapterNumber int) string {
	markers := sharedtextbook.BookEvidenceMarkers(chapter.Markdown)
	if len(markers) == 0 {
		return chapter.Markdown
	}
	byID := make(map[string]sharedtextbook.CanonicalBookCitation, len(chapter.Citations))
	for _, citation := range chapter.Citations {
		byID[citation.ID] = citation
	}
	numbers, ordered := map[string]int{}, []string{}
	var body strings.Builder
	cursor := 0
	for _, marker := range markers {
		number, exists := numbers[marker.ID]
		if !exists {
			ordered = append(ordered, marker.ID)
			number = len(ordered)
			numbers[marker.ID] = number
		}
		body.WriteString(chapter.Markdown[cursor:marker.Start])
		fmt.Fprintf(&body, "[^c%d-s%d]", chapterNumber, number)
		cursor = marker.End
	}
	body.WriteString(chapter.Markdown[cursor:])
	for index, id := range ordered {
		citation := byID[id]
		locator := citation.Evidence.Locator
		label := locator.Path
		if locator.Symbol != "" {
			label += " · " + locator.Symbol
		}
		if label == "" {
			label = citation.Snapshot.Locator
		}
		fmt.Fprintf(&body, "\n\n[^c%d-s%d]: %s", chapterNumber, index+1, escapeCitationText(label))
		if locator.StartLine > 0 {
			fmt.Fprintf(&body, "，第 %d–%d 行", locator.StartLine, locator.EndLine)
		}
		if citation.Snapshot.ResolvedVersion != "" {
			fmt.Fprintf(&body, "。固定版本 %s", escapeCitationText(citation.Snapshot.ResolvedVersion))
		}
		if sourceURL := citationSourceURL(citation); sourceURL != "" {
			fmt.Fprintf(&body, "。[查看来源](<%s>)。", sourceURL)
		} else {
			fmt.Fprintf(&body, "。来源快照 %s（%s）。", escapeCitationText(citation.Snapshot.ID), escapeCitationText(citation.Snapshot.ContentHash))
		}
	}
	return body.String()
}

func escapeCitationText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ").Replace(value)
}

func citationSourceURL(citation sharedtextbook.CanonicalBookCitation) string {
	u, err := url.Parse(citation.Snapshot.Locator)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return ""
	}
	if u.Hostname() == "github.com" && citation.Snapshot.Kind == sharedtextbook.SourceKindGitRepository && citation.Evidence.Locator.Path != "" {
		u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git") + "/blob/" + citation.Snapshot.ResolvedVersion + "/" + strings.TrimLeft(citation.Evidence.Locator.Path, "/")
		u.RawPath, u.RawQuery, u.Fragment = "", "", ""
		if citation.Evidence.Locator.StartLine > 0 {
			u.Fragment = fmt.Sprintf("L%d-L%d", citation.Evidence.Locator.StartLine, citation.Evidence.Locator.EndLine)
		}
	}
	return strings.NewReplacer("<", "%3C", ">", "%3E").Replace(u.String())
}
