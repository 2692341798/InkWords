package service

import (
	"fmt"
	"inkwords-backend/internal/parser"
)

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
