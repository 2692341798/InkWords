package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ParseOfficialWebHTML turns one already-policy-approved HTML page into
// citeable, Markdown-shaped text. It is a parser only: links and visible text
// remain untrusted source material and never become application instructions.
func (p *StructuredParser) ParseOfficialWebHTML(src io.Reader, request StructuredParseRequest) (StructuredDocument, error) {
	if err := request.Validate(); err != nil {
		return StructuredDocument{}, err
	}
	content, err := io.ReadAll(io.LimitReader(src, maxStructuredTextBytes+1))
	if err != nil {
		return StructuredDocument{}, fmt.Errorf("read official web source: %w", err)
	}
	if int64(len(content)) > maxStructuredTextBytes {
		return StructuredDocument{}, fmt.Errorf("官网页面超过可安全解析大小限制；请缩小抓取范围后重试")
	}
	markdown := officialWebMarkdownWithTabs(content, request.PreserveOfficialWebTabs)
	if request.LegacyOfficialWeb {
		markdown = legacyOfficialWebMarkdown(content)
	}
	if strings.TrimSpace(markdown) == "" {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析官网页面：没有可引用的可见文本")
	}
	parsed, err := p.parseMarkdown(markdown, request)
	if err != nil {
		return StructuredDocument{}, err
	}
	parsed.Document.MediaType = "text/html"
	parsed.Document.ArtifactPath = ""
	for index := range parsed.Chunks {
		parsed.Chunks[index].Locator.Path = ""
		parsed.Chunks[index].Locator.URL = request.CanonicalLocator
	}
	return parsed, nil
}

func legacyOfficialWebMarkdown(content []byte) string {
	tokenizer := html.NewTokenizer(bytes.NewReader(content))
	var output strings.Builder
	var captureTag string
	var captureLevel int
	var text strings.Builder
	ignoredDepth := 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		switch tokenType {
		case html.StartTagToken:
			tagBytes, _ := tokenizer.TagName()
			tag := strings.ToLower(string(tagBytes))
			if ignoredDepth > 0 {
				ignoredDepth++
				continue
			}
			if ignoredHTMLTag(tag) {
				ignoredDepth = 1
				continue
			}
			if captureTag == "" && captureHTMLTag(tag) {
				captureTag, captureLevel = tag, headingLevel(tag)
				text.Reset()
			}
		case html.TextToken:
			if ignoredDepth == 0 && captureTag != "" {
				if value := strings.TrimSpace(html.UnescapeString(string(tokenizer.Text()))); value != "" {
					if text.Len() > 0 {
						text.WriteByte(' ')
					}
					text.WriteString(value)
				}
			}
		case html.EndTagToken:
			tagBytes, _ := tokenizer.TagName()
			tag := strings.ToLower(string(tagBytes))
			if ignoredDepth > 0 {
				ignoredDepth--
				continue
			}
			if tag == captureTag {
				if value := strings.TrimSpace(text.String()); value != "" {
					if captureLevel > 0 {
						output.WriteString(strings.Repeat("#", captureLevel))
						output.WriteByte(' ')
					}
					output.WriteString(value)
					output.WriteString("\n\n")
				}
				captureTag, captureLevel = "", 0
				text.Reset()
			}
		}
	}
	return strings.TrimSpace(output.String())
}

func ignoredHTMLTag(tag string) bool {
	switch tag {
	case "script", "style", "noscript", "svg", "canvas", "template", "iframe", "object", "embed":
		return true
	default:
		return false
	}
}

func captureHTMLTag(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6", "p", "li", "pre", "blockquote", "td", "th":
		return true
	default:
		return false
	}
}

func headingLevel(tag string) int {
	if len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6' {
		return int(tag[1] - '0')
	}
	return 0
}
