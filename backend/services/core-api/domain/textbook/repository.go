package textbook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var (
	ErrNotFound                = errors.New("textbook resource not found")
	ErrVersionConflict         = errors.New("textbook version conflict")
	ErrPrimarySourceExists     = errors.New("primary source already exists")
	ErrRevisionLocked          = errors.New("chapter revision is locked")
	ErrInvalidState            = errors.New("invalid textbook state")
	ErrClaimCoverageIncomplete = errors.New("critical claim coverage is incomplete")
	// ErrEvidenceBudgetExceeded prevents an approved blueprint from accidentally
	// sending a project-sized corpus to a model. The author must narrow the
	// explicit evidence mapping; the worker must never silently truncate it.
	ErrEvidenceBudgetExceeded = errors.New("generation evidence budget exceeded")
)

type Repository interface {
	CreateProject(context.Context, CreateProjectInput) (*Project, error)
	ListProjects(context.Context, uuid.UUID) ([]Project, error)
	GetProject(context.Context, uuid.UUID, uuid.UUID) (*Project, error)
	GetProjectWorkspace(context.Context, uuid.UUID, uuid.UUID) (*ProjectWorkspace, error)
	GetProjectProgress(context.Context, uuid.UUID, uuid.UUID) (*ProjectProgress, error)
	ListSourceLibrary(context.Context, uuid.UUID, uuid.UUID) ([]SourceLibraryDocument, error)
	ListSourceEvidence(context.Context, uuid.UUID, uuid.UUID, ...string) ([]SourceLibraryEvidence, error)
	RetrieveSourceEvidence(context.Context, uuid.UUID, RetrieveSourceInput) (*SourceRetrievalPlan, error)
	GetChapterWorkspace(context.Context, uuid.UUID, uuid.UUID) (*ChapterWorkspace, error)
	CreateBookBuild(context.Context, uuid.UUID, CreateBookBuildInput) (*BookBuildRow, error)
	GetEditorialWorkspace(context.Context, uuid.UUID, uuid.UUID) (*EditorialWorkspace, error)
	AddRightsItem(context.Context, uuid.UUID, AddRightsItemInput) (*RightsItemRow, error)
	AppendRightsAmendment(context.Context, uuid.UUID, AppendRightsAmendmentInput) (*sharedtextbook.RightsAmendment, error)
	CompletePublicationReview(context.Context, uuid.UUID, CompletePublicationReviewInput) (*PublicationReviewRow, error)
	RecordDelegatedPublicationReview(context.Context, uuid.UUID, RecordDelegatedPublicationReviewInput) (*sharedtextbook.DelegatedPublicationReview, error)
	PromoteBookBuild(context.Context, uuid.UUID, uuid.UUID) (*BookBuildRow, error)
	AddSource(context.Context, uuid.UUID, uuid.UUID, CreateSourceInput) (*Source, error)
	CreateChapter(context.Context, uuid.UUID, CreateChapterInput) (*Chapter, error)
	CreateBookContract(context.Context, uuid.UUID, CreateBookContractInput) (*BookContractRevision, error)
	CreateStyleSheet(context.Context, uuid.UUID, CreateStyleSheetInput) (*StyleSheetRevision, error)
	CreateBlueprint(context.Context, uuid.UUID, CreateBlueprintInput) (*BlueprintRevision, error)
	ApproveBookContract(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*BookContractRevision, error)
	ApproveStyleSheet(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*StyleSheetRevision, error)
	ApproveBlueprint(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*BlueprintRevision, error)
	PrepareSampleGeneration(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.SampleGenerationTarget) (sharedtextbook.SampleGenerationTaskPayload, error)
	LoadGinFixture(context.Context, uuid.UUID, uuid.UUID) (*SourceSnapshot, error)
	PrepareSourceImport(context.Context, uuid.UUID, PrepareSourceImportInput) (sharedtextbook.SourceImportTaskPayload, error)
	PrepareOfficialWebImport(context.Context, uuid.UUID, PrepareOfficialWebImportInput) (sharedtextbook.OfficialWebImportTaskPayload, error)
	PersistTextbookSourceImportResult(context.Context, uuid.UUID, map[string]any) error
	PersistTextbookSampleResult(context.Context, uuid.UUID, map[string]any) error
	GetGeneratedRevisionContext(context.Context, uuid.UUID) (*GeneratedRevisionContext, error)
	RegisterGeneratedCodeArtifact(context.Context, uuid.UUID, RegisterGeneratedCodeArtifactInput) (*CodeArtifactRow, error)
	RegisterManuscriptAsset(context.Context, uuid.UUID, RegisterManuscriptAssetInput) (*ManuscriptAssetRow, error)
	AppendRevision(context.Context, uuid.UUID, AppendRevisionInput) (*ChapterRevision, error)
	AcquireLock(context.Context, uuid.UUID, LockInput) (*ChapterLock, error)
	ApplyCandidate(context.Context, uuid.UUID, ApplyCandidateInput) (*ChapterRevision, error)
	RejectCandidate(context.Context, uuid.UUID, RejectCandidateInput) (*CandidateReview, error)
	PersistDocuments(context.Context, uuid.UUID, PersistDocumentsInput) error
	SoftDeleteProject(context.Context, uuid.UUID, uuid.UUID) error
}

const ginFixtureCommit = "73726dc606796a025971fe451f0aa6f1b9b847f6"

type ginFixtureExcerpt struct {
	suffix string
	symbol string
	line   int
	end    int
	text   string
}

type ginFixtureDocument struct {
	path            string
	title           string
	scope           string
	contentIdentity string
	excerpts        []ginFixtureExcerpt
}

func ginFixtureDocuments() []ginFixtureDocument {
	return []ginFixtureDocument{
		{
			path:            "routergroup.go",
			title:           "Gin RouterGroup route registration",
			scope:           "routergroup.go",
			contentIdentity: "Gin v1.12.0 route registration excerpts: RouterGroup.GET, RouterGroup.handle, Engine.addRoute",
			excerpts: []ginFixtureExcerpt{
				{suffix: "get", symbol: "(*RouterGroup).GET", line: 115, end: 117, text: "return group.handle(http.MethodGet, relativePath, handlers)"},
				{suffix: "handle", symbol: "(*RouterGroup).handle", line: 86, end: 88, text: "handlers = group.combineHandlers(handlers)"},
				{suffix: "add-route", symbol: "(*Engine).addRoute", line: 364, end: 366, text: "root.addRoute(path, handlers)"},
			},
		},
		{
			path:            "gin.go",
			title:           "Gin request dispatch",
			scope:           "gin.go",
			contentIdentity: "Gin v1.12.0 request dispatch excerpts: Engine.ServeHTTP, Engine.handleHTTPRequest",
			excerpts: []ginFixtureExcerpt{
				{suffix: "serve-http", symbol: "(*Engine).ServeHTTP", line: 661, end: 675, text: "engine.handleHTTPRequest(c)"},
				{suffix: "handle-http-request", symbol: "(*Engine).handleHTTPRequest", line: 690, end: 724, text: "按请求方法选择路由树，调用 root.getValue(rPath, ...)，把匹配 handlers 交给 Context.Next"},
			},
		},
		{
			path:            "tree.go",
			title:           "Gin route tree lookup",
			scope:           "tree.go",
			contentIdentity: "Gin v1.12.0 route tree lookup excerpt: node.getValue",
			excerpts: []ginFixtureExcerpt{
				{suffix: "node-get-value", symbol: "(*node).getValue", line: 418, end: 452, text: "getValue 比较当前节点前缀并沿匹配的子节点继续查找"},
			},
		},
	}
}

// LoadGinFixture installs the reviewable offline excerpts used by the first vertical slice.
// It is intentionally narrow: generic repository and document-stack importing belongs to parser-service.
func (r *GormRepository) LoadGinFixture(ctx context.Context, workspaceID, projectID uuid.UUID) (*SourceSnapshot, error) {
	var snapshot *SourceSnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockWorkspaceProject(tx, workspaceID, projectID)
		if err != nil {
			return err
		}
		if project.PrimarySourceID == nil {
			return ErrInvalidState
		}
		var source Source
		if err := tx.Where("id = ? AND project_id = ?", *project.PrimarySourceID, project.ID).First(&source).Error; err != nil {
			return ErrInvalidState
		}
		if source.Kind != sharedtextbook.SourceKindGitRepository || strings.TrimSuffix(source.Locator, "/") != "https://github.com/gin-gonic/gin" {
			return ErrInvalidState
		}
		for index, fixture := range ginFixtureDocuments() {
			created, err := ensureGinFixtureDocument(tx, source.ID, fixture)
			if err != nil {
				return err
			}
			if index == 0 {
				snapshot = created
			}
		}
		return nil
	})
	return snapshot, err
}

func ensureGinFixtureDocument(tx *gorm.DB, sourceID uuid.UUID, fixture ginFixtureDocument) (*SourceSnapshot, error) {
	contentHash := strings.TrimPrefix(digestJSON(fixture.contentIdentity), "sha256:")
	var existing SourceSnapshot
	err := tx.Where("source_id = ? AND resolved_version = ? AND content_hash = ?", sourceID, ginFixtureCommit, contentHash).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	limits, err := json.Marshal(map[string]any{"mode": "offline_fixture", "scope": fixture.scope, "capture": "bounded_fixed_commit_excerpts"})
	if err != nil {
		return nil, err
	}
	created := &SourceSnapshot{SourceID: sourceID, ResolvedVersion: ginFixtureCommit, ContentHash: contentHash, CapturedAt: time.Now().UTC(), Status: "captured", LimitsJSON: datatypes.JSON(limits)}
	if err := tx.Create(created).Error; err != nil {
		return nil, err
	}
	documentID := "gin-" + strings.TrimSuffix(fixture.path, path.Ext(fixture.path)) + "-" + created.ID.String()
	document := ParsedDocument{ID: documentID, SnapshotID: created.ID, CanonicalLocator: fixture.path, Title: fixture.title, MediaType: "text/x-go", ContentHash: digestJSON(fixture.contentIdentity), ArtifactPath: fixture.path}
	if err := tx.Create(&document).Error; err != nil {
		return nil, err
	}
	for ordinal, item := range fixture.excerpts {
		locator, err := marshalStructuredJSON(sharedtextbook.EvidenceLocator{Path: fixture.path, Symbol: item.symbol, StartLine: item.line, EndLine: item.end})
		if err != nil {
			return nil, err
		}
		row := ParsedChunk{ID: "gin-" + created.ID.String() + "-" + item.suffix, DocumentID: document.ID, Ordinal: ordinal + 1, HeadingPath: datatypes.JSON(`[]`), Locator: datatypes.JSON(locator), CodeLanguage: "go", StartByte: 0, EndByte: len(item.text), TextHash: digestJSON(item.text), SearchText: item.text}
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
	}
	return created, nil
}

// PrepareSourceImport validates the project-owned source and freezes the
// metadata that parser-service is allowed to use. A snapshot row is created
// only after core-api validates the worker result in one transaction.
func (r *GormRepository) PrepareSourceImport(ctx context.Context, workspaceID uuid.UUID, input PrepareSourceImportInput) (sharedtextbook.SourceImportTaskPayload, error) {
	project, err := r.GetProject(ctx, workspaceID, input.ProjectID)
	if err != nil {
		return sharedtextbook.SourceImportTaskPayload{}, err
	}
	var source Source
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", input.SourceID, project.ID).First(&source).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharedtextbook.SourceImportTaskPayload{}, ErrNotFound
		}
		return sharedtextbook.SourceImportTaskPayload{}, err
	}
	if source.Role != sharedtextbook.SourceRolePrimary && (!source.OfficialConfirmed || source.Role != sharedtextbook.SourceRoleOfficial) {
		return sharedtextbook.SourceImportTaskPayload{}, ErrInvalidState
	}
	payload := sharedtextbook.SourceImportTaskPayload{
		TaskVersion: sharedtextbook.SourceImportTaskVersion, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype,
		ProjectID: project.ID.String(), SourceID: source.ID.String(), SnapshotID: input.SnapshotID.String(),
		SourceKind: source.Kind, SourceRole: source.Role, Locator: source.Locator, Filename: strings.TrimSpace(input.Filename),
		ContentHash: input.ContentHash, ByteSize: input.ByteSize,
	}
	if source.Kind == sharedtextbook.SourceKindGitRepository {
		payload.ResolvedVersion = strings.TrimSpace(input.ResolvedVersion)
		if len(payload.ResolvedVersion) != 40 {
			return sharedtextbook.SourceImportTaskPayload{}, ErrInvalidState
		}
		if _, err := hex.DecodeString(payload.ResolvedVersion); err != nil {
			return sharedtextbook.SourceImportTaskPayload{}, ErrInvalidState
		}
	} else {
		payload.ResolvedVersion = input.ContentHash
	}
	return payload, nil
}

// PrepareOfficialWebImport freezes an already-confirmed official source and
// only permits the caller to narrow the source's entry-path boundary.
func (r *GormRepository) PrepareOfficialWebImport(ctx context.Context, workspaceID uuid.UUID, input PrepareOfficialWebImportInput) (sharedtextbook.OfficialWebImportTaskPayload, error) {
	project, err := r.GetProject(ctx, workspaceID, input.ProjectID)
	if err != nil {
		return sharedtextbook.OfficialWebImportTaskPayload{}, err
	}
	var source Source
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", input.SourceID, project.ID).First(&source).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharedtextbook.OfficialWebImportTaskPayload{}, ErrNotFound
		}
		return sharedtextbook.OfficialWebImportTaskPayload{}, err
	}
	if source.Kind != sharedtextbook.SourceKindOfficialWeb || source.Role != sharedtextbook.SourceRoleOfficial || !source.OfficialConfirmed {
		return sharedtextbook.OfficialWebImportTaskPayload{}, ErrInvalidState
	}
	prefixes := make([]string, 0, len(input.AllowedPathPrefixes))
	for _, prefix := range input.AllowedPathPrefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			prefixes = append(prefixes, prefix)
		}
	}
	payload := sharedtextbook.OfficialWebImportTaskPayload{
		TaskVersion: 2, ParserVersion: sharedtextbook.OfficialWebParserVersion, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype,
		ProjectID: project.ID.String(), SourceID: source.ID.String(), SnapshotID: input.SnapshotID.String(),
		SourceKind: source.Kind, SourceRole: source.Role, EntryURL: source.Locator, AllowedPathPrefixes: prefixes,
	}
	payload.InputHash = sharedtextbook.OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes, payload.ParserVersion)
	if err := payload.Validate(); err != nil {
		return sharedtextbook.OfficialWebImportTaskPayload{}, ErrInvalidState
	}
	return payload, nil
}

// PersistTextbookSourceImportResult reloads the frozen task payload and makes
// imported evidence authoritative only after its full provenance validates.
func (r *GormRepository) PersistTextbookSourceImportResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	var taskSubtype struct {
		TaskSubtype string `gorm:"column:task_subtype"`
	}
	if err := r.db.WithContext(ctx).Table("job_tasks").Select("task_subtype").Where("id = ?", taskID).First(&taskSubtype).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if taskSubtype.TaskSubtype == sharedtextbook.TextbookOfficialWebImportTaskSubtype {
		return r.persistOfficialWebImportResult(ctx, taskID, result)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal source import result: %w", err)
	}
	var imported sharedtextbook.SourceImportTaskResult
	if err := json.Unmarshal(resultJSON, &imported); err != nil {
		return fmt.Errorf("decode source import result: %w", err)
	}
	var task struct {
		TaskSubtype string         `gorm:"column:task_subtype"`
		PayloadJSON datatypes.JSON `gorm:"column:payload_json"`
	}
	if err := r.db.WithContext(ctx).Table("job_tasks").Select("task_subtype, payload_json").Where("id = ?", taskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if task.TaskSubtype != sharedtextbook.TextbookSourceImportTaskSubtype {
		return ErrInvalidState
	}
	var payload sharedtextbook.SourceImportTaskPayload
	if err := json.Unmarshal(task.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode frozen source import task: %w", err)
	}
	if err := imported.ValidateAgainst(payload); err != nil {
		return fmt.Errorf("validate source import result: %w", err)
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return ErrInvalidState
	}
	sourceID, err := uuid.Parse(payload.SourceID)
	if err != nil {
		return ErrInvalidState
	}
	snapshotID, err := uuid.Parse(payload.SnapshotID)
	if err != nil {
		return ErrInvalidState
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", projectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var source Source
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND project_id = ?", sourceID, project.ID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if source.Kind != payload.SourceKind || source.Role != payload.SourceRole || source.Locator != payload.Locator || (source.Role == sharedtextbook.SourceRoleOfficial && !source.OfficialConfirmed) {
			return ErrInvalidState
		}
		// A Git source may contribute several files from one immutable commit.
		// The file bytes are independently content-addressed, so commit alone is
		// not an idempotency key for SourceSnapshot persistence.
		contentHash := strings.TrimPrefix(payload.ContentHash, "sha256:")
		existing, err := lookupImportedSnapshot(tx, source.ID, payload)
		if err == nil {
			if existing.Status != "captured" || existing.ContentHash != contentHash || existing.ResolvedVersion != payload.ResolvedVersion {
				return ErrInvalidState
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		limits, err := marshalStructuredJSON(map[string]any{"task_id": taskID.String(), "parser_version": payload.TaskVersion, "filename": payload.Filename, "byte_size": payload.ByteSize, "artifact_token": payload.ArtifactToken, "input_hash": payload.InputHash})
		if err != nil {
			return err
		}
		snapshot := SourceSnapshot{ID: snapshotID, SourceID: source.ID, ResolvedVersion: payload.ResolvedVersion, ContentHash: contentHash, CapturedAt: time.Now().UTC(), Status: "captured", LimitsJSON: datatypes.JSON(limits)}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		return persistDocumentsTx(tx, snapshot.ID, imported.Documents, imported.Chunks)
	})
}

// persistOfficialWebImportResult accepts only complete worker evidence whose
// documents still fall inside the exact URL boundary frozen by core-api.
func (r *GormRepository) persistOfficialWebImportResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal official web import result: %w", err)
	}
	var imported sharedtextbook.OfficialWebImportTaskResult
	if err := json.Unmarshal(resultJSON, &imported); err != nil {
		return fmt.Errorf("decode official web import result: %w", err)
	}
	var task struct {
		TaskSubtype string         `gorm:"column:task_subtype"`
		PayloadJSON datatypes.JSON `gorm:"column:payload_json"`
	}
	if err := r.db.WithContext(ctx).Table("job_tasks").Select("task_subtype, payload_json").Where("id = ?", taskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if task.TaskSubtype != sharedtextbook.TextbookOfficialWebImportTaskSubtype {
		return ErrInvalidState
	}
	var payload sharedtextbook.OfficialWebImportTaskPayload
	if err := json.Unmarshal(task.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode frozen official web import task: %w", err)
	}
	if err := imported.ValidateAgainst(payload); err != nil {
		return fmt.Errorf("validate official web import result: %w", err)
	}
	for _, document := range imported.Documents {
		if !officialDocumentWithinBoundary(document.CanonicalLocator, payload) {
			return ErrInvalidState
		}
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return ErrInvalidState
	}
	sourceID, err := uuid.Parse(payload.SourceID)
	if err != nil {
		return ErrInvalidState
	}
	snapshotID, err := uuid.Parse(payload.SnapshotID)
	if err != nil {
		return ErrInvalidState
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", projectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var source Source
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND project_id = ?", sourceID, project.ID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if source.Kind != sharedtextbook.SourceKindOfficialWeb || source.Role != sharedtextbook.SourceRoleOfficial || !source.OfficialConfirmed || source.Locator != payload.EntryURL {
			return ErrInvalidState
		}
		contentHash := strings.TrimPrefix(imported.ContentHash, "sha256:")
		var existing SourceSnapshot
		err = tx.Where("source_id = ? AND resolved_version = ? AND content_hash = ?", source.ID, imported.ResolvedVersion, contentHash).First(&existing).Error
		if err == nil {
			if existing.Status != "captured" {
				return ErrInvalidState
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		limitsData := map[string]any{"task_id": taskID.String(), "entry_url": payload.EntryURL, "allowed_path_prefixes": payload.AllowedPathPrefixes, "input_hash": payload.InputHash, "manifest_hash": imported.ContentHash}
		if imported.ResultVersion == 2 {
			limitsData["manifest_hash"] = imported.ManifestHash
			limitsData["parser_version"] = imported.ParserVersion
			limitsData["crawl_manifest"] = imported.CrawlManifest
		}
		limits, err := marshalStructuredJSON(limitsData)
		if err != nil {
			return err
		}
		snapshot := SourceSnapshot{ID: snapshotID, SourceID: source.ID, ResolvedVersion: imported.ResolvedVersion, ContentHash: contentHash, CapturedAt: imported.CapturedAt.UTC(), Status: "captured", LimitsJSON: datatypes.JSON(limits)}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		return persistDocumentsTx(tx, snapshot.ID, imported.Documents, imported.Chunks)
	})
}

func officialDocumentWithinBoundary(raw string, payload sharedtextbook.OfficialWebImportTaskPayload) bool {
	document, err := url.Parse(raw)
	entry, entryErr := url.Parse(payload.EntryURL)
	if err != nil || entryErr != nil || document.Scheme != "https" || document.Hostname() != entry.Hostname() || document.User != nil {
		return false
	}
	for _, prefix := range payload.AllowedPathPrefixes {
		clean := path.Clean("/" + strings.TrimPrefix(prefix, "/"))
		if clean == "/" || document.Path == clean || strings.HasPrefix(document.Path, clean+"/") {
			return true
		}
	}
	return false
}

// PersistTextbookSampleResult materializes a worker result only after reloading and validating
// the immutable task payload stored by core-api. The worker never writes textbook facts directly.
func (r *GormRepository) PersistTextbookSampleResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal textbook sample result: %w", err)
	}
	var generated sharedtextbook.SampleGenerationTaskResult
	if err := json.Unmarshal(resultJSON, &generated); err != nil {
		return fmt.Errorf("decode textbook sample result: %w", err)
	}
	var task struct {
		TaskSubtype string         `gorm:"column:task_subtype"`
		PayloadJSON datatypes.JSON `gorm:"column:payload_json"`
	}
	if err := r.db.WithContext(ctx).Table("job_tasks").Select("task_subtype, payload_json").Where("id = ?", taskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if task.TaskSubtype != sharedtextbook.TextbookSampleGenerationTaskSubtype {
		return ErrInvalidState
	}
	var payload sharedtextbook.SampleGenerationTaskPayload
	if err := json.Unmarshal(task.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode frozen textbook task: %w", err)
	}
	if err := generated.ValidateAgainst(payload); err != nil {
		return fmt.Errorf("validate textbook generation result: %w", err)
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return ErrInvalidState
	}
	chapterID, err := uuid.Parse(payload.ChapterID)
	if err != nil {
		return ErrInvalidState
	}
	bookID, err := uuid.Parse(generated.BookContractRevision)
	if err != nil {
		return ErrInvalidState
	}
	styleID, err := uuid.Parse(generated.StyleSheetRevision)
	if err != nil {
		return ErrInvalidState
	}
	blueprintID, err := uuid.Parse(generated.BlueprintRevision)
	if err != nil {
		return ErrInvalidState
	}
	var project Project
	if err := r.db.WithContext(ctx).Where("id = ?", projectID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	var parentRevisionID *uuid.UUID
	if payload.ParentRevisionID != "" {
		parsed, err := uuid.Parse(payload.ParentRevisionID)
		if err != nil {
			return ErrInvalidState
		}
		parentRevisionID = &parsed
	}
	contentHash := strings.TrimPrefix(generated.ContentHash, "sha256:")
	if len(contentHash) != 64 {
		return ErrInvalidState
	}
	input := AppendRevisionInput{
		ChapterID: chapterID, ExpectedVersion: payload.ExpectedChapterVersion, Kind: sharedtextbook.RevisionKindCandidate,
		Markdown: generated.Markdown, DocumentJSON: generated.DocumentJSON, ContentHash: contentHash, CreatedBy: RevisionCreatorGeneration,
		ParentRevisionID: parentRevisionID, BookContractRevisionID: &bookID, StyleSheetRevisionID: &styleID, BlueprintRevisionID: &blueprintID,
		EvidencePackHash: generated.EvidencePackHash, PromptHash: generated.PromptHash, ProviderName: generated.ProviderName, ModelName: generated.ModelName,
		ProviderUsageJSON: generated.ProviderUsageJSON, QualityReportJSON: generated.QualityReportJSON, GenerationTaskID: &taskID,
	}
	if payload.Correction != nil {
		return r.persistCorrectionResult(ctx, project.WorkspaceID, taskID, payload, generated, input)
	}
	_, err = r.AppendRevision(ctx, project.WorkspaceID, input)
	return err
}

// GetGeneratedRevisionContext resolves only the revision created by the
// immutable task id. It is for trusted post-generation projections, not an
// author-selected revision lookup.
func (r *GormRepository) GetGeneratedRevisionContext(ctx context.Context, taskID uuid.UUID) (*GeneratedRevisionContext, error) {
	if taskID == uuid.Nil {
		return nil, ErrInvalidState
	}
	var row struct {
		ChapterRevision
		WorkspaceID      uuid.UUID `gorm:"column:workspace_id"`
		BookContractHash string    `gorm:"column:book_contract_hash"`
		StyleSheetHash   string    `gorm:"column:style_sheet_hash"`
	}
	if err := r.db.WithContext(ctx).Table("chapter_revisions").
		Select("chapter_revisions.*, textbook_projects.workspace_id, book_contract_revisions.content_hash AS book_contract_hash, style_sheet_revisions.content_hash AS style_sheet_hash").
		Joins("JOIN textbook_chapters ON textbook_chapters.id = chapter_revisions.chapter_id").
		Joins("JOIN textbook_projects ON textbook_projects.id = textbook_chapters.project_id").
		Joins("JOIN book_contract_revisions ON book_contract_revisions.id = chapter_revisions.book_contract_revision_id").
		Joins("JOIN style_sheet_revisions ON style_sheet_revisions.id = chapter_revisions.style_sheet_revision_id").
		Where("chapter_revisions.generation_task_id = ?", taskID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if row.CreatedBy != RevisionCreatorGeneration || row.Kind != sharedtextbook.RevisionKindCandidate || row.WorkspaceID == uuid.Nil || row.BookContractHash == "" || row.StyleSheetHash == "" {
		return nil, ErrInvalidState
	}
	return &GeneratedRevisionContext{Revision: row.ChapterRevision, WorkspaceID: row.WorkspaceID, BookContractHash: canonicalContractHash(row.BookContractHash), StyleSheetHash: canonicalContractHash(row.StyleSheetHash)}, nil
}

// ListSourceLibrary returns metadata only, preventing one project-open request from loading all excerpts.
func (r *GormRepository) ListSourceLibrary(ctx context.Context, workspaceID, projectID uuid.UUID) ([]SourceLibraryDocument, error) {
	if _, err := r.GetProject(ctx, workspaceID, projectID); err != nil {
		return nil, err
	}
	var rows []SourceLibraryDocument
	err := r.db.WithContext(ctx).Table("source_documents").
		Select("source_documents.id, source_snapshots.source_id, source_documents.snapshot_id, source_documents.canonical_locator, source_documents.title, source_documents.media_type, source_documents.content_hash, source_documents.artifact_path, COUNT(source_chunks.id) AS chunk_count").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("LEFT JOIN source_chunks ON source_chunks.document_id = source_documents.id").
		Where("source_snapshots.source_id IN (SELECT id FROM textbook_sources WHERE project_id = ? AND deleted_at IS NULL)", projectID).
		Group("source_documents.id, source_snapshots.source_id, source_documents.snapshot_id").
		Order("source_documents.created_at ASC, source_documents.id ASC").Scan(&rows).Error
	return rows, err
}

type retrievalSourceRow struct {
	ChunkID            string                    `gorm:"column:chunk_id"`
	DocumentID         string                    `gorm:"column:document_id"`
	DocumentTitle      string                    `gorm:"column:document_title"`
	DocumentLocator    string                    `gorm:"column:document_locator"`
	DocumentMediaType  string                    `gorm:"column:document_media_type"`
	DocumentHash       string                    `gorm:"column:document_hash"`
	ArtifactPath       string                    `gorm:"column:artifact_path"`
	ChunkOrdinal       int                       `gorm:"column:chunk_ordinal"`
	ChunkHeadingPath   []byte                    `gorm:"column:chunk_heading_path"`
	ChunkLocator       []byte                    `gorm:"column:chunk_locator"`
	ChunkParagraph     *int                      `gorm:"column:chunk_paragraph"`
	ChunkCodeLanguage  string                    `gorm:"column:chunk_code_language"`
	ChunkStartByte     int                       `gorm:"column:chunk_start_byte"`
	ChunkEndByte       int                       `gorm:"column:chunk_end_byte"`
	ChunkTextHash      string                    `gorm:"column:chunk_text_hash"`
	ChunkSearchText    string                    `gorm:"column:chunk_search_text"`
	SnapshotID         uuid.UUID                 `gorm:"column:snapshot_id"`
	SourceID           uuid.UUID                 `gorm:"column:source_id"`
	SourceKind         sharedtextbook.SourceKind `gorm:"column:source_kind"`
	SourceRole         sharedtextbook.SourceRole `gorm:"column:source_role"`
	SourceLocator      string                    `gorm:"column:source_locator"`
	ResolvedVersion    string                    `gorm:"column:resolved_version"`
	SnapshotHash       string                    `gorm:"column:snapshot_hash"`
	SnapshotCapturedAt time.Time                 `gorm:"column:snapshot_captured_at"`
}

// RetrieveSourceEvidence reads only active, captured primary and explicitly
// confirmed official sources, ranks at most 500 matching chunks locally, then persists
// the decision metadata for author review and later reproduction.
func (r *GormRepository) RetrieveSourceEvidence(ctx context.Context, workspaceID uuid.UUID, input RetrieveSourceInput) (*SourceRetrievalPlan, error) {
	if _, err := r.GetProject(ctx, workspaceID, input.ProjectID); err != nil {
		return nil, err
	}
	var rows []retrievalSourceRow
	query := r.db.WithContext(ctx).Table("source_chunks").
		Select("source_chunks.id AS chunk_id, source_chunks.document_id, source_documents.title AS document_title, source_documents.canonical_locator AS document_locator, source_documents.media_type AS document_media_type, source_documents.content_hash AS document_hash, source_documents.artifact_path, source_chunks.ordinal AS chunk_ordinal, source_chunks.heading_path AS chunk_heading_path, source_chunks.locator AS chunk_locator, source_chunks.paragraph AS chunk_paragraph, source_chunks.code_language AS chunk_code_language, source_chunks.start_byte AS chunk_start_byte, source_chunks.end_byte AS chunk_end_byte, source_chunks.text_hash AS chunk_text_hash, source_chunks.search_text AS chunk_search_text, source_snapshots.id AS snapshot_id, source_snapshots.source_id, textbook_sources.kind AS source_kind, textbook_sources.role AS source_role, textbook_sources.locator AS source_locator, source_snapshots.resolved_version, source_snapshots.content_hash AS snapshot_hash, source_snapshots.captured_at AS snapshot_captured_at").
		Joins("JOIN source_documents ON source_documents.id = source_chunks.document_id").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").
		Where("textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL AND source_snapshots.status = ? AND (textbook_sources.role = ? OR (textbook_sources.role = ? AND textbook_sources.official_confirmed = TRUE))", input.ProjectID, "captured", sharedtextbook.SourceRolePrimary, sharedtextbook.SourceRoleOfficial)
	err := matchingRetrievalRows(query, input.Query).
		Order("source_snapshots.captured_at DESC, source_documents.created_at ASC, source_chunks.ordinal ASC, source_chunks.id ASC").
		Limit(500).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	candidates := make([]sharedtextbook.RetrievalCandidate, 0, len(rows))
	for _, row := range rows {
		var headingPath []string
		var locator sharedtextbook.EvidenceLocator
		if json.Unmarshal(row.ChunkHeadingPath, &headingPath) != nil || json.Unmarshal(row.ChunkLocator, &locator) != nil || locator.Validate() != nil {
			return nil, ErrInvalidState
		}
		paragraph := 0
		if row.ChunkParagraph != nil {
			paragraph = *row.ChunkParagraph
		}
		snapshot := sharedtextbook.SourceSnapshot{ID: row.SnapshotID.String(), SourceID: row.SourceID.String(), Kind: row.SourceKind, Role: row.SourceRole, Locator: row.SourceLocator, ResolvedVersion: row.ResolvedVersion, ContentHash: sourceSnapshotDigest(row.SnapshotHash), CapturedAt: row.SnapshotCapturedAt}
		document := sharedtextbook.SourceDocument{ID: row.DocumentID, SnapshotID: snapshot.ID, CanonicalLocator: row.DocumentLocator, Title: row.DocumentTitle, MediaType: row.DocumentMediaType, ContentHash: row.DocumentHash, ArtifactPath: row.ArtifactPath}
		chunk := sharedtextbook.SourceChunk{ID: row.ChunkID, DocumentID: row.DocumentID, Ordinal: row.ChunkOrdinal, HeadingPath: headingPath, Locator: locator, Paragraph: paragraph, CodeLanguage: row.ChunkCodeLanguage, StartByte: row.ChunkStartByte, EndByte: row.ChunkEndByte, TextHash: row.ChunkTextHash, SearchText: row.ChunkSearchText}
		candidates = append(candidates, sharedtextbook.RetrievalCandidate{Snapshot: snapshot, Document: document, Chunk: chunk})
	}
	plan, err := sharedtextbook.RetrieveEvidence(input.Query, candidates, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("retrieve textbook evidence: %w", err)
	}
	return r.persistRetrievalPlan(ctx, input.ProjectID, plan)
}

func (r *GormRepository) persistRetrievalPlan(ctx context.Context, projectID uuid.UUID, plan sharedtextbook.RetrievalPlan) (*SourceRetrievalPlan, error) {
	candidates := retrievalPlanCandidates(plan.Candidates)
	selected := retrievalPlanCandidates(plan.Selected)
	candidatesJSON, err := marshalStructuredJSON(candidates)
	if err != nil {
		return nil, err
	}
	selectedJSON, err := marshalStructuredJSON(selected)
	if err != nil {
		return nil, err
	}
	var run RetrievalRun
	err = r.db.WithContext(ctx).Where("project_id = ? AND input_hash = ?", projectID, plan.InputHash).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		run = RetrievalRun{ProjectID: projectID, Query: plan.Query, CandidatesJSON: datatypes.JSON(candidatesJSON), SelectedJSON: datatypes.JSON(selectedJSON), InputHash: plan.InputHash}
		if err := r.db.WithContext(ctx).Create(&run).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return sourceRetrievalPlanFromRun(run)
}

func sourceRetrievalPlanFromRun(run RetrievalRun) (*SourceRetrievalPlan, error) {
	var candidates, selected []SourceRetrievalCandidate
	if json.Unmarshal(run.CandidatesJSON, &candidates) != nil || json.Unmarshal(run.SelectedJSON, &selected) != nil {
		return nil, ErrInvalidState
	}
	return &SourceRetrievalPlan{ID: run.ID, Query: run.Query, InputHash: run.InputHash, Candidates: candidates, Selected: selected, CreatedAt: run.CreatedAt}, nil
}

func retrievalPlanCandidates(candidates []sharedtextbook.RetrievalCandidate) []SourceRetrievalCandidate {
	result := make([]SourceRetrievalCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, SourceRetrievalCandidate{ChunkID: candidate.Chunk.ID, DocumentID: candidate.Document.ID, DocumentTitle: candidate.Document.Title, SnapshotID: candidate.Snapshot.ID, SourceRole: candidate.Snapshot.Role, CanonicalLocator: candidate.Document.CanonicalLocator, ArtifactPath: candidate.Document.ArtifactPath, Ordinal: candidate.Chunk.Ordinal, Score: candidate.Score, Reasons: append([]string(nil), candidate.Reasons...)})
		result[len(result)-1].Locator = candidate.Chunk.Locator
	}
	return result
}

// PersistDocuments stores only source structures whose snapshot belongs to the workspace.
func (r *GormRepository) PersistDocuments(ctx context.Context, workspaceID uuid.UUID, input PersistDocumentsInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var snapshot SourceSnapshot
		query := tx.Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").Joins("JOIN textbook_projects ON textbook_projects.id = textbook_sources.project_id").Where("source_snapshots.id = ? AND textbook_projects.workspace_id = ?", input.SnapshotID, workspaceID).First(&snapshot)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if query.Error != nil {
			return query.Error
		}
		return persistDocumentsTx(tx, input.SnapshotID, input.Documents, input.Chunks)
	})
}

func persistDocumentsTx(tx *gorm.DB, snapshotID uuid.UUID, documents []sharedtextbook.SourceDocument, chunks []sharedtextbook.SourceChunk) error {
	for _, document := range documents {
		if document.SnapshotID != snapshotID.String() {
			return ErrInvalidState
		}
		if err := document.Validate(); err != nil {
			return fmt.Errorf("validate source document: %w", err)
		}
		row := ParsedDocument{ID: document.ID, SnapshotID: snapshotID, CanonicalLocator: document.CanonicalLocator, Title: document.Title, MediaType: document.MediaType, ContentHash: document.ContentHash, ParentID: optionalString(document.ParentID), ArtifactPath: document.ArtifactPath}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	for _, chunk := range chunks {
		if err := chunk.Validate(); err != nil {
			return fmt.Errorf("validate source chunk: %w", err)
		}
		headingPath, err := marshalStructuredJSON(orEmptyStrings(chunk.HeadingPath))
		if err != nil {
			return err
		}
		locator, err := marshalStructuredJSON(chunk.Locator)
		if err != nil {
			return err
		}
		row := ParsedChunk{ID: chunk.ID, DocumentID: chunk.DocumentID, Ordinal: chunk.Ordinal, HeadingPath: datatypes.JSON(headingPath), Locator: datatypes.JSON(locator), Paragraph: optionalPositiveInt(chunk.Paragraph), CodeLanguage: chunk.CodeLanguage, StartByte: chunk.StartByte, EndByte: chunk.EndByte, TextHash: chunk.TextHash, SearchText: chunk.SearchText}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func optionalPositiveInt(value int) *int {
	if value < 1 {
		return nil
	}
	return &value
}
func orEmptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

type GormRepository struct {
	db               *gorm.DB
	generationTarget sharedtextbook.SampleGenerationTarget
}

func NewGormRepository(db *gorm.DB, targets ...sharedtextbook.SampleGenerationTarget) *GormRepository {
	target := sharedtextbook.SampleGenerationTarget{
		ProviderName: sharedtextbook.SampleFixtureProviderName,
		ModelName:    sharedtextbook.SampleFixtureModelName,
	}
	if len(targets) > 0 {
		target = targets[0]
	}
	return &GormRepository{db: db, generationTarget: target}
}

func (r *GormRepository) candidateMatchesGenerationTarget(candidate ChapterRevision) bool {
	return r.generationTarget.Validate() == nil &&
		candidate.ProviderName == r.generationTarget.ProviderName &&
		candidate.ModelName == r.generationTarget.ModelName
}

func (r *GormRepository) CreateProject(ctx context.Context, input CreateProjectInput) (*Project, error) {
	project := &Project{WorkspaceID: input.WorkspaceID, Title: strings.TrimSpace(input.Title), AudienceLevel: input.Audience, Status: StatusDraft, RevisionVersion: 1}
	source := &Source{Kind: input.Primary.Kind, Role: sharedtextbook.SourceRolePrimary, Locator: strings.TrimSpace(input.Primary.Locator), OfficialConfirmed: input.Primary.OfficialConfirmed, LicenseStatus: input.Primary.LicenseStatus}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		source.ProjectID = project.ID
		if err := tx.Create(source).Error; err != nil {
			return err
		}
		return tx.Model(project).Update("primary_source_id", source.ID).Error
	}); err != nil {
		return nil, fmt.Errorf("create textbook project: %w", err)
	}
	project.PrimarySourceID = &source.ID
	return project, nil
}

func (r *GormRepository) ListProjects(ctx context.Context, workspaceID uuid.UUID) ([]Project, error) {
	var projects []Project
	err := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID).Order("updated_at DESC").Find(&projects).Error
	return projects, err
}

func (r *GormRepository) GetProject(ctx context.Context, workspaceID, projectID uuid.UUID) (*Project, error) {
	var project Project
	err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", projectID, workspaceID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &project, err
}

// GetProjectWorkspace returns only resources reachable from the workspace-owned project.
func (r *GormRepository) GetProjectWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*ProjectWorkspace, error) {
	project, err := r.GetProject(ctx, workspaceID, projectID)
	if err != nil {
		return nil, err
	}

	// JSON clients render both collections immediately; encode an empty project
	// as [] rather than null so a newly created workspace remains usable.
	workspace := &ProjectWorkspace{Project: project, Sources: []Source{}, Chapters: []Chapter{}}
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("created_at ASC").Find(&workspace.Sources).Error; err != nil {
		return nil, fmt.Errorf("list textbook sources: %w", err)
	}
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("sort_order ASC, created_at ASC").Find(&workspace.Chapters).Error; err != nil {
		return nil, fmt.Errorf("list textbook chapters: %w", err)
	}
	var bookContract BookContractRevision
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("revision_number DESC").First(&bookContract).Error; err == nil {
		workspace.BookContract = &bookContract
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get latest book contract: %w", err)
	}
	var styleSheet StyleSheetRevision
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("revision_number DESC").First(&styleSheet).Error; err == nil {
		workspace.StyleSheet = &styleSheet
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get latest style sheet: %w", err)
	}
	var blueprint BlueprintRevision
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("revision_number DESC").First(&blueprint).Error; err == nil {
		workspace.Blueprint = &blueprint
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get latest blueprint: %w", err)
	}
	var latestBuild BookBuildRow
	if err := r.db.WithContext(ctx).Where("project_id = ?", project.ID).Order("created_at DESC, id DESC").First(&latestBuild).Error; err == nil {
		workspace.LatestBookBuild = &latestBuild
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get latest book build: %w", err)
	}
	return workspace, nil
}

// GetProjectProgress returns the production pipeline state derived from persisted facts.
func (r *GormRepository) GetProjectProgress(ctx context.Context, workspaceID, projectID uuid.UUID) (*ProjectProgress, error) {
	project, err := r.GetProject(ctx, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	var sourceCount, evidenceCount, chapterCount, approvedChapterCount, artifactCount, currentVerifiedEvidenceCount, currentBookBuildCount, currentPublicationCandidateCount int64
	if err := r.db.WithContext(ctx).Model(&Source{}).Where("project_id = ?", project.ID).Count(&sourceCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Table("source_chunks").
		Joins("JOIN source_documents ON source_documents.id = source_chunks.document_id").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").
		Where("textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL", project.ID).
		Count(&evidenceCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&Chapter{}).Where("project_id = ?", project.ID).Count(&chapterCount).Error; err != nil {
		return nil, err
	}
	if project.ApprovedBlueprintRevisionID != nil {
		if err := r.db.WithContext(ctx).Table("textbook_chapters").
			Joins("JOIN chapter_revisions ON chapter_revisions.id = textbook_chapters.approved_revision_id").
			Where("textbook_chapters.project_id = ? AND chapter_revisions.blueprint_revision_id = ?", project.ID, project.ApprovedBlueprintRevisionID).
			Count(&approvedChapterCount).Error; err != nil {
			return nil, err
		}
	}
	requiredSampleProfiles, approvedSampleProfiles, err := r.sampleProfileCoverage(ctx, project)
	if err != nil {
		return nil, err
	}
	missingSampleProfiles := missingChapterProfiles(requiredSampleProfiles, approvedSampleProfiles)
	projectRevisionIDs := r.db.WithContext(ctx).Model(&ChapterRevision{}).
		Select("chapter_revisions.id").
		Joins("JOIN textbook_chapters ON textbook_chapters.id = chapter_revisions.chapter_id").
		Where("textbook_chapters.project_id = ?", project.ID)
	var projectArtifacts []CodeArtifactRow
	if err := r.db.WithContext(ctx).Where("revision_id IN (?)", projectRevisionIDs).Find(&projectArtifacts).Error; err != nil {
		return nil, err
	}
	artifactCount = int64(len(projectArtifacts))
	if len(projectArtifacts) > 0 {
		artifactIDs := make([]uuid.UUID, 0, len(projectArtifacts))
		for _, artifact := range projectArtifacts {
			artifactIDs = append(artifactIDs, artifact.ID)
		}
		var projectEvidence []RuntimeEvidenceRow
		if err := r.db.WithContext(ctx).Where("code_artifact_id IN ?", artifactIDs).Find(&projectEvidence).Error; err != nil {
			return nil, err
		}
		bookContractHash, styleSheetHash, err := r.currentApprovedContractHashes(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		deriveRuntimeEvidenceCurrentness(projectArtifacts, projectEvidence, bookContractHash, styleSheetHash, now)
		for _, evidence := range projectEvidence {
			if evidence.StaleReason == "" && runtimeEvidenceContract(evidence).IsCurrent(now) {
				currentVerifiedEvidenceCount++
			}
		}
	}
	var candidateRows []ChapterRevision
	if err := r.db.WithContext(ctx).Table("chapter_revisions AS revision").Select("revision.*").
		Joins("JOIN textbook_chapters AS chapter ON chapter.current_revision_id = revision.id").
		Joins("LEFT JOIN textbook_candidate_reviews AS review ON review.candidate_revision_id = revision.id").
		Where("chapter.project_id = ? AND chapter.deleted_at IS NULL AND revision.kind = ? AND review.id IS NULL", project.ID, sharedtextbook.RevisionKindCandidate).
		Find(&candidateRows).Error; err != nil {
		return nil, err
	}
	candidateCount := int64(len(candidateRows))
	var currentCandidateCount int64
	for _, candidate := range candidateRows {
		if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil || project.ApprovedBlueprintRevisionID == nil ||
			candidate.BookContractRevisionID == nil || *candidate.BookContractRevisionID != *project.ApprovedBookContractRevisionID ||
			candidate.StyleSheetRevisionID == nil || *candidate.StyleSheetRevisionID != *project.ApprovedStyleSheetRevisionID ||
			candidate.BlueprintRevisionID == nil || *candidate.BlueprintRevisionID != *project.ApprovedBlueprintRevisionID {
			continue
		}
		if sharedtextbook.ValidateCurrentSampleQualityReport(json.RawMessage(candidate.QualityReportJSON)) == nil && r.candidateMatchesGenerationTarget(candidate) {
			currentCandidateCount++
		}
	}
	if project.ApprovedBookContractRevisionID != nil && project.ApprovedStyleSheetRevisionID != nil {
		if err := r.db.WithContext(ctx).Model(&BookBuildRow{}).
			Where("project_id = ? AND book_contract_revision_id = ? AND style_sheet_revision_id = ?", project.ID, *project.ApprovedBookContractRevisionID, *project.ApprovedStyleSheetRevisionID).
			Count(&currentBookBuildCount).Error; err != nil {
			return nil, err
		}
		if err := r.db.WithContext(ctx).Model(&BookBuildRow{}).
			Where("project_id = ? AND book_contract_revision_id = ? AND style_sheet_revision_id = ? AND status = ?", project.ID, *project.ApprovedBookContractRevisionID, *project.ApprovedStyleSheetRevisionID, sharedtextbook.BookBuildPublicationCandidate).
			Count(&currentPublicationCandidateCount).Error; err != nil {
			return nil, err
		}
	}

	sources := ProjectStageState{Key: "sources", Status: "in_progress", Reason: "已登记资料，等待形成不可变快照和可引用片段。"}
	if sourceCount == 0 {
		sources = ProjectStageState{Key: "sources", Status: "blocked", Reason: "请先登记唯一主资料。"}
	} else if evidenceCount > 0 {
		sources = ProjectStageState{Key: "sources", Status: "approved", Reason: fmt.Sprintf("已有 %d 个可引用资料片段。", evidenceCount)}
	}

	blueprint := ProjectStageState{Key: "blueprint", Status: "blocked", Reason: "需要已批准的 BookContract、StyleSheet 和可引用资料片段。"}
	if project.ApprovedBlueprintRevisionID != nil {
		blueprint = ProjectStageState{Key: "blueprint", Status: "approved", Reason: "当前蓝图已人工批准，并绑定当前生成契约。"}
	} else if project.ApprovedBookContractRevisionID != nil && project.ApprovedStyleSheetRevisionID != nil && evidenceCount > 0 {
		blueprint = ProjectStageState{Key: "blueprint", Status: "ready", Reason: "可以基于当前契约和资料创建蓝图草稿。"}
	}

	sample := ProjectStageState{Key: "sample", Status: "blocked", Reason: "需要先批准蓝图。"}
	if approvedChapterCount > 0 && len(missingSampleProfiles) == 0 {
		sample = ProjectStageState{Key: "sample", Status: "approved", Reason: fmt.Sprintf("已人工批准覆盖 %s 的代表性样章。", joinChapterProfiles(requiredSampleProfiles))}
	} else if approvedChapterCount > 0 {
		sample = ProjectStageState{Key: "sample", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 章人工批准的样章，但仍缺少 %s 画像的代表性样章。", approvedChapterCount, joinChapterProfiles(missingSampleProfiles))}
	} else if currentCandidateCount > 0 {
		sample = ProjectStageState{Key: "sample", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 份符合当前质量合同与生成目标的候选稿，等待人工审阅和批准。", currentCandidateCount)}
	} else if candidateCount > 0 && project.ApprovedBlueprintRevisionID != nil {
		sample = ProjectStageState{Key: "sample", Status: "ready", Reason: fmt.Sprintf("已有 %d 份候选稿，但都不符合当前合同、已批准生成输入或生成目标；请按当前规则重新生成。", candidateCount)}
	} else if project.ApprovedBlueprintRevisionID != nil {
		sample = ProjectStageState{Key: "sample", Status: "ready", Reason: "可为蓝图中已映射证据的章节冻结输入并生成候选稿。"}
	}

	chapters := ProjectStageState{Key: "chapters", Status: "blocked", Reason: "需要已批准蓝图和覆盖主要章节画像的样章。"}
	if approvedChapterCount > 0 && len(missingSampleProfiles) == 0 {
		chapters = ProjectStageState{Key: "chapters", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 章人工批准的母稿修订。", approvedChapterCount)}
	} else if approvedChapterCount > 0 {
		chapters = ProjectStageState{Key: "chapters", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 章人工批准的母稿修订；仍缺少 %s 画像的代表性样章，批量生产尚未就绪。", approvedChapterCount, joinChapterProfiles(missingSampleProfiles))}
	} else if chapterCount > 0 {
		chapters = ProjectStageState{Key: "chapters", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 个章节位置，尚无批准母稿。", chapterCount)}
	} else if project.ApprovedBlueprintRevisionID != nil {
		chapters = ProjectStageState{Key: "chapters", Status: "ready", Reason: "可按已批准蓝图建立章节与候选稿。"}
	}

	verification := ProjectStageState{Key: "verification", Status: "blocked", Reason: "需要先生成并人工批准候选稿，才能从批准修订创建教学工件。"}
	if candidateCount > 0 {
		verification = ProjectStageState{Key: "verification", Status: "blocked", Reason: "候选稿已生成；需要先人工批准，才能从批准修订创建教学工件。"}
	}
	if approvedChapterCount > 0 {
		verification = ProjectStageState{Key: "verification", Status: "ready", Reason: fmt.Sprintf("已有 %d 章批准母稿，可创建教学工件并提交隔离验证。", approvedChapterCount)}
	}
	if artifactCount > 0 {
		verification = ProjectStageState{Key: "verification", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 份生成教学工件，但尚无当前有效的隔离运行证据；请勿将预期输出标为已实测。", artifactCount)}
	}
	if currentVerifiedEvidenceCount > 0 {
		verification = ProjectStageState{Key: "verification", Status: "approved", Reason: fmt.Sprintf("已有 %d 份当前有效的隔离运行证据；其余工件仍需单独查看验证状态。", currentVerifiedEvidenceCount)}
	}

	learning := ProjectStageState{Key: "learning", Status: "blocked", Reason: "需要先人工批准至少一章，才能从批准修订创建六维学习目标。"}
	if approvedChapterCount > 0 {
		learning = ProjectStageState{Key: "learning", Status: "ready", Reason: fmt.Sprintf("已有 %d 章批准母稿，可从批准修订创建六维学习目标；到期任务仅在打开应用时检查。", approvedChapterCount)}
	}
	publication := ProjectStageState{Key: "publication", Status: "blocked", Reason: "需要先人工批准至少一章，才能冻结待审构建。"}
	if approvedChapterCount > 0 {
		publication = ProjectStageState{Key: "publication", Status: "ready", Reason: fmt.Sprintf("已有 %d 章批准母稿，可冻结待审构建；它仍需技术、权利、文字、版式和试学审校。", approvedChapterCount)}
	}
	if currentBookBuildCount > 0 {
		publication = ProjectStageState{Key: "publication", Status: "in_progress", Reason: fmt.Sprintf("已有 %d 个绑定当前合同的待审构建；仍需完成技术、权利、文字、版式和试学审校。", currentBookBuildCount)}
	}
	if currentPublicationCandidateCount > 0 {
		publication = ProjectStageState{Key: "publication", Status: "approved", Reason: fmt.Sprintf("已有 %d 个构建通过 InkWords 内部出版预检；这不代表出版社、ISBN 或 CIP 批准。", currentPublicationCandidateCount)}
	}

	return &ProjectProgress{Stages: []ProjectStageState{
		sources,
		blueprint,
		sample,
		chapters,
		verification,
		learning,
		publication,
	}}, nil
}

// sampleProfileCoverage reads the approved BookContract rather than inferring
// coverage from the latest draft. A new unapproved contract cannot silently
// expand the required sample set for the current production pipeline.
func (r *GormRepository) sampleProfileCoverage(ctx context.Context, project *Project) ([]sharedtextbook.ChapterProfile, []sharedtextbook.ChapterProfile, error) {
	if project == nil || project.ApprovedBookContractRevisionID == nil {
		return nil, nil, nil
	}
	var contract BookContractRevision
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ? AND status = ?", *project.ApprovedBookContractRevisionID, project.ID, StatusApproved).First(&contract).Error; err != nil {
		return nil, nil, fmt.Errorf("get approved book contract for sample coverage: %w", err)
	}
	var document sharedtextbook.BookContract
	if err := json.Unmarshal(contract.DocumentJSON, &document); err != nil || document.Validate() != nil {
		return nil, nil, ErrInvalidState
	}
	profiles := make([]sharedtextbook.ChapterProfile, 0)
	if project.ApprovedBlueprintRevisionID == nil {
		return document.ChapterProfiles, profiles, nil
	}
	if err := r.db.WithContext(ctx).Table("textbook_chapters AS chapter").
		Select("chapter.chapter_profile").
		Joins("JOIN chapter_revisions AS revision ON revision.id = chapter.approved_revision_id").
		Where("chapter.project_id = ? AND revision.blueprint_revision_id = ?", project.ID, *project.ApprovedBlueprintRevisionID).
		Scan(&profiles).Error; err != nil {
		return nil, nil, fmt.Errorf("list approved sample chapter profiles: %w", err)
	}
	return document.ChapterProfiles, profiles, nil
}

func missingChapterProfiles(required, approved []sharedtextbook.ChapterProfile) []sharedtextbook.ChapterProfile {
	seen := make(map[sharedtextbook.ChapterProfile]bool, len(approved))
	for _, profile := range approved {
		seen[profile] = true
	}
	missing := make([]sharedtextbook.ChapterProfile, 0, len(required))
	for _, profile := range required {
		if !seen[profile] {
			missing = append(missing, profile)
		}
	}
	return missing
}

func joinChapterProfiles(profiles []sharedtextbook.ChapterProfile) string {
	labels := map[sharedtextbook.ChapterProfile]string{
		sharedtextbook.ChapterProfileConcept:           "概念解释",
		sharedtextbook.ChapterProfileHandsOn:           "动手实践",
		sharedtextbook.ChapterProfileSourceWalkthrough: "源码走读",
		sharedtextbook.ChapterProfileProjectIteration:  "项目迭代",
		sharedtextbook.ChapterProfileTroubleshooting:   "故障排查",
		sharedtextbook.ChapterProfileIntegrationReview: "集成复盘",
		sharedtextbook.ChapterProfileReference:         "参考手册",
	}
	values := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		values = append(values, labels[profile])
	}
	return strings.Join(values, "、")
}

// CreateBookBuild freezes only the currently approved chapter revisions. It
// never follows drafts and it always starts in review state: publication is a
// separate, human-gated decision.
func (r *GormRepository) CreateBookContract(ctx context.Context, workspaceID uuid.UUID, input CreateBookContractInput) (*BookContractRevision, error) {
	var revision *BookContractRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockWorkspaceProject(tx, workspaceID, input.ProjectID)
		if err != nil {
			return err
		}
		if input.Reader.Audience != project.AudienceLevel {
			return ErrInvalidState
		}
		var latest BookContractRevision
		next := 1
		if err := tx.Where("project_id = ?", project.ID).Order("revision_number DESC").First(&latest).Error; err == nil {
			next = latest.RevisionNumber + 1
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		id := uuid.New()
		contract := sharedtextbook.BookContract{RevisionID: id.String(), ProjectID: project.ID.String(), RevisionNumber: next, ContentHash: digestJSON(bookContractPayload(input)), Reader: input.Reader, Promise: strings.TrimSpace(input.Promise), ChapterProfiles: input.ChapterProfiles, TerminologyVersion: strings.TrimSpace(input.TerminologyVersion), PublicationProfile: strings.TrimSpace(input.PublicationProfile)}
		if err := contract.Validate(); err != nil {
			return fmt.Errorf("validate book contract: %w", err)
		}
		document, err := json.Marshal(contract)
		if err != nil {
			return err
		}
		revision = &BookContractRevision{ID: id, ProjectID: project.ID, RevisionNumber: next, DocumentJSON: datatypes.JSON(document), ContentHash: strings.TrimPrefix(contract.ContentHash, "sha256:"), Status: StatusDraft}
		return tx.Create(revision).Error
	})
	return revision, err
}

func (r *GormRepository) CreateStyleSheet(ctx context.Context, workspaceID uuid.UUID, input CreateStyleSheetInput) (*StyleSheetRevision, error) {
	var revision *StyleSheetRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockWorkspaceProject(tx, workspaceID, input.ProjectID)
		if err != nil {
			return err
		}
		var latest StyleSheetRevision
		next := 1
		if err := tx.Where("project_id = ?", project.ID).Order("revision_number DESC").First(&latest).Error; err == nil {
			next = latest.RevisionNumber + 1
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		id := uuid.New()
		sheet := sharedtextbook.StyleSheet{RevisionID: id.String(), ProjectID: project.ID.String(), RevisionNumber: next, ContentHash: digestJSON(styleSheetPayload(input)), Language: strings.TrimSpace(input.Language), TerminologyRules: input.TerminologyRules, CodeRules: input.CodeRules, VisualRules: input.VisualRules, CitationRules: input.CitationRules, ForbiddenPhrases: input.ForbiddenPhrases}
		if err := sheet.Validate(); err != nil {
			return fmt.Errorf("validate style sheet: %w", err)
		}
		document, err := json.Marshal(sheet)
		if err != nil {
			return err
		}
		revision = &StyleSheetRevision{ID: id, ProjectID: project.ID, RevisionNumber: next, DocumentJSON: datatypes.JSON(document), ContentHash: strings.TrimPrefix(sheet.ContentHash, "sha256:"), Status: StatusDraft}
		return tx.Create(revision).Error
	})
	return revision, err
}

func (r *GormRepository) ApproveBookContract(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*BookContractRevision, error) {
	var revision BookContractRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockWorkspaceProject(tx, workspaceID, projectID); err != nil {
			return err
		}
		if err := tx.Where("id = ? AND project_id = ?", revisionID, projectID).First(&revision).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Model(&BookContractRevision{}).Where("id = ?", revision.ID).Update("status", StatusApproved).Error; err != nil {
			return err
		}
		return tx.Model(&Project{}).Where("id = ?", projectID).Updates(map[string]any{
			"approved_book_contract_revision_id": revision.ID,
			"approved_blueprint_revision_id":     nil,
		}).Error
	})
	if err == nil {
		revision.Status = StatusApproved
	}
	return &revision, err
}

func (r *GormRepository) ApproveStyleSheet(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*StyleSheetRevision, error) {
	var revision StyleSheetRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockWorkspaceProject(tx, workspaceID, projectID); err != nil {
			return err
		}
		if err := tx.Where("id = ? AND project_id = ?", revisionID, projectID).First(&revision).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Model(&StyleSheetRevision{}).Where("id = ?", revision.ID).Update("status", StatusApproved).Error; err != nil {
			return err
		}
		return tx.Model(&Project{}).Where("id = ?", projectID).Updates(map[string]any{
			"approved_style_sheet_revision_id": revision.ID,
			"approved_blueprint_revision_id":   nil,
		}).Error
	})
	if err == nil {
		revision.Status = StatusApproved
	}
	return &revision, err
}

func lockWorkspaceProject(tx *gorm.DB, workspaceID, projectID uuid.UUID) (*Project, error) {
	var project Project
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND workspace_id = ?", projectID, workspaceID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func digestJSON(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func bookContractPayload(input CreateBookContractInput) any {
	return struct {
		Reader             sharedtextbook.ReaderModel      `json:"reader"`
		Promise            string                          `json:"promise"`
		ChapterProfiles    []sharedtextbook.ChapterProfile `json:"chapter_profiles"`
		TerminologyVersion string                          `json:"terminology_version"`
		PublicationProfile string                          `json:"publication_profile"`
	}{input.Reader, strings.TrimSpace(input.Promise), input.ChapterProfiles, strings.TrimSpace(input.TerminologyVersion), strings.TrimSpace(input.PublicationProfile)}
}

func styleSheetPayload(input CreateStyleSheetInput) any {
	return struct {
		Language         string   `json:"language"`
		TerminologyRules []string `json:"terminology_rules"`
		CodeRules        []string `json:"code_rules"`
		VisualRules      []string `json:"visual_rules"`
		CitationRules    []string `json:"citation_rules"`
		ForbiddenPhrases []string `json:"forbidden_phrases"`
	}{strings.TrimSpace(input.Language), input.TerminologyRules, input.CodeRules, input.VisualRules, input.CitationRules, input.ForbiddenPhrases}
}

// GetChapterWorkspace verifies the project boundary before loading its immutable history.
func (r *GormRepository) GetChapterWorkspace(ctx context.Context, workspaceID, chapterID uuid.UUID) (*ChapterWorkspace, error) {
	var chapter Chapter
	if err := r.db.WithContext(ctx).Where("id = ?", chapterID).First(&chapter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := r.GetProject(ctx, workspaceID, chapter.ProjectID); err != nil {
		return nil, err
	}

	chapterWorkspace := &ChapterWorkspace{
		Chapter:             &chapter,
		GenerationTarget:    r.generationTarget,
		HumanReviewContract: sharedtextbook.CurrentSampleHumanReviewContract(),
	}
	var latestSampleTask ChapterGenerationTaskRef
	if err := r.db.WithContext(ctx).Table("job_tasks").
		Select("id AS task_id, status, created_at").
		Where("workspace_id = ? AND textbook_chapter_id = ? AND task_subtype = ?", workspaceID, chapter.ID, sharedtextbook.TextbookSampleGenerationTaskSubtype).
		Order("created_at DESC, id DESC").
		Take(&latestSampleTask).Error; err == nil {
		chapterWorkspace.LatestSampleTask = &latestSampleTask
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get latest chapter generation task: %w", err)
	}
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("revision_number DESC").Find(&chapterWorkspace.Revisions).Error; err != nil {
		return nil, fmt.Errorf("list chapter revisions: %w", err)
	}
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).Order("created_at DESC, id DESC").Find(&chapterWorkspace.CandidateReviews).Error; err != nil {
		return nil, fmt.Errorf("list candidate reviews: %w", err)
	}
	revisionIDs := r.db.WithContext(ctx).Model(&ChapterRevision{}).Select("id").Where("chapter_id = ?", chapter.ID)
	if err := r.db.WithContext(ctx).Where("revision_id IN (?)", revisionIDs).Order("created_at DESC, id DESC").Find(&chapterWorkspace.CodeArtifacts).Error; err != nil {
		return nil, fmt.Errorf("list chapter teaching artifacts: %w", err)
	}
	artifactIDs := r.db.WithContext(ctx).Model(&CodeArtifactRow{}).Select("id").Where("revision_id IN (?)", revisionIDs)
	if err := r.db.WithContext(ctx).Where("code_artifact_id IN (?)", artifactIDs).Order("created_at DESC, id DESC").Find(&chapterWorkspace.RuntimeEvidence).Error; err != nil {
		return nil, fmt.Errorf("list chapter runtime evidence: %w", err)
	}
	currentBookContractHash, currentStyleSheetHash, err := r.currentApprovedContractHashes(ctx, chapter.ProjectID)
	if err != nil {
		return nil, err
	}
	deriveRuntimeEvidenceCurrentness(chapterWorkspace.CodeArtifacts, chapterWorkspace.RuntimeEvidence, currentBookContractHash, currentStyleSheetHash, time.Now().UTC())
	evidenceIDs := r.db.WithContext(ctx).Model(&RuntimeEvidenceRow{}).Select("id").Where("code_artifact_id IN (?)", artifactIDs)
	if err := r.db.WithContext(ctx).Where("evidence_id IN (?)", evidenceIDs).Order("created_at DESC, id DESC").Find(&chapterWorkspace.Assets).Error; err != nil {
		return nil, fmt.Errorf("list chapter manuscript assets: %w", err)
	}
	var lock ChapterLock
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).First(&lock).Error; err == nil {
		chapterWorkspace.Lock = &lock
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get chapter lock: %w", err)
	}
	return chapterWorkspace, nil
}

func (r *GormRepository) currentApprovedContractHashes(ctx context.Context, projectID uuid.UUID) (string, string, error) {
	var row struct {
		BookContractHash string `gorm:"column:book_contract_hash"`
		StyleSheetHash   string `gorm:"column:style_sheet_hash"`
	}
	err := r.db.WithContext(ctx).Table("textbook_projects AS project").
		Select("book_contract.content_hash AS book_contract_hash, style_sheet.content_hash AS style_sheet_hash").
		Joins("LEFT JOIN book_contract_revisions AS book_contract ON book_contract.id = project.approved_book_contract_revision_id").
		Joins("LEFT JOIN style_sheet_revisions AS style_sheet ON style_sheet.id = project.approved_style_sheet_revision_id").
		Where("project.id = ?", projectID).
		Take(&row).Error
	if err != nil {
		return "", "", fmt.Errorf("get approved teaching contract hashes: %w", err)
	}
	return canonicalContractHash(row.BookContractHash), canonicalContractHash(row.StyleSheetHash), nil
}

func canonicalContractHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}

func deriveRuntimeEvidenceCurrentness(artifacts []CodeArtifactRow, evidence []RuntimeEvidenceRow, bookContractHash, styleSheetHash string, now time.Time) {
	artifactsByID := make(map[uuid.UUID]CodeArtifactRow, len(artifacts))
	for _, artifact := range artifacts {
		artifactsByID[artifact.ID] = artifact
	}
	for index := range evidence {
		item := &evidence[index]
		if item.Status != sharedtextbook.ArtifactStatusVerified {
			continue
		}
		artifact, ok := artifactsByID[item.CodeArtifactID]
		if !ok {
			item.StaleReason = "关联的代码工件已不可用，需重新验证。"
			continue
		}
		var manifest sharedtextbook.TeachingArtifactManifest
		if err := json.Unmarshal(artifact.ManifestJSON, &manifest); err != nil {
			item.StaleReason = "教学工件清单无效，不能将此运行记录作为当前证据。"
			continue
		}
		item.StaleReason = sharedtextbook.TeachingArtifactEvidenceStaleReason(manifest, bookContractHash, styleSheetHash, runtimeEvidenceContract(*item), now)
	}
}

func runtimeEvidenceContract(row RuntimeEvidenceRow) sharedtextbook.RuntimeEvidence {
	var samplingConditions []string
	_ = json.Unmarshal(row.SamplingConditionsJSON, &samplingConditions)
	return sharedtextbook.RuntimeEvidence{ID: row.ID.String(), RevisionID: row.RevisionID.String(), CodeArtifactID: row.CodeArtifactID.String(), CodeArtifactHash: row.CodeArtifactHash, InputHash: row.InputHash, Kind: row.Kind, Status: row.Status, CommandManifestHash: row.CommandManifestHash, RunnerImageDigest: row.RunnerImageDigest, ToolchainVersion: row.ToolchainVersion, ToolName: row.ToolName, ToolVersion: row.ToolVersion, SamplingConditions: samplingConditions, StructuredOutput: row.StructuredOutput, RawEvidenceRef: row.RawEvidenceRef, OutputTruncated: row.OutputTruncated, CapturedAt: dereferenceTime(row.CapturedAt), ExpiresAt: row.ExpiresAt, StaleReason: row.StaleReason}
}

func dereferenceTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func (r *GormRepository) AddSource(ctx context.Context, workspaceID, projectID uuid.UUID, input CreateSourceInput) (*Source, error) {
	if _, err := r.GetProject(ctx, workspaceID, projectID); err != nil {
		return nil, err
	}
	source := &Source{ProjectID: projectID, Kind: input.Kind, Role: input.Role, Locator: strings.TrimSpace(input.Locator), OfficialConfirmed: input.OfficialConfirmed, LicenseStatus: input.LicenseStatus}
	if err := r.db.WithContext(ctx).Create(source).Error; err != nil {
		if input.Role == sharedtextbook.SourceRolePrimary {
			return nil, ErrPrimarySourceExists
		}
		return nil, err
	}
	return source, nil
}

func (r *GormRepository) CreateChapter(ctx context.Context, workspaceID uuid.UUID, input CreateChapterInput) (*Chapter, error) {
	if _, err := r.GetProject(ctx, workspaceID, input.ProjectID); err != nil {
		return nil, err
	}
	chapter := &Chapter{ProjectID: input.ProjectID, SortOrder: input.SortOrder, Title: strings.TrimSpace(input.Title), ChapterProfile: input.ChapterProfile, Status: StatusDraft}
	if err := r.db.WithContext(ctx).Create(chapter).Error; err != nil {
		return nil, err
	}
	return chapter, nil
}

// RegisterManuscriptAsset records an immutable visual only after proving that
// its evidence, revision, and chapter all belong to the local workspace.
func (r *GormRepository) RegisterManuscriptAsset(ctx context.Context, workspaceID uuid.UUID, input RegisterManuscriptAssetInput) (*ManuscriptAssetRow, error) {
	var asset *ManuscriptAssetRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var revision ChapterRevision
		if err := tx.Where("id = ?", input.Asset.RevisionID).First(&revision).Error; err != nil || revision.ChapterID != input.ChapterID {
			return ErrNotFound
		}
		var chapter Chapter
		if err := tx.Where("id = ?", input.ChapterID).First(&chapter).Error; err != nil {
			return ErrNotFound
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		var evidence RuntimeEvidenceRow
		if err := tx.Where("id = ? AND revision_id = ?", input.Asset.EvidenceID, revision.ID).First(&evidence).Error; err != nil {
			return ErrNotFound
		}
		var existing ManuscriptAssetRow
		if err := tx.Where("stable_ref = ?", input.Asset.StableRef).First(&existing).Error; err == nil {
			if existing.ContentHash != input.Asset.ContentHash || existing.RevisionID != revision.ID {
				return ErrInvalidState
			}
			asset = &existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		assetID, err := uuid.Parse(input.Asset.ID)
		if err != nil {
			return ErrInvalidState
		}
		evidenceID, err := uuid.Parse(input.Asset.EvidenceID)
		if err != nil {
			return ErrInvalidState
		}
		asset = &ManuscriptAssetRow{ID: assetID, RevisionID: revision.ID, EvidenceID: evidenceID, StableRef: input.Asset.StableRef, Kind: input.Asset.Kind, ContentHash: input.Asset.ContentHash, AltText: input.Asset.AltText, Source: input.Asset.Source, GenerationMethod: input.Asset.GenerationMethod, VisualPurpose: input.Asset.VisualPurpose, RightsStatus: input.Asset.RightsStatus, Status: input.Asset.Status}
		return tx.Create(asset).Error
	})
	return asset, err
}

func (r *GormRepository) AppendRevision(ctx context.Context, workspaceID uuid.UUID, input AppendRevisionInput) (*ChapterRevision, error) {
	var revision *ChapterRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var chapter Chapter
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.ChapterID).First(&chapter)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if query.Error != nil {
			return query.Error
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		if input.GenerationTaskID != nil {
			var existing ChapterRevision
			err := tx.Where("generation_task_id = ?", *input.GenerationTaskID).First(&existing).Error
			if err == nil {
				if existing.ChapterID != chapter.ID || existing.Kind != input.Kind || existing.ContentHash != input.ContentHash {
					return ErrInvalidState
				}
				revision = &existing
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if chapter.RevisionVersion != input.ExpectedVersion {
			return ErrVersionConflict
		}
		if chapter.ApprovedRevisionID != nil && input.Kind == sharedtextbook.RevisionKindApproved {
			return ErrInvalidState
		}
		if input.CreatedBy == RevisionCreatorGeneration {
			// A first candidate starts from the approved blueprint rather than inventing an empty
			// manual revision. Once any revision exists, the explicit parent protects it from
			// stale worker output and guarantees the review diff has a truthful baseline.
			if chapter.CurrentRevisionID == nil && input.ParentRevisionID != nil {
				return ErrVersionConflict
			}
			if chapter.CurrentRevisionID != nil && (input.ParentRevisionID == nil || *input.ParentRevisionID != *chapter.CurrentRevisionID) {
				return ErrVersionConflict
			}
			if err := r.validateGenerationProvenance(tx, project, input); err != nil {
				return err
			}
		}
		if input.Kind == sharedtextbook.RevisionKindApproved {
			var lock ChapterLock
			if err := tx.Where("chapter_id = ?", chapter.ID).First(&lock).Error; err != nil || lock.OwnerID != input.LockOwnerID || lock.Version != input.LockVersion || !lock.LeaseExpiresAt.After(time.Now().UTC()) {
				return ErrRevisionLocked
			}
		}
		candidate := &ChapterRevision{ChapterID: chapter.ID, RevisionNumber: chapter.RevisionVersion + 1, Kind: input.Kind, Markdown: input.Markdown, DocumentJSON: datatypes.JSON(input.DocumentJSON), ContentHash: input.ContentHash, CreatedBy: input.CreatedBy, ParentRevisionID: input.ParentRevisionID, BookContractRevisionID: input.BookContractRevisionID, StyleSheetRevisionID: input.StyleSheetRevisionID, BlueprintRevisionID: input.BlueprintRevisionID, EvidencePackHash: input.EvidencePackHash, PromptHash: input.PromptHash, ProviderName: input.ProviderName, ModelName: input.ModelName, ProviderUsageJSON: datatypes.JSON(input.ProviderUsageJSON), QualityReportJSON: datatypes.JSON(input.QualityReportJSON), GenerationTaskID: input.GenerationTaskID}
		if err := tx.Create(candidate).Error; err != nil {
			return err
		}
		updates := map[string]any{"current_revision_id": candidate.ID, "revision_version": candidate.RevisionNumber, "status": string(input.Kind), "updated_at": gorm.Expr("CURRENT_TIMESTAMP")}
		if input.Kind == sharedtextbook.RevisionKindApproved {
			updates["approved_revision_id"] = candidate.ID
		}
		if result := tx.Model(&Chapter{}).Where("id = ? AND revision_version = ?", chapter.ID, input.ExpectedVersion).Updates(updates); result.Error != nil || result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		revision = candidate
		return nil
	})
	return revision, err
}

func (r *GormRepository) AcquireLock(ctx context.Context, workspaceID uuid.UUID, input LockInput) (*ChapterLock, error) {
	var lock ChapterLock
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var chapter Chapter
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.ChapterID).First(&chapter).Error; err != nil {
			return ErrNotFound
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		if chapter.RevisionVersion != input.ExpectedVersion {
			return ErrVersionConflict
		}
		now := time.Now().UTC()
		if err := tx.Where("chapter_id = ?", chapter.ID).First(&lock).Error; err == nil && lock.OwnerID != input.OwnerID && lock.LeaseExpiresAt.After(now) {
			return ErrRevisionLocked
		}
		lock = ChapterLock{ChapterID: chapter.ID, OwnerID: input.OwnerID, Version: lock.Version + 1, LeaseExpiresAt: now.Add(input.LeaseDuration)}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "chapter_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"owner_id":         lock.OwnerID,
				"version":          lock.Version,
				"lease_expires_at": lock.LeaseExpiresAt,
				"updated_at":       gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).Create(&lock).Error
	})
	return &lock, err
}

func (r *GormRepository) RejectCandidate(ctx context.Context, workspaceID uuid.UUID, input RejectCandidateInput) (*CandidateReview, error) {
	var rejected *CandidateReview
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var chapter Chapter
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.ChapterID).First(&chapter).Error; err != nil {
			return ErrNotFound
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		if chapter.RevisionVersion != input.ExpectedVersion {
			return ErrVersionConflict
		}
		var lock ChapterLock
		if err := tx.Where("chapter_id = ?", chapter.ID).First(&lock).Error; err != nil || lock.OwnerID != input.LockOwnerID || lock.Version != input.LockVersion || !lock.LeaseExpiresAt.After(time.Now().UTC()) {
			return ErrRevisionLocked
		}
		var candidate ChapterRevision
		if err := tx.Where("id = ? AND chapter_id = ? AND kind = ?", input.CandidateRevisionID, chapter.ID, sharedtextbook.RevisionKindCandidate).First(&candidate).Error; err != nil {
			return ErrInvalidState
		}
		if chapter.CurrentRevisionID == nil || *chapter.CurrentRevisionID != candidate.ID {
			return ErrVersionConflict
		}

		var existing CandidateReview
		if err := tx.Where("candidate_revision_id = ?", candidate.ID).First(&existing).Error; err == nil {
			if existing.ReviewerWorkspaceID == workspaceID && existing.Decision == CandidateDecisionRejected && existing.Reason == input.Reason {
				rejected = &existing
				return nil
			}
			return ErrInvalidState
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		qualityContractVersion := "unknown"
		var quality struct {
			ContractVersion string `json:"contract_version"`
		}
		if json.Unmarshal(candidate.QualityReportJSON, &quality) == nil && strings.TrimSpace(quality.ContractVersion) != "" {
			qualityContractVersion = strings.TrimSpace(quality.ContractVersion)
		}
		rejected = &CandidateReview{
			ProjectID:              project.ID,
			ChapterID:              chapter.ID,
			CandidateRevisionID:    candidate.ID,
			ReviewerWorkspaceID:    workspaceID,
			Decision:               CandidateDecisionRejected,
			Reason:                 input.Reason,
			CandidateContentHash:   candidate.ContentHash,
			QualityContractVersion: qualityContractVersion,
			HumanReviewJSON:        datatypes.JSON([]byte(`{}`)),
		}
		return tx.Create(rejected).Error
	})
	return rejected, err
}

func (r *GormRepository) validateGenerationProvenance(tx *gorm.DB, project Project, input AppendRevisionInput) error {
	if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil || project.ApprovedBlueprintRevisionID == nil || input.BookContractRevisionID == nil || input.StyleSheetRevisionID == nil || input.BlueprintRevisionID == nil || *input.BookContractRevisionID != *project.ApprovedBookContractRevisionID || *input.StyleSheetRevisionID != *project.ApprovedStyleSheetRevisionID || *input.BlueprintRevisionID != *project.ApprovedBlueprintRevisionID {
		return ErrInvalidState
	}
	var bookCount, styleCount, blueprintCount int64
	if err := tx.Model(&BookContractRevision{}).Where("id = ? AND project_id = ? AND status = ?", input.BookContractRevisionID, project.ID, StatusApproved).Count(&bookCount).Error; err != nil {
		return err
	}
	if err := tx.Model(&StyleSheetRevision{}).Where("id = ? AND project_id = ? AND status = ?", input.StyleSheetRevisionID, project.ID, StatusApproved).Count(&styleCount).Error; err != nil {
		return err
	}
	if err := tx.Model(&BlueprintRevision{}).Where("id = ? AND project_id = ? AND status = ?", input.BlueprintRevisionID, project.ID, StatusApproved).Count(&blueprintCount).Error; err != nil {
		return err
	}
	if bookCount != 1 || styleCount != 1 || blueprintCount != 1 {
		return ErrInvalidState
	}
	return nil
}

func (r *GormRepository) SoftDeleteProject(ctx context.Context, workspaceID, projectID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", projectID, workspaceID).Delete(&Project{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrNotFound
	}
	return nil
}

var _ Repository = (*GormRepository)(nil)
