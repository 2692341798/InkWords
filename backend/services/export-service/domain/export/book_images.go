package export

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	shared "inkwords-backend/shared/kernel/textbook"
)

// BookImageSource reads bytes by a frozen content hash, never a manuscript URL
// or host path. Renderers independently verify the returned bytes before use.
type BookImageSource interface {
	ReadBookImage(contentHash string) ([]byte, error)
}

const maxBookImageBytes = 10 << 20
const maxBookImagePixels = 40_000_000

type bookImageProjection struct {
	source   BookImageSource
	cache    map[string]string
	bytes    int
	required bool
}

func newBookImageProjection(sources []BookImageSource) (*bookImageProjection, error) {
	if len(sources) > 1 {
		return nil, fmt.Errorf("only one frozen image source is allowed")
	}
	p := &bookImageProjection{cache: map[string]string{}}
	if len(sources) == 1 {
		p.source = sources[0]
		p.required = true
	}
	return p, nil
}

func (p *bookImageProjection) destination(asset shared.CanonicalBookAsset) (string, error) {
	// Pure structural projections are used by font-surface preflight. Actual
	// download/render entrypoints supply a source and must resolve every image.
	if p.source == nil {
		if p.required {
			return "", fmt.Errorf("frozen image source is not configured")
		}
		return asset.StableRef, nil
	}
	if cached, ok := p.cache[asset.ContentHash]; ok {
		return cached, nil
	}
	b, err := p.source.ReadBookImage(asset.ContentHash)
	if err != nil {
		return "", fmt.Errorf("read frozen image %s: %w", asset.ID, err)
	}
	if len(b) == 0 || len(b) > maxBookImageBytes || p.bytes+len(b) > 100<<20 {
		return "", fmt.Errorf("frozen image byte budget exceeded")
	}
	if fmt.Sprintf("sha256:%x", sha256.Sum256(b)) != asset.ContentHash {
		return "", fmt.Errorf("frozen image content hash mismatch")
	}
	config, kind, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxBookImagePixels {
		return "", fmt.Errorf("frozen image dimensions or encoding are invalid")
	}
	if kind != "png" && kind != "jpeg" {
		return "", fmt.Errorf("unsupported frozen image format")
	}
	if _, _, err = image.Decode(bytes.NewReader(b)); err != nil {
		return "", fmt.Errorf("frozen image is truncated or invalid: %w", err)
	}
	uri := "data:image/" + kind + ";base64," + base64.StdEncoding.EncodeToString(b)
	p.cache[asset.ContentHash] = uri
	p.bytes += len(b)
	return uri, nil
}

type bookImageSpan struct {
	node       *ast.Image
	start, end int
}

// Capture spans from Goldmark's own link parser, which handles nesting,
// references and code spans. Do not search/replace URL text in manuscript code.
type bookImageSpanParser struct {
	parser.InlineParser
	spans []bookImageSpan
}

func (p *bookImageSpanParser) Parse(parent ast.Node, reader text.Reader, pc parser.Context) ast.Node {
	node := p.InlineParser.Parse(parent, reader, pc)
	if img, ok := node.(*ast.Image); ok {
		_, end := reader.Position()
		p.spans = append(p.spans, bookImageSpan{img, img.Pos(), end.Start})
	}
	return node
}

func (p *bookImageProjection) chapter(markdown string, assets []shared.CanonicalBookAsset) (string, error) {
	destinations := map[string]string{}
	for _, asset := range assets {
		if _, exists := destinations[asset.StableRef]; exists {
			return "", fmt.Errorf("duplicate frozen image reference")
		}
		uri, err := p.destination(asset)
		if err != nil {
			return "", err
		}
		destinations[asset.StableRef] = uri
	}
	spans := &bookImageSpanParser{InlineParser: parser.NewLinkParser()}
	inlines := parser.DefaultInlineParsers()
	for i := range inlines {
		if inlines[i].Value == parser.NewLinkParser() {
			inlines[i].Value = spans
		}
	}
	md := goldmark.New(goldmark.WithParser(parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(inlines...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)), goldmark.WithExtensions(extension.GFM, extension.Footnote))
	md.Parser().Parse(text.NewReader([]byte(markdown)))
	sort.Slice(spans.spans, func(i, j int) bool { return spans.spans[i].start < spans.spans[j].start })
	used := map[string]bool{}
	var out strings.Builder
	end := 0
	for _, span := range spans.spans {
		ref := string(span.node.Destination)
		uri, ok := destinations[ref]
		if !ok {
			return "", fmt.Errorf("manuscript image is not bound to a frozen asset")
		}
		if span.start < end || span.end > len(markdown) || span.end <= span.start {
			return "", fmt.Errorf("unsupported overlapping image source spans")
		}
		out.WriteString(markdown[end:span.start])
		out.WriteString(bookImageMarkdown(bookImageAlt(span.node, []byte(markdown)), uri))
		end = span.end
		used[ref] = true
	}
	out.WriteString(markdown[end:])
	for _, asset := range assets {
		// Keep identity even when an image already appears inline. The caption
		// and fallback placement belong to this projection, not a new revision.
		fmt.Fprintf(&out, "\n\n<!-- inkwords:asset:%s stable-ref:%s -->", safeAssetComment(asset.ID), safeAssetComment(asset.StableRef))
		if !used[asset.StableRef] {
			out.WriteString("\n\n" + bookImageMarkdown(asset.AltText, destinations[asset.StableRef]))
		}
	}
	return out.String(), nil
}

func bookImageMarkdown(alt, destination string) string {
	alt = strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ").Replace(alt)
	return "![" + alt + "](" + destination + ")"
}

func bookImageAlt(img *ast.Image, source []byte) string {
	return string(util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(img.Text(source)))))
}

func safeAssetComment(value string) string {
	return strings.NewReplacer("--", "—", "<", "", ">", "", "\n", " ", "\r", " ").Replace(value)
}

type bookPackageImageSource []BookPackageFile

func (s bookPackageImageSource) ReadBookImage(hash string) ([]byte, error) {
	for _, file := range s {
		for _, ext := range []string{".png", ".jpg", ".webp"} {
			if file.Path == strings.TrimPrefix(hash, "sha256:")+ext {
				return file.Content, nil
			}
		}
	}
	return nil, fmt.Errorf("frozen image is absent from package")
}
