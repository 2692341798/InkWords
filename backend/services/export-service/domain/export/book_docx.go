package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const bookDOCXMediaType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// BookDOCXProjection is a generated, editable projection of a frozen
// CanonicalBookAST. It retains the renderer version for BookBuild provenance.
type BookDOCXProjection struct {
	Content      []byte
	MediaType    string
	ToolVersions map[string]string
}

// PandocBookRenderer renders the canonical Markdown projection with an
// approved local reference document. It has no repository access: callers
// must supply the frozen AST owned by core-api.
type PandocBookRenderer struct {
	executable        string
	referenceDocument string
	images            BookImageSource
}

// WithBookImages supplies the trusted content-addressed reader for frozen images.
func (renderer *PandocBookRenderer) WithBookImages(source BookImageSource) *PandocBookRenderer {
	renderer.images = source
	return renderer
}

func NewPandocBookRenderer(executable, referenceDocument string) *PandocBookRenderer {
	return &PandocBookRenderer{
		executable:        strings.TrimSpace(executable),
		referenceDocument: strings.TrimSpace(referenceDocument),
	}
}

//nolint:gosec // Executable and reference document are trusted deployment configuration, never manuscript data.
func (renderer *PandocBookRenderer) RenderDOCX(ctx context.Context, book sharedtextbook.CanonicalBookAST) (BookDOCXProjection, error) {
	if renderer == nil || renderer.executable == "" {
		return BookDOCXProjection{}, fmt.Errorf("Pandoc renderer is not configured")
	}
	if err := validateReferenceDocument(renderer.referenceDocument); err != nil {
		return BookDOCXProjection{}, err
	}
	markdown, err := RenderBookMarkdown(book, renderer.images)
	if err != nil {
		return BookDOCXProjection{}, err
	}
	temporaryDirectory, err := os.MkdirTemp("", "inkwords-book-docx-*")
	if err != nil {
		return BookDOCXProjection{}, fmt.Errorf("create DOCX render directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporaryDirectory) }()
	// Freeze the exact reference bytes used by this invocation so its provenance
	// cannot describe a different file after a deployment changes the template.
	referenceBytes, err := os.ReadFile(renderer.referenceDocument)
	if err != nil {
		return BookDOCXProjection{}, fmt.Errorf("read Pandoc reference document: %w", err)
	}
	referencePath := filepath.Join(temporaryDirectory, "reference.docx")
	if err := os.WriteFile(referencePath, referenceBytes, 0600); err != nil {
		return BookDOCXProjection{}, fmt.Errorf("freeze Pandoc reference document: %w", err)
	}
	outputPath := filepath.Join(temporaryDirectory, "book.docx")
	filterPath := filepath.Join(temporaryDirectory, "layout.lua")
	if err := os.WriteFile(filterPath, []byte(bookDOCXLayoutFilter), 0600); err != nil {
		return BookDOCXProjection{}, fmt.Errorf("write DOCX layout filter: %w", err)
	}
	command := exec.CommandContext(ctx, renderer.executable,
		"--from=markdown",
		"--to=docx",
		"--reference-doc="+referencePath,
		"--lua-filter="+filterPath,
		"--output="+outputPath,
	)
	command.Stdin = bytes.NewReader(markdown)
	if _, err := command.CombinedOutput(); err != nil {
		return BookDOCXProjection{}, fmt.Errorf("render DOCX with Pandoc: %w", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		return BookDOCXProjection{}, fmt.Errorf("read Pandoc DOCX output: %w", err)
	}
	if !bytes.HasPrefix(content, []byte("PK\x03\x04")) {
		return BookDOCXProjection{}, fmt.Errorf("Pandoc DOCX output is not an OOXML document")
	}
	version, err := pandocVersion(ctx, renderer.executable)
	if err != nil {
		return BookDOCXProjection{}, err
	}
	return BookDOCXProjection{Content: content, MediaType: bookDOCXMediaType, ToolVersions: map[string]string{
		"pandoc": version, "docx_layout": "inkwords.docx-layout.v2",
		"docx_reference_sha256": fmt.Sprintf("sha256:%x", sha256.Sum256(referenceBytes)),
		"book_images":           "inkwords.frozen-book-images.v1",
	}}, nil
}

func validateReferenceDocument(path string) error {
	if path == "" {
		return fmt.Errorf("Pandoc reference document is not configured")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat Pandoc reference document: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Pandoc reference document must be a regular file")
	}
	return nil
}

//nolint:gosec // The executable is trusted deployment configuration, never manuscript data.
func pandocVersion(ctx context.Context, executable string) (string, error) {
	output, err := exec.CommandContext(ctx, executable, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("read Pandoc version: %w", err)
	}
	line, _, _ := strings.Cut(string(output), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("Pandoc version output is empty")
	}
	return line, nil
}
