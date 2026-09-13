package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"inkwords-backend/shared/platform/sourceartifact"
)

const (
	maxDOCXCompressedBytes  int64 = sourceartifact.MaxTextbookSourceBytes
	maxDOCXDocumentXMLBytes       = 16 << 20
)

// parseDocx extracts paragraphs in document order from word/document.xml.
// DOCX is a ZIP container, so its XML is read under explicit budgets rather than by
// stripping tags from library output; this preserves line breaks, lists, tables and styles.
func (p *DocParser) parseDocx(file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("读取 DOCX 元数据失败: %w", err)
	}
	if info.Size() > maxDOCXCompressedBytes {
		return "", fmt.Errorf("DOCX 文件超过最大压缩大小限制")
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return "", fmt.Errorf("打开 DOCX 容器失败: %w", err)
	}
	for _, entry := range reader.File {
		if entry.Name != "word/document.xml" {
			continue
		}
		if entry.Flags&0x1 != 0 || entry.UncompressedSize64 > maxDOCXDocumentXMLBytes {
			return "", fmt.Errorf("DOCX 文档内容不安全或过大")
		}
		xmlBytes, err := readZIPEntry(entry, maxDOCXDocumentXMLBytes)
		if err != nil {
			return "", fmt.Errorf("读取 DOCX 文档内容失败: %w", err)
		}
		text, err := extractDOCXText(xmlBytes)
		if err != nil {
			return "", fmt.Errorf("解析 DOCX 文档内容失败: %w", err)
		}
		if strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("无法可靠解析 DOCX：没有可用文本；请确认文件未加密或重新导出")
		}
		return text, nil
	}
	return "", fmt.Errorf("DOCX 缺少 word/document.xml")
}

func readZIPEntry(entry *zip.File, limit int64) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("解压后内容超过限制")
	}
	return content, nil
}

type docxParagraph struct {
	text    strings.Builder
	style   string
	isList  bool
	inTable bool
}

func extractDOCXText(content []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	blocks := make([]string, 0)
	var paragraph *docxParagraph
	tableDepth := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "tbl":
				tableDepth++
			case "p":
				paragraph = &docxParagraph{inTable: tableDepth > 0}
			case "pStyle":
				if paragraph != nil {
					paragraph.style = xmlAttribute(value.Attr, "val")
				}
			case "numPr":
				if paragraph != nil {
					paragraph.isList = true
				}
			case "t", "delText", "instrText":
				if paragraph != nil {
					var text string
					if err := decoder.DecodeElement(&text, &value); err != nil {
						return "", err
					}
					paragraph.text.WriteString(text)
				}
			case "tab":
				if paragraph != nil {
					paragraph.text.WriteByte('\t')
				}
			case "br", "cr":
				if paragraph != nil {
					paragraph.text.WriteByte('\n')
				}
			case "tc":
				if paragraph != nil && paragraph.text.Len() > 0 {
					paragraph.text.WriteString(" | ")
				}
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "p":
				if paragraph != nil {
					if block := formatDOCXParagraph(*paragraph); block != "" {
						blocks = append(blocks, block)
					}
				}
				paragraph = nil
			case "tbl":
				if tableDepth > 0 {
					tableDepth--
				}
			}
		}
	}
	return strings.Join(blocks, "\n\n"), nil
}

func xmlAttribute(attributes []xml.Attr, name string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func formatDOCXParagraph(paragraph docxParagraph) string {
	text := strings.TrimSpace(paragraph.text.String())
	if text == "" {
		return ""
	}
	style := strings.ToLower(strings.ReplaceAll(paragraph.style, " ", ""))
	if level := docxHeadingLevel(style); level > 0 {
		return strings.Repeat("#", level) + " " + text
	}
	if paragraph.isList {
		return "- " + text
	}
	if strings.Contains(style, "code") || strings.Contains(style, "source") {
		return "```text\n" + text + "\n```"
	}
	return text
}

func docxHeadingLevel(style string) int {
	for level := 1; level <= 6; level++ {
		if style == fmt.Sprintf("heading%d", level) || style == fmt.Sprintf("title%d", level) {
			return level
		}
	}
	if style == "title" || style == "subtitle" {
		return 1
	}
	return 0
}
