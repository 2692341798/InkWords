package parse

import (
	"io"
	"path/filepath"
	"strings"

	parserinfra "inkwords-backend/shared/platform/parser"
)

// ParseResult is the normalized parser-service response for plain files and ZIP archives.
type ParseResult struct {
	SourceContent  string
	ArchiveSummary *parserinfra.ArchiveSummary
}

// Service owns parser-service document parsing orchestration.
type Service struct {
	docParser     *parserinfra.DocParser
	archiveParser *parserinfra.ArchiveParser
}

// NewService creates the service-owned parse orchestrator.
func NewService(docParser *parserinfra.DocParser, archiveParser *parserinfra.ArchiveParser) *Service {
	return &Service{
		docParser:     docParser,
		archiveParser: archiveParser,
	}
}

// Parse dispatches plain files and ZIP archives to the matching parser implementation.
func (s *Service) Parse(file io.Reader, filename string) (ParseResult, error) {
	if strings.EqualFold(filepath.Ext(filename), ".zip") {
		result, err := s.archiveParser.ParseArchive(file, filename)
		if err != nil {
			return ParseResult{}, err
		}
		return ParseResult{
			SourceContent:  result.SourceContent,
			ArchiveSummary: result.ArchiveSummary,
		}, nil
	}

	content, err := s.docParser.Parse(file, filename)
	if err != nil {
		return ParseResult{}, err
	}
	return ParseResult{SourceContent: content}, nil
}
