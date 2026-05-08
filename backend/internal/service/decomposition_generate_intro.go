package service

import (
	"context"
	"encoding/json"
	"fmt"
	"inkwords-backend/internal/db"
	"inkwords-backend/internal/llm"
	"inkwords-backend/internal/model"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (s *DecompositionService) generateSeriesIntro(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, seriesTitle string, outline []Chapter, progressChan chan<- string, errChan chan<- error) {
	sendProgress := func(status string, content string, message string) {
		msg := map[string]interface{}{
			"status":       status,
			"chapter_sort": 0,
			"content":      content,
			"message":      message,
			"title":        "系列导读",
		}
		bytes, _ := json.Marshal(msg)
		progressChan <- string(bytes)
	}

	sendProgress("generating", "", "")

	var outlineStrBuilder strings.Builder
	for _, ch := range outline {
		outlineStrBuilder.WriteString(fmt.Sprintf("- %s: %s\n", ch.Title, ch.Summary))
	}

	prompt := fmt.Sprintf(`你是一个高级技术博客作者。请根据以下系列文章的大纲，编写一篇高质量的“系列导读”或“总结”文章（约500-800字）。
这篇文章将作为整个系列的入口，吸引读者阅读。
系列标题：%s
各章节大纲：
%s

要求：
1. 简明扼要地介绍这个系列将要解决的问题和核心价值。
2. 简述各个章节的精彩看点，引导读者循序渐进地阅读。
3. 结尾给出学习建议或寄语。
`, seriesTitle, outlineStrBuilder.String())

	messages := []llm.Message{
		{Role: "system", Content: "你是一个高级技术博客作者，擅长编写引人入胜的系列导读。"},
		{Role: "user", Content: prompt},
	}

	llmModel := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		llmModel = envModel
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	chunkChan := make(chan string, 100)
	internalErrChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(internalErrChan)

		tempChunkChan := make(chan string)
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			for chunk := range tempChunkChan {
				chunkChan <- chunk
			}
		}()

		_, err := s.llmClient.GenerateStream(streamCtx, llmModel, messages, tempChunkChan)
		wg.Wait()
		if err != nil {
			internalErrChan <- err
		}
	}()

	var contentBuilder strings.Builder
	idleTimeout := 60 * time.Second
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		case <-timer.C:
			streamCancel()
			errChan <- fmt.Errorf("intro generation idle timeout")
			return
		case err, ok := <-internalErrChan:
			if ok && err != nil {
				sendProgress("error", "", err.Error())
				db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]interface{}{
					"status": 2,
				})
				return
			}
		case chunk, ok := <-chunkChan:
			if !ok {
				finalContent := contentBuilder.String()
				db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]interface{}{
					"content": finalContent,
					"status":  1,
				})
				sendProgress("completed", "", "")
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)
			contentBuilder.WriteString(chunk)
			sendProgress("streaming", chunk, "")
		}
	}
}
