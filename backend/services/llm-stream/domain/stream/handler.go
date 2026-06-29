package stream

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type streamOperation string

const (
	streamOperationGenerate streamOperation = "generate"
	streamOperationContinue streamOperation = "continue"
	streamOperationPolish   streamOperation = "polish"
	streamOperationAnalyze  streamOperation = "analyze"
	streamOperationScan     streamOperation = "scan"
	streamChannelBufferSize                 = 128
)

type Handler struct {
	service  streamService
	blogRepo BlogReadable
}

type streamService interface {
	CheckQuota(uuid.UUID) error
	Generate(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	Continue(context.Context, uuid.UUID, uuid.UUID, chan<- string, chan<- error)
	Polish(context.Context, PolishRequest, chan<- string, chan<- error)
	AnalyzeStream(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	ScanProjectModules(context.Context, string, chan<- string) ([]ModuleCard, error)
}

func NewHandler(service streamService, blogRepo BlogReadable) *Handler {
	return &Handler{service: service, blogRepo: blogRepo}
}

func (h *Handler) getUserID(c *gin.Context) uuid.UUID {
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

func (h *Handler) maybeCheckQuota(c *gin.Context) bool {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			if err := h.service.CheckQuota(uid); err != nil {
				c.JSON(http.StatusPaymentRequired, gin.H{"error": "quota exceeded"})
				return false
			}
		}
	}
	return true
}

func externalStreamErrorMessage(operation streamOperation, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "request canceled"
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(err.Error()), "blog not found") {
		return "blog not found"
	}

	switch operation {
	case streamOperationContinue:
		return "blog continuation failed"
	case streamOperationPolish:
		return "blog polish failed"
	case streamOperationAnalyze:
		return "blog analysis failed"
	case streamOperationScan:
		return "project scan failed"
	default:
		return "blog generation failed"
	}
}
