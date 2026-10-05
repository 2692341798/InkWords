package textbook

import (
	"strings"

	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

// Filter before LIMIT: a newly imported document must remain reachable even
// when older snapshots already contain more than the candidate budget.
func matchingRetrievalRows(query *gorm.DB, text string) *gorm.DB {
	terms := shared.RetrievalQueryTerms(text)
	if len(terms) == 0 {
		return query.Where("FALSE")
	}
	const predicate = "strpos(lower(concat_ws(' ', source_chunks.search_text, source_chunks.locator->>'symbol', source_chunks.heading_path::text, source_documents.artifact_path, source_documents.canonical_locator)), ?) > 0"
	predicates := make([]string, 0, len(terms))
	values := make([]any, 0, len(terms))
	for _, term := range terms {
		predicates = append(predicates, predicate)
		values = append(values, term)
	}
	return query.Where("("+strings.Join(predicates, " OR ")+")", values...)
}
