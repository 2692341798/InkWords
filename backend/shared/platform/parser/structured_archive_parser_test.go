package parser

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveParserParseStructuredArchiveKeepsIndependentDocumentsAndLocations(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	markdown, err := writer.Create("docs/intro.md")
	require.NoError(t, err)
	_, err = markdown.Write([]byte("# 入门\n\n从这里开始。\n"))
	require.NoError(t, err)
	code, err := writer.Create("cmd/main.go")
	require.NoError(t, err)
	_, err = code.Write([]byte("package main\n\nfunc main() {}\n"))
	require.NoError(t, err)
	pdf, err := writer.Create("docs/guide.pdf")
	require.NoError(t, err)
	_, err = pdf.Write(minimalTextPDF("PDF archive evidence"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	result, err := NewArchiveParser(NewDocParser()).ParseStructuredArchive(bytes.NewReader(archive.Bytes()), StructuredParseRequest{
		SourceID: "source-zip", SnapshotID: "snapshot-zip", CanonicalLocator: "file:///course.zip", Filename: "course.zip",
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.ArchiveSummary.KeptFiles)
	require.Len(t, result.Documents, 3)
	require.Equal(t, "cmd/main.go", result.Documents[0].Document.ArtifactPath)
	require.Equal(t, "text/x-go", result.Documents[0].Document.MediaType)
	require.Equal(t, "go", result.Documents[0].Chunks[0].CodeLanguage)
	require.Equal(t, "file:///course.zip#cmd/main.go", result.Documents[0].Document.CanonicalLocator)
	require.Equal(t, "docs/guide.pdf", result.Documents[1].Document.ArtifactPath)
	require.Equal(t, 1, result.Documents[1].Chunks[0].Locator.Page)
	require.Equal(t, "docs/intro.md", result.Documents[2].Document.ArtifactPath)
	require.Equal(t, []string{"入门"}, result.Documents[2].Chunks[0].HeadingPath)
	for _, document := range result.Documents {
		require.NoError(t, document.Document.Validate())
		for _, chunk := range document.Chunks {
			require.NoError(t, chunk.Validate())
		}
	}
}

func TestArchiveParserParseStructuredArchiveReportsPDFAsNonCiteable(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	pdf, err := writer.Create("slides.pdf")
	require.NoError(t, err)
	_, err = pdf.Write([]byte("not-a-real-pdf"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, err = NewArchiveParser(NewDocParser()).ParseStructuredArchive(bytes.NewReader(archive.Bytes()), StructuredParseRequest{
		SourceID: "source-zip", SnapshotID: "snapshot-zip", CanonicalLocator: "file:///course.zip", Filename: "course.zip",
	})
	require.ErrorContains(t, err, "没有可可靠引用")
}
