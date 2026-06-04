package review

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type reviewService interface {
	GetTodayCard(context.Context, uuid.UUID) (ReviewCardResponse, error)
	GetHistory(context.Context, uuid.UUID, int) (ReviewHistoryResponse, error)
	PickRandomCard(context.Context, uuid.UUID) (ReviewCardResponse, error)
	ListNotes(context.Context, uuid.UUID, ListNotesQuery) (ListNotesResponse, error)
	CreateSession(context.Context, uuid.UUID, CreateSessionRequest) (ReviewSessionResponse, error)
	GetSession(context.Context, uuid.UUID, uuid.UUID) (ReviewSessionResponse, error)
	Respond(context.Context, uuid.UUID, uuid.UUID, RespondRequest) (RespondResponse, error)
	RequestHint(context.Context, uuid.UUID, uuid.UUID) (HintResponse, error)
	Finish(context.Context, uuid.UUID, uuid.UUID) (FinishResponse, error)
}

// Handler 提供 review 领域的 HTTP 适配层。
type Handler struct {
	service reviewService
}

// NewHandler 创建 review 领域的 HTTP Handler。
func NewHandler(service reviewService) *Handler {
	return &Handler{service: service}
}

// GetTodayCard 返回今日推荐题卡。
func (h *Handler) GetTodayCard(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		return
	}

	resp, err := h.service.GetTodayCard(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// GetHistory 返回最近复习记录摘要。
func (h *Handler) GetHistory(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		return
	}

	resp, err := h.service.GetHistory(c.Request.Context(), userID, parseQueryInt(c.Query("limit"), 5))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// PickRandom 返回一次手动随机抽取的题卡。
func (h *Handler) PickRandom(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		return
	}

	resp, err := h.service.PickRandomCard(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// ListNotes 返回手动选择复习入口的候选文章列表。
func (h *Handler) ListNotes(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		return
	}

	query := ListNotesQuery{
		Query:       strings.TrimSpace(c.Query("query")),
		SeriesTitle: strings.TrimSpace(c.Query("series_title")),
		Page:        parseQueryInt(c.Query("page"), defaultListNotesPage),
		PageSize:    parseQueryInt(c.Query("page_size"), defaultListNotesPageSize),
	}

	resp, err := h.service.ListNotes(c.Request.Context(), userID, query)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// CreateSession 基于题卡创建一次 review session。
func (h *Handler) CreateSession(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeError(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	resp, err := h.service.CreateSession(c.Request.Context(), userID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// GetSession 返回一次 review session 的当前状态。
func (h *Handler) GetSession(c *gin.Context) {
	userID, sessionID, ok := h.requireSessionContext(c)
	if !ok {
		return
	}

	resp, err := h.service.GetSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// Respond 提交一轮回答并推进 review session。
func (h *Handler) Respond(c *gin.Context) {
	userID, sessionID, ok := h.requireSessionContext(c)
	if !ok {
		return
	}

	var req RespondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeError(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	resp, err := h.service.Respond(c.Request.Context(), userID, sessionID, req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// RequestHint 为当前 review session 请求一条更具体的提示。
func (h *Handler) RequestHint(c *gin.Context) {
	userID, sessionID, ok := h.requireSessionContext(c)
	if !ok {
		return
	}

	resp, err := h.service.RequestHint(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

// Finish 显式结束当前 review session。
func (h *Handler) Finish(c *gin.Context) {
	userID, sessionID, ok := h.requireSessionContext(c)
	if !ok {
		return
	}

	resp, err := h.service.Finish(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.writeSuccess(c, resp)
}

func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := raw.(uuid.UUID)
	if !ok {
		h.writeError(c, http.StatusInternalServerError, "用户 ID 类型错误")
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) requireSessionContext(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := h.getUserID(c)
	if !ok {
		if !c.Writer.Written() {
			h.writeError(c, http.StatusUnauthorized, "未授权的访问")
		}
		return uuid.Nil, uuid.Nil, false
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.writeError(c, http.StatusBadRequest, "无效的复习会话 ID")
		return uuid.Nil, uuid.Nil, false
	}

	return userID, sessionID, true
}

func (h *Handler) writeSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    data,
	})
}

func (h *Handler) writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"code":    status,
		"message": message,
		"data":    nil,
	})
}

func (h *Handler) handleServiceError(c *gin.Context, err error) {
	// Why: 领域服务已经把业务失败归类成稳定错误，handler 只负责做 HTTP 语义映射，避免把 transport 逻辑反向渗透进 service。
	switch {
	case errors.Is(err, errNoEligibleReviewNotes),
		errors.Is(err, errReviewNoteNotFound),
		errors.Is(err, errReviewSessionNotFound):
		h.writeError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, errInvalidReviewMode),
		errors.Is(err, errInvalidReviewEntry),
		errors.Is(err, errEmptyReviewAnswer):
		h.writeError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, errReviewSessionDenied):
		h.writeError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, errReviewSessionClosed),
		errors.Is(err, errReviewHintExhausted):
		h.writeError(c, http.StatusConflict, err.Error())
	default:
		h.writeError(c, http.StatusInternalServerError, err.Error())
	}
}

func parseQueryInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}
