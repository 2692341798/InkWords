package export

import (
	"bytes"
	"fmt"
	xhtml "golang.org/x/net/html"
	"html"
	"io"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RenderBookHTML produces a printable projection exclusively from the frozen
// AST. Goldmark retains its default rejection of raw HTML and scripts.
func RenderBookHTML(book sharedtextbook.CanonicalBookAST, sources ...BookImageSource) ([]byte, error) {
	markdown, err := RenderBookMarkdown(book, sources...)
	if err != nil {
		return nil, err
	}
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(bookCodeRenderer{}, 100))))
	doc := md.Parser().Parse(text.NewReader(markdown))
	first := doc.FirstChild()
	if heading, ok := first.(*ast.Heading); !ok || heading.Level != 1 {
		return nil, fmt.Errorf("canonical book title is missing")
	}
	doc.RemoveChild(doc, first)
	var nav strings.Builder
	nav.WriteString("<nav aria-label=\"目录\"><h2>目录</h2><ol>")
	if err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := node.(*ast.Heading); ok && entering && h.Level <= 2 {
			id, exists := h.AttributeString("id")
			if !exists {
				return ast.WalkStop, fmt.Errorf("book heading has no anchor")
			}
			fmt.Fprintf(&nav, "<li class=\"toc-level-%d\"><a href=\"#%s\">%s</a></li>", h.Level, html.EscapeString(string(id.([]byte))), html.EscapeString(string(h.Text(markdown))))
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return nil, err
	}
	nav.WriteString("</ol></nav>")
	var body bytes.Buffer
	if err := md.Renderer().Render(&body, markdown, doc); err != nil {
		return nil, fmt.Errorf("render canonical book HTML: %w", err)
	}
	separated, err := separateBookFootnotes(body.Bytes())
	if err != nil {
		return nil, err
	}
	var output strings.Builder
	output.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src 'none'\"><title>")
	output.WriteString(html.EscapeString(book.Title))
	output.WriteString("</title><style>" + bookPrintCSS + "</style></head><body><header><h1>")
	output.WriteString(html.EscapeString(book.Title))
	output.WriteString("</h1></header>")
	output.WriteString(nav.String())
	output.WriteString("<main>")
	output.Write(separated)
	output.WriteString("</main></body></html>")
	return []byte(output.String()), nil
}

const bookPrintCSS = `
@page{size:A4;margin:18mm 16mm 20mm}
body{margin:0;font-family:"Noto Sans CJK SC","Noto Sans",sans-serif;color:#17212b;font-size:10.5pt;line-height:1.7}
header h1{font-size:24pt;line-height:1.35;margin:12mm 0 10mm}
nav{break-after:page}nav h2{font-size:16pt}nav ol{list-style:none;padding:0}nav li{margin:3mm 0;break-inside:avoid}nav .toc-level-2{padding-left:7mm}
a{color:#235782;text-decoration:none}main>h1{font-size:20pt;line-height:1.4;break-before:page;margin:0 0 8mm}h2{font-size:16pt;margin:7mm 0 3mm}h3{font-size:13pt;margin:5mm 0 2mm}h1,h2,h3,h4{break-after:avoid}
p{margin:0 0 3mm;orphans:3;widows:3}li{orphans:3;widows:3}li>p{margin-bottom:2mm}
img{display:block;max-width:100%;max-height:220mm;width:auto;height:auto;object-fit:contain;margin:4mm auto}p:has(>img){break-inside:avoid}
.book-image-caption{display:block;font-size:9pt;line-height:1.6;color:#45515e;margin:2mm 0 4mm}
p:has(+pre),p:has(+ul),p:has(+ol){break-after:avoid}
pre{white-space:pre-wrap;overflow-wrap:anywhere;tab-size:4;background:#f3f5f7;padding:3mm;margin:3mm 0;break-inside:avoid;line-height:1.5;font-size:9pt}
code{font-family:"Noto Sans Mono","DejaVu Sans Mono","FreeMono",monospace}p code,li code{font-size:9pt;overflow-wrap:anywhere}
table{border-collapse:collapse;width:100%;margin:4mm 0;table-layout:fixed}th,td{border:1px solid #cbd2d9;padding:2mm;text-align:left;vertical-align:top;overflow-wrap:anywhere}thead{display:table-header-group}tr{break-inside:avoid}th{background:#eef2f6}
.footnotes{break-before:page;font-size:9pt;overflow-wrap:anywhere}.footnotes::before{content:"来源注释";display:block;font-size:16pt;font-weight:bold;margin-bottom:5mm}.footnotes li{break-inside:avoid;margin-bottom:3mm}.footnotes p{font-size:9pt;line-height:1.6}sup{font-size:7pt}
`

type bookCodeRenderer struct{}

// CSS sibling selectors ignore intervening text. Token order preserves that
// distinction so prose-separated footnotes do not acquire stray punctuation.
func separateBookFootnotes(input []byte) ([]byte, error) {
	var output bytes.Buffer
	tokens := xhtml.NewTokenizer(bytes.NewReader(input))
	previousSup := false
	for {
		kind := tokens.Next()
		if kind == xhtml.ErrorToken {
			if errors.Is(tokens.Err(), io.EOF) {
				return output.Bytes(), nil
			}
			return nil, tokens.Err()
		}
		raw := append([]byte(nil), tokens.Raw()...)
		name, _ := tokens.TagName()
		if previousSup && kind == xhtml.StartTagToken && string(name) == "sup" {
			output.WriteString(`<sup class="footnote-separator">,</sup>`)
		}
		output.Write(raw)
		previousSup = kind == xhtml.EndTagToken && string(name) == "sup"
	}
}

func (bookCodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, renderBookCode)
	reg.Register(ast.KindCodeBlock, renderBookCode)
	reg.Register(ast.KindImage, renderBookImage)
}

func renderBookImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	img := node.(*ast.Image)
	alt := html.EscapeString(bookImageAlt(img, source))
	// RenderBookMarkdown already resolved only frozen raster assets. Keep the
	// explanatory caption visible in PDF as Pandoc does for DOCX figures.
	_, err := w.WriteString(`<img src="` + html.EscapeString(string(img.Destination)) + `" alt="` + alt + `"><span class="book-image-caption">` + alt + `</span>`)
	return ast.WalkSkipChildren, err
}

func renderBookCode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	var raw strings.Builder
	for i := 0; i < node.Lines().Len(); i++ {
		segment := node.Lines().At(i)
		raw.Write(segment.Value(source))
	}
	// Every unit retains original newlines, so concatenation recovers the code.
	lines := strings.SplitAfter(raw.String(), "\n")
	for start := 0; start < len(lines); {
		end := min(start+26, len(lines))
		if end < len(lines) {
			for i := end - 1; i >= start+9; i-- {
				if strings.TrimSpace(lines[i]) == "" {
					end = i + 1
					break
				}
			}
		}
		unit := strings.Join(lines[start:end], "")
		if unit != "" {
			if _, err := w.WriteString("<pre><code>" + html.EscapeString(unit) + "</code></pre>\n"); err != nil {
				return ast.WalkStop, err
			}
		}
		start = end
	}
	return ast.WalkSkipChildren, nil
}
