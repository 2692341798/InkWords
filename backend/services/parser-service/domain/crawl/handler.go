package crawl

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// crawlService keeps the HTTP adapter independent of concrete transport or storage.
type crawlService interface {
	Crawl(context.Context, Policy) (Manifest, error)
	Resume(context.Context, string) (Manifest, error)
}

// Handler exposes a bounded preview of an official documentation stack. It returns
// only the manifest; creating source snapshots is intentionally a later core-owned step.
type Handler struct{ service crawlService }

func NewHandler(service crawlService) *Handler { return &Handler{service: service} }

func (handler *Handler) Crawl(c *gin.Context) {
	var request struct {
		EntryURL            string   `json:"entry_url"`
		AllowedPathPrefixes []string `json:"allowed_path_prefixes"`
		CheckpointID        string   `json:"checkpoint_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_CRAWL_POLICY", "message": "抓取入口必须是有效 JSON"})
		return
	}
	if checkpointID := strings.TrimSpace(request.CheckpointID); checkpointID != "" {
		manifest, err := handler.service.Resume(c.Request.Context(), checkpointID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "CRAWL_CHECKPOINT_REJECTED", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": manifest})
		return
	}
	policy, err := DefaultPolicy(strings.TrimSpace(request.EntryURL))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_CRAWL_POLICY", "message": "抓取入口不符合安全边界"})
		return
	}
	if len(request.AllowedPathPrefixes) > 0 {
		policy.AllowedPathPrefixes = request.AllowedPathPrefixes
	}
	if err := policy.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_CRAWL_POLICY", "message": "抓取边界不符合安全要求"})
		return
	}
	manifest, err := handler.service.Crawl(c.Request.Context(), policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "CRAWL_REJECTED", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": manifest})
}
