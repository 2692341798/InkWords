package stream

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
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
	Generate(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	Continue(context.Context, uuid.UUID, uuid.UUID, chan<- string, chan<- error)
	Polish(context.Context, PolishRequest, chan<- string, chan<- error)
	AnalyzeStream(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	ScanProjectModules(context.Context, string, chan<- string) ([]ModuleCard, error)
}

func NewHandler(service streamService, blogRepo BlogReadable) *Handler {
	return &Handler{service: service, blogRepo: blogRepo}
}

func (h *Handler) getWorkspaceID(c *gin.Context) uuid.UUID {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil {
		return uuid.Nil
	}
	return workspaceID
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
