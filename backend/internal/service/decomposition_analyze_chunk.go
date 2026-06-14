package service

import (
	"fmt"
	"inkwords-backend/shared/platform/parser"
)

const (
	fileAnalyzeDirectRuneLimit = 1000000
	fileAnalyzeChunkRuneSize   = 120000
)

// Why: 电子书/课件这类长文的章节粒度比源码模块更细，切块过大时后半部分章节会在局部摘要里被稀释。
// 这里对超长文件使用更小的块尺寸，优先保住篇章覆盖率，而不是追求单块越大越省请求。
func resolveFileAnalyzeChunkSize(totalRunes int) int {
	if totalRunes <= fileAnalyzeDirectRuneLimit {
		return 0
	}
	return fileAnalyzeChunkRuneSize
}

func chunkFileContent(content string, targetChunkSize int) []parser.FileChunk {
	var chunks []parser.FileChunk
	runes := []rune(content)
	totalLen := len(runes)

	if totalLen <= targetChunkSize {
		chunks = append(chunks, parser.FileChunk{Dir: "全文", Content: content})
		return chunks
	}

	start := 0
	part := 1
	for start < totalLen {
		end := start + targetChunkSize
		if end >= totalLen {
			end = totalLen
			chunks = append(chunks, parser.FileChunk{Dir: fmt.Sprintf("第 %d 部分", part), Content: string(runes[start:end])})
			break
		}

		splitIdx := end
		for i := end; i > start && i > end-2000; i-- {
			if runes[i] == '\n' {
				splitIdx = i + 1
				break
			}
		}

		chunks = append(chunks, parser.FileChunk{Dir: fmt.Sprintf("第 %d 部分", part), Content: string(runes[start:splitIdx])})
		start = splitIdx
		part++
	}

	return chunks
}
