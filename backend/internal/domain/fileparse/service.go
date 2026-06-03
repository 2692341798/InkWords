package fileparse

import (
	"io"
	"path/filepath"
	"strings"

	"inkwords-backend/internal/infra/parser"
)

type ParseResult struct {
	SourceContent  string
	ArchiveSummary *parser.ArchiveSummary
}

type Service struct {
	docParser     *parser.DocParser
	archiveParser *parser.ArchiveParser
}

func NewService(docParser *parser.DocParser, archiveParser *parser.ArchiveParser) *Service {
	return &Service{
		docParser:     docParser,
		archiveParser: archiveParser,
	}
}

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
