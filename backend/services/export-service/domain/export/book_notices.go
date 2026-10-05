package export

import (
	"fmt"
	"sort"
	"strings"

	shared "inkwords-backend/shared/kernel/textbook"
)

func renderBookNotices(notices []shared.PublicationNotice) string {
	if len(notices) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n\n# 来源与分发声明\n\n以下声明随本冻结稿分发；声明文本不等于整书出版审校已通过。\n")
	for _, notice := range notices {
		// Keep the heading adjacent to the first literal paragraph in both
		// Pandoc and Chromium. Separate metadata paragraphs can strand a heading
		// when Word moves a kept-together code paragraph to the next page.
		fmt.Fprintf(&out, "\n## [%s](<%s>)\n\n", escapeNoticeInline(notice.Title), notice.SourceURL)
		// A literal fenced block preserves license text without interpreting
		// source-supplied Markdown/HTML as document structure or instructions.
		fence := "~~~"
		for strings.Contains(notice.Text, fence) {
			fence += "~"
		}
		fmt.Fprintf(&out, "%stext\n整理记录：%s\n来源：%s\n\n%s", fence, notice.PreparedBy, notice.SourceURL, notice.Text)
		if !strings.HasSuffix(notice.Text, "\n") {
			out.WriteByte('\n')
		}
		out.WriteString(fence + "\n")
	}
	return out.String()
}

func escapeNoticeInline(text string) string {
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ").Replace(text)
}

func bookNoticeFiles(book shared.CanonicalBookAST) []BookPackageFile {
	if len(book.Notices) == 0 {
		return nil
	}
	byArtifact := map[string][]shared.PublicationNotice{}
	for _, notice := range book.Notices {
		for _, subject := range notice.SubjectRefs {
			if artifact, ok := strings.CutPrefix(subject, "code-artifact:"); ok {
				byArtifact[artifact] = append(byArtifact[artifact], notice)
			}
		}
	}
	all := []BookPackageFile{{Path: "publication-notices.txt", Content: []byte(plainBookNotices(book.Notices))}}
	for artifact, notices := range byArtifact {
		all = append(all, BookPackageFile{Path: "code/" + artifact + "/THIRD-PARTY-NOTICES.txt", Content: []byte(plainBookNotices(notices))})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	return all
}

func plainBookNotices(notices []shared.PublicationNotice) string {
	var text strings.Builder
	for i, notice := range notices {
		if i > 0 {
			text.WriteString("\n\n")
		}
		fmt.Fprintf(&text, "%s\nSource: %s\nPrepared by: %s\n\n%s", notice.Title, notice.SourceURL, notice.PreparedBy, notice.Text)
	}
	return text.String()
}
