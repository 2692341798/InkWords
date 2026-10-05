package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// SampleGenerationTask is the minimal task projection that the textbook HTTP surface exposes.
type SampleGenerationTask struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

// SampleGenerationTaskCreator keeps the HTTP adapter independent of queue implementation.
type SampleGenerationTaskCreator interface {
	PrepareSampleGeneration(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (SampleGenerationPreflight, error)
	CreateSampleGenerationTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (SampleGenerationTask, error)
}

// SourceImportTask is the small task projection exposed after a local file is queued.
type SourceImportTask struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

type SourceImportTaskCreator interface {
	CreateSourceImportTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, io.Reader, string) (SourceImportTask, error)
}

// OfficialWebImportTaskCreator isolates the HTTP adapter from task and crawler
// implementations. Only a confirmed source ID and a narrower path boundary
// arrive from the browser.
type OfficialWebImportTaskCreator interface {
	CreateOfficialWebImportTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (SourceImportTask, error)
}

type VisualAssetUploadCreator interface {
	UploadManualVisualAsset(context.Context, uuid.UUID, VisualAssetUploadInput) (*ManuscriptAssetRow, error)
}

type Handler struct {
	service             *Service
	taskCreator         SampleGenerationTaskCreator
	sourceImportCreator SourceImportTaskCreator
	officialWebCreator  OfficialWebImportTaskCreator
	verificationCreator TextbookVerificationTaskCreator
	visualAssetCreator  VisualAssetUploadCreator
}

func (h *Handler) WithSourceImportCreator(creator SourceImportTaskCreator) *Handler {
	h.sourceImportCreator = creator
	return h
}

func (h *Handler) WithOfficialWebImportCreator(creator OfficialWebImportTaskCreator) *Handler {
	h.officialWebCreator = creator
	return h
}

func (h *Handler) WithVerificationTaskCreator(creator TextbookVerificationTaskCreator) *Handler {
	h.verificationCreator = creator
	return h
}
func (h *Handler) WithVisualAssetCreator(creator VisualAssetUploadCreator) *Handler {
	h.visualAssetCreator = creator
	return h
}

func NewHandler(service *Service, taskCreators ...SampleGenerationTaskCreator) *Handler {
	var taskCreator SampleGenerationTaskCreator
	if len(taskCreators) > 0 {
		taskCreator = taskCreators[0]
	}
	return &Handler{service: service, taskCreator: taskCreator}
}

func (h *Handler) CreateProject(c *gin.Context) {
	workspaceID, ok := textbookWorkspace(c)
	if !ok {
		return
	}
	var request struct {
		Title    string                       `json:"title"`
		Audience sharedtextbook.AudienceLevel `json:"audience_level"`
		Primary  struct {
			Kind    sharedtextbook.SourceKind `json:"kind"`
			Locator string                    `json:"locator"`
		} `json:"primary_source"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	project, err := h.service.CreateProject(c.Request.Context(), CreateProjectInput{WorkspaceID: workspaceID, Title: request.Title, Audience: request.Audience, Primary: CreateSourceInput{Kind: request.Primary.Kind, Locator: request.Primary.Locator}})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": project})
}

func (h *Handler) ListProjects(c *gin.Context) {
	workspaceID, ok := textbookWorkspace(c)
	if !ok {
		return
	}
	projects, err := h.service.repository.ListProjects(c.Request.Context(), workspaceID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": projects})
}

func (h *Handler) GetProject(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	project, err := h.service.GetProject(c.Request.Context(), workspaceID, projectID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": project})
}

// GetProjectWorkspace returns editable sources and chapter headings for a project.
func (h *Handler) GetProjectWorkspace(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	workspace, err := h.service.GetProjectWorkspace(c.Request.Context(), workspaceID, projectID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workspace})
}

// GetProjectProgress returns server-authoritative textbook production stages.
func (h *Handler) GetProjectProgress(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	progress, err := h.service.GetProjectProgress(c.Request.Context(), workspaceID, projectID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": progress})
}

// CreateBookBuild freezes the project's current approved manuscript set.
// Only distribution notices are accepted; callers cannot choose revision IDs.
func (h *Handler) CreateBookBuild(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Notices []sharedtextbook.PublicationNoticeDraft `json:"publication_notices"`
	}
	if c.Request.ContentLength != 0 {
		if err := decodeStrictJSON(c, &request); err != nil {
			textbookError(c, http.StatusBadRequest, "INVALID_STATE")
			return
		}
	}
	build, err := h.service.CreateBookBuild(c.Request.Context(), workspaceID, projectID, request.Notices...)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": build})
}

// GetEditorialWorkspace returns only persisted evidence and derived preflight
// state for a workspace-owned frozen build.
func (h *Handler) GetEditorialWorkspace(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	workspace, err := h.service.GetEditorialWorkspace(c.Request.Context(), workspaceID, buildID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workspace})
}

func (h *Handler) AddRightsItem(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	var request struct {
		SubjectRef        string                        `json:"subject_ref"`
		WorkType          sharedtextbook.RightsWorkType `json:"work_type"`
		RightsBasis       string                        `json:"rights_basis"`
		AllowedUse        string                        `json:"allowed_use"`
		Attribution       string                        `json:"attribution"`
		PublicationStatus sharedtextbook.RightsStatus   `json:"publication_status"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	item, err := h.service.AddRightsItem(c.Request.Context(), workspaceID, AddRightsItemInput{
		BuildID: buildID, SubjectRef: request.SubjectRef, WorkType: request.WorkType,
		RightsBasis: request.RightsBasis, AllowedUse: request.AllowedUse, Attribution: request.Attribution,
		PublicationStatus: request.PublicationStatus,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": item})
}

func (h *Handler) CompletePublicationReview(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	var request CompletePublicationReviewInput
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	request.BuildID = buildID
	review, err := h.service.CompletePublicationReview(c.Request.Context(), workspaceID, request)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": review})
}

// PromoteBookBuild performs the explicit internal preflight transition. It
// does not claim publisher approval and accepts no client-supplied status.
func (h *Handler) PromoteBookBuild(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	build, err := h.service.PromoteBookBuild(c.Request.Context(), workspaceID, buildID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": build})
}

// ListSourceLibrary exposes document metadata without sending full source excerpts to the browser.
func (h *Handler) ListSourceLibrary(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	documents, err := h.service.ListSourceLibrary(c.Request.Context(), workspaceID, projectID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": documents})
}

// ListSourceEvidence exposes bounded evidence identifiers, without returning source excerpts.
func (h *Handler) ListSourceEvidence(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	evidence, err := h.service.ListSourceEvidence(c.Request.Context(), workspaceID, projectID, c.QueryArray("chunk_id")...)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": evidence})
}

// RetrieveSourceEvidence exposes a bounded, persisted explanation of why a
// source chunk is relevant. Raw excerpts remain in the core evidence pack.
func (h *Handler) RetrieveSourceEvidence(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	plan, err := h.service.RetrieveSourceEvidence(c.Request.Context(), workspaceID, RetrieveSourceInput{ProjectID: projectID, Query: request.Query, Limit: request.Limit})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": plan})
}

// GetChapterWorkspace returns a chapter, its immutable revisions, and the current lease.
func (h *Handler) GetChapterWorkspace(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	workspace, err := h.service.GetChapterWorkspace(c.Request.Context(), workspaceID, chapterID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workspace})
}

func (h *Handler) UploadVisualAsset(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	if h.visualAssetCreator == nil {
		textbookError(c, http.StatusServiceUnavailable, "INVALID_STATE")
		return
	}
	revisionID, err := uuid.Parse(c.PostForm("revision_id"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	evidenceID, err := uuid.Parse(c.PostForm("evidence_id"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	defer file.Close()
	asset, err := h.visualAssetCreator.UploadManualVisualAsset(c.Request.Context(), workspaceID, VisualAssetUploadInput{ChapterID: chapterID, RevisionID: revisionID, EvidenceID: evidenceID, MediaType: header.Header.Get("Content-Type"), AltText: c.PostForm("alt_text"), Source: c.PostForm("source"), GenerationMethod: c.PostForm("generation_method"), VisualPurpose: sharedtextbook.VisualEvidencePurpose(c.PostForm("visual_purpose")), RightsStatus: sharedtextbook.RightsStatus(c.PostForm("rights_status")), Content: file})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": asset})
}

func (h *Handler) AddSource(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Kind              sharedtextbook.SourceKind `json:"kind"`
		Role              sharedtextbook.SourceRole `json:"role"`
		Locator           string                    `json:"locator"`
		OfficialConfirmed bool                      `json:"official_confirmed"`
		LicenseStatus     string                    `json:"license_status"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	source, err := h.service.AddSource(c.Request.Context(), workspaceID, projectID, CreateSourceInput{Kind: request.Kind, Role: request.Role, Locator: request.Locator, OfficialConfirmed: request.OfficialConfirmed, LicenseStatus: request.LicenseStatus})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": source})
}

// LoadGinFixture loads the fixed offline source excerpts used by the first Gin sample chapter.
func (h *Handler) LoadGinFixture(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	snapshot, err := h.service.LoadGinFixture(c.Request.Context(), workspaceID, projectID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": snapshot})
}

// CreateSourceImport streams a registered local file into the local artifact
// store. It intentionally accepts multipart only: Base64 JSON would duplicate
// large source bytes in browser memory, HTTP buffers, and task messages.
func (h *Handler) CreateSourceImport(c *gin.Context) {
	if h.sourceImportCreator == nil {
		textbookError(c, http.StatusServiceUnavailable, "SOURCE_IMPORT_UNAVAILABLE")
		return
	}
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	defer file.Close()
	sourceID, err := uuid.Parse(c.PostForm("source_id"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	task, err := h.sourceImportCreator.CreateSourceImportTask(c.Request.Context(), workspaceID, projectID, sourceID, header.Filename, file, c.PostForm("resolved_version"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": task})
}

// CreateOfficialWebImport queues one bounded crawl for a project-owned,
// explicitly confirmed official source. It cannot accept a caller-supplied URL.
func (h *Handler) CreateOfficialWebImport(c *gin.Context) {
	if h.officialWebCreator == nil {
		textbookError(c, http.StatusServiceUnavailable, "OFFICIAL_WEB_IMPORT_UNAVAILABLE")
		return
	}
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		SourceID            uuid.UUID `json:"source_id"`
		AllowedPathPrefixes []string  `json:"allowed_path_prefixes"`
	}
	if err := decodeStrictJSON(c, &request); err != nil || request.SourceID == uuid.Nil || len(request.AllowedPathPrefixes) == 0 {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	task, err := h.officialWebCreator.CreateOfficialWebImportTask(c.Request.Context(), workspaceID, projectID, request.SourceID, request.AllowedPathPrefixes)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": task})
}

func (h *Handler) CreateChapter(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		SortOrder      int                           `json:"sort_order"`
		Title          string                        `json:"title"`
		ChapterProfile sharedtextbook.ChapterProfile `json:"chapter_profile"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	chapter, err := h.service.CreateChapter(c.Request.Context(), workspaceID, CreateChapterInput{ProjectID: projectID, SortOrder: request.SortOrder, Title: request.Title, ChapterProfile: request.ChapterProfile})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": chapter})
}

// CreateBookContract records a draft teaching contract. Approval is a separate explicit action.
func (h *Handler) CreateBookContract(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Reader             sharedtextbook.ReaderModel      `json:"reader"`
		Promise            string                          `json:"promise"`
		ChapterProfiles    []sharedtextbook.ChapterProfile `json:"chapter_profiles"`
		TerminologyVersion string                          `json:"terminology_version"`
		PublicationProfile string                          `json:"publication_profile"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.CreateBookContract(c.Request.Context(), workspaceID, CreateBookContractInput{ProjectID: projectID, Reader: request.Reader, Promise: request.Promise, ChapterProfiles: request.ChapterProfiles, TerminologyVersion: request.TerminologyVersion, PublicationProfile: request.PublicationProfile})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": revision})
}

// CreateStyleSheet records a draft writing contract. Approval is a separate explicit action.
func (h *Handler) CreateStyleSheet(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Language         string   `json:"language"`
		TerminologyRules []string `json:"terminology_rules"`
		CodeRules        []string `json:"code_rules"`
		VisualRules      []string `json:"visual_rules"`
		CitationRules    []string `json:"citation_rules"`
		ForbiddenPhrases []string `json:"forbidden_phrases"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.CreateStyleSheet(c.Request.Context(), workspaceID, CreateStyleSheetInput{ProjectID: projectID, Language: request.Language, TerminologyRules: request.TerminologyRules, CodeRules: request.CodeRules, VisualRules: request.VisualRules, CitationRules: request.CitationRules, ForbiddenPhrases: request.ForbiddenPhrases})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": revision})
}

// CreateBlueprint records a draft teaching outline. Its contract revisions are assigned server-side.
func (h *Handler) CreateBlueprint(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	var request struct {
		Volumes []sharedtextbook.BlueprintVolume `json:"volumes"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.CreateBlueprint(c.Request.Context(), workspaceID, CreateBlueprintInput{ProjectID: projectID, Volumes: request.Volumes})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": revision})
}

// ApproveBookContract makes one reviewed contract eligible for textbook generation.
func (h *Handler) ApproveBookContract(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	revisionID, err := uuid.Parse(c.Param("revisionID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.ApproveBookContract(c.Request.Context(), workspaceID, projectID, revisionID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": revision})
}

// ApproveStyleSheet makes one reviewed style sheet eligible for textbook generation.
func (h *Handler) ApproveStyleSheet(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	revisionID, err := uuid.Parse(c.Param("revisionID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.ApproveStyleSheet(c.Request.Context(), workspaceID, projectID, revisionID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": revision})
}

// ApproveBlueprint makes one reviewed outline eligible for generation.
func (h *Handler) ApproveBlueprint(c *gin.Context) {
	workspaceID, projectID, ok := textbookProjectIDs(c)
	if !ok {
		return
	}
	revisionID, err := uuid.Parse(c.Param("revisionID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.ApproveBlueprint(c.Request.Context(), workspaceID, projectID, revisionID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": revision})
}

func (h *Handler) AcquireLock(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	var request struct {
		OwnerID         uuid.UUID `json:"owner_id"`
		ExpectedVersion int       `json:"expected_version"`
		LeaseSeconds    int       `json:"lease_seconds"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	lock, err := h.service.AcquireLock(c.Request.Context(), workspaceID, LockInput{ChapterID: chapterID, OwnerID: request.OwnerID, ExpectedVersion: request.ExpectedVersion, LeaseDuration: time.Duration(request.LeaseSeconds) * time.Second})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": lock})
}

func (h *Handler) AppendRevision(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	var request struct {
		ExpectedVersion int                         `json:"expected_version"`
		Kind            sharedtextbook.RevisionKind `json:"kind"`
		Markdown        string                      `json:"markdown"`
		DocumentJSON    json.RawMessage             `json:"document_json"`
		ContentHash     string                      `json:"content_hash"`
		LockOwnerID     uuid.UUID                   `json:"lock_owner_id"`
		LockVersion     int                         `json:"lock_version"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	if request.Kind == sharedtextbook.RevisionKindCandidate {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.AppendRevision(c.Request.Context(), workspaceID, AppendRevisionInput{ChapterID: chapterID, ExpectedVersion: request.ExpectedVersion, Kind: request.Kind, Markdown: request.Markdown, DocumentJSON: request.DocumentJSON, ContentHash: request.ContentHash, CreatedBy: RevisionCreatorManual, LockOwnerID: request.LockOwnerID, LockVersion: request.LockVersion})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": revision})
}

func (h *Handler) RejectCandidate(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	candidateID, err := uuid.Parse(c.Param("revisionID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	var request struct {
		ExpectedVersion int       `json:"expected_version"`
		LockOwnerID     uuid.UUID `json:"lock_owner_id"`
		LockVersion     int       `json:"lock_version"`
		Reason          string    `json:"reason"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	review, err := h.service.RejectCandidate(c.Request.Context(), workspaceID, RejectCandidateInput{ChapterID: chapterID, CandidateRevisionID: candidateID, ExpectedVersion: request.ExpectedVersion, LockOwnerID: request.LockOwnerID, LockVersion: request.LockVersion, Reason: request.Reason})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": review})
}

// GetSampleGenerationPreflight estimates the exact frozen input without creating a task.
func (h *Handler) GetSampleGenerationPreflight(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	if h.taskCreator == nil {
		textbookError(c, http.StatusServiceUnavailable, "TEXTBOOK_GENERATION_UNAVAILABLE")
		return
	}
	preflight, err := h.taskCreator.PrepareSampleGeneration(c.Request.Context(), workspaceID, projectID, chapterID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": preflight})
}

// GenerateSample starts only after the client confirms the current frozen input hash.
// Clients never submit prompts or source excerpts.
func (h *Handler) GenerateSample(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	if h.taskCreator == nil {
		textbookError(c, http.StatusServiceUnavailable, "TEXTBOOK_GENERATION_UNAVAILABLE")
		return
	}
	var request struct {
		ConfirmedInputHash string `json:"confirmed_input_hash"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	task, err := h.taskCreator.CreateSampleGenerationTask(c.Request.Context(), workspaceID, projectID, chapterID, request.ConfirmedInputHash)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": gin.H{"task_id": task.ID, "status": task.Status}})
}

func textbookWorkspace(c *gin.Context) (uuid.UUID, bool) {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil {
		textbookError(c, http.StatusServiceUnavailable, "LOCAL_WORKSPACE_UNAVAILABLE")
		return uuid.Nil, false
	}
	return workspaceID, true
}

func textbookProjectIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	workspaceID, ok := textbookWorkspace(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, projectID, true
}

func textbookChapterIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	workspaceID, ok := textbookWorkspace(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	chapterID, err := uuid.Parse(c.Param("chapterID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, chapterID, true
}

func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		textbookError(c, http.StatusNotFound, "TEXTBOOK_NOT_FOUND")
	case errors.Is(err, ErrVersionConflict):
		textbookError(c, http.StatusConflict, "TEXTBOOK_VERSION_CONFLICT")
	case errors.Is(err, ErrPrimarySourceExists):
		textbookError(c, http.StatusConflict, "PRIMARY_SOURCE_EXISTS")
	case errors.Is(err, ErrRevisionLocked):
		textbookError(c, http.StatusConflict, "REVISION_LOCKED")
	case errors.Is(err, ErrClaimCoverageIncomplete):
		textbookError(c, http.StatusUnprocessableEntity, "CLAIM_COVERAGE_INCOMPLETE")
	case errors.Is(err, ErrEvidenceBudgetExceeded):
		textbookError(c, http.StatusUnprocessableEntity, "EVIDENCE_BUDGET_EXCEEDED")
	case errors.Is(err, ErrInvalidState):
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
	default:
		textbookError(c, http.StatusInternalServerError, "TEXTBOOK_OPERATION_FAILED")
	}
}

func textbookBuildIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	workspaceID, ok := textbookWorkspace(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	buildID, err := uuid.Parse(c.Param("buildID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, buildID, true
}

func textbookError(c *gin.Context, status int, code string) {
	message := map[string]string{
		"TEXTBOOK_NOT_FOUND":              "未找到教材项目或它不属于当前本地工作区。",
		"TEXTBOOK_VERSION_CONFLICT":       "章节已被更新。请刷新当前章节并检查差异后重试。",
		"PRIMARY_SOURCE_EXISTS":           "一个教材项目只能有一份主资料；请将补充资料登记为已确认的官方资料。",
		"REVISION_LOCKED":                 "章节正由另一个编辑会话处理，或你的编辑锁已过期。请重新获取锁后重试。",
		"CLAIM_COVERAGE_INCOMPLETE":       "本章尚未把关键事实逐项绑定到已选资料。请补全蓝图中的关键事实与证据后再批准或生成。",
		"EVIDENCE_BUDGET_EXCEEDED":        "已选证据过多，生成前无法保证可控的 Token 消耗。请在蓝图中保留本章必需片段，再重试。",
		"INVALID_STATE":                   "请求数据不完整或当前教材状态不允许此操作；已保留本地输入，可修正后重试。",
		"LOCAL_WORKSPACE_UNAVAILABLE":     "本地工作区尚未初始化完成。请检查数据库连接后重试。",
		"TEXTBOOK_GENERATION_UNAVAILABLE": "教材生成任务暂不可用。请检查本地任务服务后重试。",
		"TEXTBOOK_OPERATION_FAILED":       "教材操作未完成。请保留当前输入并检查本地服务日志后重试。",
	}[code]
	c.JSON(status, gin.H{"code": code, "message": message, "data": nil})
}

func decodeStrictJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}
