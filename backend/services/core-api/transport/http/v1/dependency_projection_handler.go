package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"inkwords-backend/services/core-api/app/textbookartifact"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	"inkwords-backend/shared/kernel/httpx"
)

type dependencyOptionReader interface {
	Options(context.Context, uuid.UUID, uuid.UUID) ([]textbookartifact.DependencyOption, error)
}
type dependencyProjectionActions interface {
	Preview(context.Context, uuid.UUID, uuid.UUID, textbookartifact.DependencyProjectionRequest) (textbookartifact.DependencyProjectionPreview, error)
	Apply(context.Context, uuid.UUID, uuid.UUID, textbookartifact.DependencyProjectionRequest, string) (*textbookdomain.CodeArtifactRow, error)
}

// DependencyProjectionHandler only accepts references to saved candidates and configured packages.
type DependencyProjectionHandler struct {
	catalog dependencyOptionReader
	service dependencyProjectionActions
	slots   chan struct{}
}

// NewDependencyProjectionHandler wires the local dependency preview/apply surface.
func NewDependencyProjectionHandler(catalog dependencyOptionReader, service dependencyProjectionActions) *DependencyProjectionHandler {
	return &DependencyProjectionHandler{catalog: catalog, service: service, slots: make(chan struct{}, 1)}
}

// RegisterDependencyProjectionRoutes requires the same local workspace boundary as other textbook actions.
func RegisterDependencyProjectionRoutes(r *gin.Engine, workspaceMiddleware gin.HandlerFunc, h *DependencyProjectionHandler) {
	if workspaceMiddleware == nil || h == nil {
		panic("dependency projection routes require workspace middleware and handler")
	}
	group := r.Group("/api/v1/textbook-projects/chapters/:chapterID/dependency-projection", workspaceMiddleware, h.bound)
	group.GET("/options", h.Options)
	group.POST("/preview", h.Preview)
	group.POST("/apply", h.Apply)
}

func (h *DependencyProjectionHandler) bound(c *gin.Context) {
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
		c.Next()
	default:
		dependencyHTTPError(c, http.StatusTooManyRequests, "DEPENDENCY_BUSY", "已有依赖核对正在进行，请稍后再试。")
		c.Abort()
	}
}

func (h *DependencyProjectionHandler) identities(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	w, err := httpx.GetLocalWorkspaceID(c)
	if err != nil || h == nil || h.catalog == nil || h.service == nil {
		dependencyHTTPError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "本地依赖准备服务尚未就绪。")
		return uuid.Nil, uuid.Nil, false
	}
	ch, err := uuid.Parse(c.Param("chapterID"))
	if err != nil || ch == uuid.Nil {
		dependencyHTTPError(c, http.StatusBadRequest, "INVALID_STATE", "章节标识无效。")
		return uuid.Nil, uuid.Nil, false
	}
	return w, ch, true
}

// Options does not activate a package or create any artifact.
func (h *DependencyProjectionHandler) Options(c *gin.Context) {
	w, ch, ok := h.identities(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	options, err := h.catalog.Options(ctx, w, ch)
	if err != nil {
		h.failure(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": options})
}

// Preview returns the exact summary the user must confirm before registration.
func (h *DependencyProjectionHandler) Preview(c *gin.Context) { h.project(c, false) }

// Apply registers the confirmed projection as unverified; execution is a separate action.
func (h *DependencyProjectionHandler) Apply(c *gin.Context) { h.project(c, true) }

func (h *DependencyProjectionHandler) project(c *gin.Context, apply bool) {
	w, ch, ok := h.identities(c)
	if !ok {
		return
	}
	var request struct {
		textbookartifact.DependencyProjectionRequest
		ConfirmedPreviewHash string `json:"confirmed_preview_hash"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF {
		dependencyHTTPError(c, http.StatusBadRequest, "INVALID_STATE", "仅接受已保存的候选、依赖选项与确认指纹。")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if apply {
		row, err := h.service.Apply(ctx, w, ch, request.DependencyProjectionRequest, request.ConfirmedPreviewHash)
		if err != nil {
			h.failure(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": row})
		return
	}
	preview, err := h.service.Preview(ctx, w, ch, request.DependencyProjectionRequest)
	if err != nil {
		h.failure(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": preview})
}

func (h *DependencyProjectionHandler) failure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, textbookartifact.ErrDependencyConfirmationChanged):
		dependencyHTTPError(c, http.StatusConflict, "DEPENDENCY_CONFIRMATION_CHANGED", "预览已变化或尚未确认。请重新预览后再登记。")
	case errors.Is(err, textbookdomain.ErrNotFound):
		dependencyHTTPError(c, http.StatusNotFound, "DEPENDENCY_NOT_FOUND", "当前章节没有这个候选或依赖选项。")
	case errors.Is(err, textbookdomain.ErrInvalidState):
		dependencyHTTPError(c, http.StatusConflict, "INVALID_STATE", "候选或来源身份不匹配。请刷新章节后重新预览。")
	default:
		dependencyHTTPError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "依赖文件、来源或工件登记未通过核对。请检查本地准备记录，原候选保持不变。")
	}
}

func dependencyHTTPError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"code": code, "message": message})
}
