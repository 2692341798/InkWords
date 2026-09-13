package parse

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	parserinfra "inkwords-backend/shared/platform/parser"
)

// ParseStructured preserves source locations for textbook evidence rather than returning
// only a merged text string. Persistence is intentionally separate from parsing so an
// incomplete import can be reviewed before it becomes a source snapshot.
func (s *Service) ParseStructured(file io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
	return s.docParser.ParseStructured(file, request)
}

// ParseStructuredArchive keeps every safe ZIP entry independent so core-api can
// validate and persist a batch without losing per-file provenance.
func (s *Service) ParseStructuredArchive(file io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredArchiveResult, error) {
	if !strings.EqualFold(filepath.Ext(request.Filename), ".zip") {
		return parserinfra.StructuredArchiveResult{}, fmt.Errorf("structured archive parsing requires a ZIP input")
	}
	return s.archiveParser.ParseStructuredArchive(file, request)
}

// ParseOfficialWebHTML preserves URL and heading provenance for a policy-approved
// HTML page. Remote fetching remains outside this parser boundary.
func (s *Service) ParseOfficialWebHTML(file io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
	return s.docParser.ParseOfficialWebHTML(file, request)
}
