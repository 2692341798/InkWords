package export

import (
	"fmt"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RenderBookMarkdown is the editable projection of the canonical AST. DOCX,
// PDF and package builders must consume this same validated AST rather than
// re-querying chapter rows or independently sorting manuscript content.
func RenderBookMarkdown(book sharedtextbook.CanonicalBookAST, sources ...BookImageSource) ([]byte, error) {
	if err := book.Validate(); err != nil {
		return nil, fmt.Errorf("validate canonical book AST: %w", err)
	}
	images, err := newBookImageProjection(sources)
	if err != nil {
		return nil, err
	}
	var output strings.Builder
	output.WriteString("# ")
	output.WriteString(book.Title)
	output.WriteString("\n\n")
	for index, chapter := range book.Chapters {
		if index > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString("<!-- inkwords:chapter:")
		output.WriteString(chapter.ID)
		output.WriteString(" content-hash:")
		output.WriteString(chapter.ContentHash)
		output.WriteString(" -->\n")
		body, err := images.chapter(renderChapterCitations(chapter, index+1), chapter.Assets)
		if err != nil {
			return nil, err
		}
		output.WriteString(body)
	}
	output.WriteString(renderBookNotices(book.Notices))
	return []byte(output.String()), nil
}
