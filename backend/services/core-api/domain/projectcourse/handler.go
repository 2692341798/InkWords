package projectcourse

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler 是 ProjectCourse 的 HTTP 适配层；不允许客户端提交事实、证据或覆盖矩阵。
type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type createRequest struct {
	RepositoryURL     string `json:"repository_url" binding:"required"`
	RequestedRef      string `json:"requested_ref" binding:"required"`
	ResolvedCommitSHA string `json:"resolved_commit_sha" binding:"required"`
	AudienceLevel     string `json:"audience_level" binding:"required"`
}

type blueprintUpdateRequest struct {
	ExpectedVersion int                  `json:"expected_version" binding:"required,min=1"`
	Chapters        []chapterUpdateInput `json:"chapters" binding:"required"`
}

type chapterUpdateInput struct {
	ChapterID string `json:"chapter_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := requestUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid project course request")
		return
	}
	if req.RepositoryURL == "" || req.RequestedRef == "" || req.ResolvedCommitSHA == "" || req.AudienceLevel == "" {
		writeError(c, http.StatusBadRequest, "required project course fields are missing")
		return
	}
	course, err := h.service.Create(c.Request.Context(), CreateInput{UserID: userID, RepositoryURL: req.RepositoryURL, RequestedRef: req.RequestedRef, ResolvedCommitSHA: req.ResolvedCommitSHA, AudienceLevel: req.AudienceLevel})
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": course})
}

func (h *Handler) Get(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	course, err := h.service.Get(c.Request.Context(), userID, courseID)
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": course})
}

func (h *Handler) UpdateBlueprint(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	var req blueprintUpdateRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "only chapter title, sort and enabled may be updated")
		return
	}
	if req.ExpectedVersion < 1 || req.Chapters == nil {
		writeError(c, http.StatusBadRequest, "expected_version and chapters are required")
		return
	}
	// 仅把允许编辑的四个字段序列化为内部更新载荷；客户端无法注入证据、事实或学习目标。
	blob, err := json.Marshal(req.Chapters)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid blueprint chapters")
		return
	}
	err = h.service.UpdateBlueprint(c.Request.Context(), userID, courseID, BlueprintUpdate{ExpectedVersion: req.ExpectedVersion, BlueprintJSON: blob, CoverageJSON: []byte(`{}`)})
	if err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"blueprint_version": req.ExpectedVersion + 1}})
}

func (h *Handler) Approve(c *gin.Context) {
	userID, courseID, ok := h.ids(c)
	if !ok {
		return
	}
	var req struct {
		ExpectedVersion int `json:"expected_version" binding:"required,min=1"`
	}
	if err := decodeStrictJSON(c, &req); err != nil || req.ExpectedVersion < 1 {
		writeError(c, http.StatusBadRequest, "expected_version is required")
		return
	}
	if err := h.service.Approve(c.Request.Context(), userID, courseID, req.ExpectedVersion); err != nil {
		h.writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"status": StatusApproved}})
}

func (h *Handler) ids(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := requestUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid project course id")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, courseID, true
}

func (h *Handler) writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(c, http.StatusNotFound, "project course not found")
	case errors.Is(err, ErrVersionConflict):
		writeError(c, http.StatusConflict, "blueprint version conflict")
	case errors.Is(err, ErrBlueprintImmutable):
		writeError(c, http.StatusConflict, "approved blueprint is immutable")
	default:
		writeError(c, http.StatusInternalServerError, "project course operation failed")
	}
}

func requestUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
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
