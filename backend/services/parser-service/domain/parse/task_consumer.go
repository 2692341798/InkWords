package parse

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	crawldomain "inkwords-backend/services/parser-service/domain/crawl"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	parserinfra "inkwords-backend/shared/platform/parser"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
	"inkwords-backend/shared/platform/sourceartifact"
)

type parseTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type textbookWorkspaceTaskStore interface {
	TextbookWorkspaceMatches(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

type parseExecutor interface {
	Parse(src io.Reader, filename string) (ParseResult, error)
}

type structuredParseExecutor interface {
	ParseStructured(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error)
	ParseStructuredArchive(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredArchiveResult, error)
	ParseOfficialWebHTML(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error)
}

type officialWebCrawlExecutor interface {
	CrawlCaptured(context.Context, crawldomain.Policy) (crawldomain.Manifest, []crawldomain.CapturedPage, error)
}

type sourceArtifactOpener interface {
	Open(string) (io.ReadCloser, error)
}

// deliveryAcknowledger 将 RabbitMQ amqp.Delivery 的 Ack/Nack 抽象为接口，使无 broker 环境也可测试。
type deliveryAcknowledger interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
}

// TaskConsumer converts RabbitMQ parse tasks into service-owned parse executions.
type TaskConsumer struct {
	tasks              parseTaskService
	parser             parseExecutor
	artifacts          sourceArtifactOpener
	officialWebCrawler officialWebCrawlExecutor
}

// WithOfficialWebCrawler supplies the bounded crawler only to the worker path;
// it is not exposed to generic parsing or browser-controlled code.
func (consumer *TaskConsumer) WithOfficialWebCrawler(crawler officialWebCrawlExecutor) *TaskConsumer {
	consumer.officialWebCrawler = crawler
	return consumer
}

// NewTaskConsumer wires the parser-service parse worker with task persistence and parse execution.
func NewTaskConsumer(tasks parseTaskService, parser parseExecutor, artifacts ...sourceArtifactOpener) *TaskConsumer {
	var artifactStore sourceArtifactOpener
	if len(artifacts) > 0 {
		artifactStore = artifacts[0]
	}
	return &TaskConsumer{
		tasks: tasks, parser: parser, artifacts: artifactStore,
	}
}

type parsePayload struct {
	Filename string
	Content  []byte
}

// StartParseConsumer boots the parser-service RabbitMQ worker for parse.requested messages.
func StartParseConsumer(ctx context.Context, taskService parseTaskService, parseService *Service, artifacts ...sourceArtifactOpener) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, parse consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_PARSE_QUEUE", "inkwords.parse")
	routingKey := sharedmq.ParseRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	if err := channel.QueueBind(queue.Name, routingKey, exchangeName, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	deliveries, err := channel.Consume(queue.Name, "parser-service-parse-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	checkpointRoot := envOrDefault("TEXTBOOK_CRAWL_CHECKPOINTS_DIR", "/app/crawl-checkpoints")
	checkpointStore, err := crawldomain.NewFilesystemCheckpointStore(checkpointRoot)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, fmt.Errorf("initialize official web crawl checkpoints: %w", err)
	}
	consumer := NewTaskConsumer(taskService, parseService, artifacts...).WithOfficialWebCrawler(crawldomain.NewService(crawldomain.NewDefaultHTTPFetcher(), checkpointStore))
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				if err := consumer.ConsumeMessage(ctx, delivery.Body, delivery); err != nil {
					log.Printf("consume parse message failed: %v", err)
				}
			}
		}
	}()

	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
}

// ConsumeMessage 消费一条原始消息体，负责反序列化、业务处理、以及投递确认（Ack/Nack）。
// 将消息生命周期收敛到单一可测试方法中，避免确认逻辑分散在 goroutine 循环里。
func (c *TaskConsumer) ConsumeMessage(ctx context.Context, body []byte, ack deliveryAcknowledger) error {
	var message sharedmq.ParseRequestedMessage
	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("invalid parse message payload: %v", err)
		if ackErr := ack.Ack(false); ackErr != nil {
			return fmt.Errorf("ack malformed parse message: %w", ackErr)
		}
		return nil
	}

	if err := c.HandleParseRequested(ctx, message); err != nil {
		log.Printf("parse task handling failed for %s: %v", message.TaskID, err)
		if nackErr := ack.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack for parse task %s: %w (work: %w)", message.TaskID, nackErr, err)
		}
		return nil
	}

	if err := ack.Ack(false); err != nil {
		return fmt.Errorf("ack for parse task %s: %w", message.TaskID, err)
	}
	return nil
}

// HandleParseRequested consumes one parse.requested message and persists the parse result to the task store.
func (c *TaskConsumer) HandleParseRequested(ctx context.Context, message sharedmq.ParseRequestedMessage) error {
	if c == nil || c.tasks == nil || c.parser == nil {
		return errors.New("parse task consumer dependencies are not configured")
	}
	if isTextbookParseKind(message.Kind) {
		valid, err := c.validateTextbookWorkspace(ctx, message.TaskID, message.WorkspaceID, message.Kind)
		if err != nil || !valid {
			return err
		}
	}

	cancelled, err := c.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}
	if message.Kind == sharedtextbook.TextbookSourceImportTaskSubtype {
		return c.handleTextbookSourceImport(ctx, message)
	}
	if message.Kind == sharedtextbook.TextbookOfficialWebImportTaskSubtype {
		return c.handleOfficialWebImport(ctx, message)
	}

	payload, err := decodeParsePayload(message.Payload)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid parse payload")
	}

	result, err := c.parser.Parse(bytes.NewReader(payload.Content), payload.Filename)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	resultJSON, err := buildParseResultJSON(result)
	if err != nil {
		return fmt.Errorf("marshal parse result failed: %w", err)
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, resultJSON)
}

func isTextbookParseKind(kind string) bool {
	return kind == sharedtextbook.TextbookSourceImportTaskSubtype || kind == sharedtextbook.TextbookOfficialWebImportTaskSubtype
}

func (c *TaskConsumer) validateTextbookWorkspace(ctx context.Context, taskID uuid.UUID, workspaceID *uuid.UUID, taskSubtype string) (bool, error) {
	if workspaceID == nil || *workspaceID == uuid.Nil {
		return false, c.tasks.MarkFailed(ctx, taskID, "textbook task workspace identity is missing")
	}
	store, ok := c.tasks.(textbookWorkspaceTaskStore)
	if !ok {
		return false, errors.New("textbook task workspace store is not configured")
	}
	matches, err := store.TextbookWorkspaceMatches(ctx, taskID, *workspaceID, taskSubtype)
	if err != nil {
		return false, err
	}
	if !matches {
		return false, c.tasks.MarkFailed(ctx, taskID, "textbook task workspace identity does not match")
	}
	return true, nil
}

func (c *TaskConsumer) handleOfficialWebImport(ctx context.Context, message sharedmq.ParseRequestedMessage) error {
	structured, ok := c.parser.(structuredParseExecutor)
	if !ok || c.officialWebCrawler == nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web import is not configured")
	}
	var payload sharedtextbook.OfficialWebImportTaskPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil || payload.Validate() != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid official web import payload")
	}
	policy, err := crawldomain.DefaultPolicy(payload.EntryURL)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web import policy is invalid")
	}
	policy.AllowedPathPrefixes = append([]string(nil), payload.AllowedPathPrefixes...)
	if err := policy.Validate(); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web import policy is invalid")
	}
	manifest, pages, err := c.officialWebCrawler.CrawlCaptured(ctx, policy)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web crawl failed: "+err.Error())
	}
	if manifest.Status != "complete" || len(pages) == 0 || manifest.SnapshotInputHash == "" {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web crawl did not complete within its approved boundary")
	}
	result := sharedtextbook.OfficialWebImportTaskResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash, ContentHash: manifest.SnapshotInputHash, ResolvedVersion: manifest.SnapshotInputHash, CapturedAt: time.Now().UTC()}
	if payload.TaskVersion == 2 {
		if manifest.SnapshotInputHash != crawldomain.SnapshotInputHash(manifest) {
			return c.tasks.MarkFailed(ctx, message.TaskID, "official web manifest hash is invalid")
		}
		result.ResultVersion = 2
		result.ParserVersion = payload.ParserVersion
		result.ManifestHash = manifest.SnapshotInputHash
		result.ContentHash = sharedtextbook.OfficialWebParsedContentHash(result.ManifestHash, result.ParserVersion)
		result.ResolvedVersion = result.ContentHash
		result.CrawlManifest, err = json.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("marshal official web crawl audit: %w", err)
		}
	}
	for _, page := range pages {
		if !strings.Contains(strings.ToLower(page.Manifest.ContentType), "html") {
			return c.tasks.MarkFailed(ctx, message.TaskID, "official web crawl contains an unsupported non-HTML page")
		}
		parsed, err := structured.ParseOfficialWebHTML(bytes.NewReader(page.Body), parserinfra.StructuredParseRequest{SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, CanonicalLocator: page.Manifest.CanonicalURL, Filename: "official-page.md", LegacyOfficialWeb: payload.TaskVersion == 1, PreserveOfficialWebTabs: payload.ParserVersion == sharedtextbook.OfficialWebParserVersion})
		if err != nil {
			return c.tasks.MarkFailed(ctx, message.TaskID, "official web page cannot be structured: "+err.Error())
		}
		result.Documents = append(result.Documents, parsed.Document)
		result.Chunks = append(result.Chunks, parsed.Chunks...)
	}
	if err := result.ValidateAgainst(payload); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "official web crawler returned invalid structured source result: "+err.Error())
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal official web import result: %w", err)
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, resultJSON)
}

func (c *TaskConsumer) handleTextbookSourceImport(ctx context.Context, message sharedmq.ParseRequestedMessage) error {
	structured, ok := c.parser.(structuredParseExecutor)
	if !ok {
		return c.tasks.MarkFailed(ctx, message.TaskID, "structured textbook parsing is not configured")
	}
	var payload sharedtextbook.SourceImportTaskPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil || payload.Validate() != nil || c.artifacts == nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid textbook source import payload")
	}
	artifact, err := c.artifacts.Open(payload.ArtifactToken)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "textbook source artifact is unavailable")
	}
	defer artifact.Close()
	sourceFile, err := materializeTextbookArtifact(artifact, payload.ByteSize, payload.ContentHash)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "textbook source artifact does not match frozen import")
	}
	defer func() {
		_ = sourceFile.Close()
		_ = os.Remove(sourceFile.Name())
	}()
	request := parserinfra.StructuredParseRequest{SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, CanonicalLocator: payload.Locator, ArtifactPath: payload.Filename, Filename: payload.Filename, LegacyCodeParagraphs: payload.TaskVersion == 2}
	result := sharedtextbook.SourceImportTaskResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash}
	if strings.EqualFold(filepath.Ext(payload.Filename), ".zip") {
		archive, err := structured.ParseStructuredArchive(sourceFile, request)
		if err != nil {
			return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
		}
		for _, document := range archive.Documents {
			result.Documents = append(result.Documents, document.Document)
			result.Chunks = append(result.Chunks, document.Chunks...)
		}
	} else {
		document, err := structured.ParseStructured(sourceFile, request)
		if err != nil {
			return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
		}
		result.Documents = []sharedtextbook.SourceDocument{document.Document}
		result.Chunks = document.Chunks
	}
	if err := result.ValidateAgainst(payload); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "parser returned invalid structured source result")
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal textbook source import result: %w", err)
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, resultJSON)
}

// materializeTextbookArtifact verifies the immutable source as it streams to a
// temporary file. The parser then receives a seekable reader without forcing an
// otherwise valid 888 MiB local source into the worker heap.
func materializeTextbookArtifact(source io.Reader, expectedSize int64, expectedHash string) (*os.File, error) {
	file, err := os.CreateTemp("", "inkwords-textbook-source-*")
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(source, sourceartifact.MaxTextbookSourceBytes+1))
	if err != nil {
		return fail(err)
	}
	if size != expectedSize || size > sourceartifact.MaxTextbookSourceBytes {
		return fail(errors.New("source artifact size mismatch"))
	}
	actualHash := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fail(errors.New("source artifact hash mismatch"))
	}
	if err := file.Sync(); err != nil {
		return fail(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	return file, nil
}

func decodeParsePayload(raw []byte) (parsePayload, error) {
	var payload struct {
		Filename      string `json:"filename"`
		ContentBase64 string `json:"content_base64"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return parsePayload{}, errors.New("invalid parse payload")
	}
	if strings.TrimSpace(payload.Filename) == "" {
		return parsePayload{}, errors.New("invalid parse payload")
	}

	content, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
	if err != nil {
		return parsePayload{}, errors.New("invalid parse payload")
	}
	return parsePayload{
		Filename: strings.TrimSpace(payload.Filename),
		Content:  content,
	}, nil
}

func buildParseResultJSON(result ParseResult) ([]byte, error) {
	payload := map[string]any{
		"source_content": result.SourceContent,
	}
	if result.ArchiveSummary != nil {
		payload["archive_summary"] = result.ArchiveSummary
	}
	return json.Marshal(payload)
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
