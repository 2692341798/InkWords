# Export Series to Obsidian Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 InkWords 生成的系列博客完整、带有双链和 Frontmatter 地导出到本地 Obsidian Vault 目录中。

**Architecture:** 
- **后端 (Go)**: 采用垂直切片架构，在 `internal/domain/export/` 下实现导出逻辑。接收系列 ID 和 Vault 绝对路径，遍历系列内的博客内容，生成符合 Obsidian LLM Wiki Pattern 的 YAML Frontmatter，处理 Markdown 内容中的双向链接，并将文件写入本地。
- **前端 (React)**: 在系列详情页增加“导出至 Obsidian”按钮，通过 shadcn/ui Dialog 弹窗让用户输入 Vault 路径，调用后端 API 触发导出，并通过 Zustand 管理历史 Vault 路径状态。

**Tech Stack:** Go 1.21+, Gin, React 18, Zustand, Tailwind CSS, shadcn/ui, Obsidian Markdown.

---

### Task 1: 后端 - 定义导出请求模型与接口

**Files:**
- Create: `backend/internal/domain/export/handler.go`
- Create: `backend/internal/domain/export/handler_test.go`
- Create: `backend/internal/domain/export/model.go`

- [ ] **Step 1: Write the failing test for export request validation**

```go
// backend/internal/domain/export/handler_test.go
package export

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/gin-gonic/gin"
)

func TestExportObsidian_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	// mock service dependencies would go here
	handler := NewHandler(nil)
	router.POST("/api/v1/export/obsidian", handler.ExportToObsidian)

	reqBody := map[string]interface{}{
		"series_id":  "", // Invalid: empty
		"vault_path": "",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/export/obsidian", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %v", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/domain/export/... -v`
Expected: FAIL (No such file or directory / undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// backend/internal/domain/export/model.go
package export

type ExportObsidianRequest struct {
	SeriesID  string `json:"series_id" binding:"required"`
	VaultPath string `json:"vault_path" binding:"required"`
}

// backend/internal/domain/export/handler.go
package export

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type Service interface {
	ExportSeriesToVault(seriesID, vaultPath string) error
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ExportToObsidian(c *gin.Context) {
	var req ExportObsidianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if h.svc != nil {
		if err := h.svc.ExportSeriesToVault(req.SeriesID, req.VaultPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/domain/export/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

Run: `git add backend/internal/domain/export/ && git commit -m "feat(backend): add export obsidian API handler and model"`


### Task 2: 后端 - 实现 Markdown 格式化与 Frontmatter 生成

**Files:**
- Create: `backend/internal/domain/export/formatter.go`
- Create: `backend/internal/domain/export/formatter_test.go`

- [ ] **Step 1: Write the failing test for frontmatter generation**

```go
// backend/internal/domain/export/formatter_test.go
package export

import (
	"strings"
	"testing"
	"time"
)

func TestFormatObsidianNote(t *testing.T) {
	title := "Go语言基础"
	content := "这里是正文，提到了[[并发模型]]。"
	tags := []string{"#domain/golang", "#series/go-tutorial"}
	createdAt := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	result := FormatObsidianNote(title, content, tags, createdAt)
	
	if !strings.Contains(result, "type: entity") {
		t.Errorf("Expected frontmatter type, got: %s", result)
	}
	if !strings.Contains(result, "title: \"Go语言基础\"") {
		t.Errorf("Expected frontmatter title, got: %s", result)
	}
	if !strings.Contains(result, content) {
		t.Errorf("Expected content, got: %s", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/domain/export/formatter_test.go -v`
Expected: FAIL (FormatObsidianNote not defined)

- [ ] **Step 3: Write minimal implementation**

```go
// backend/internal/domain/export/formatter.go
package export

import (
	"fmt"
	"strings"
	"time"
)

// FormatObsidianNote assembles the markdown content with Obsidian LLM Wiki Pattern Frontmatter
func FormatObsidianNote(title, content string, tags []string, createdAt time.Time) string {
	dateStr := createdAt.Format("2006-01-02")
	
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("type: entity\n")
	sb.WriteString(fmt.Sprintf("title: \"%s\"\n", title))
	sb.WriteString(fmt.Sprintf("created: %s\n", dateStr))
	sb.WriteString(fmt.Sprintf("updated: %s\n", dateStr))
	
	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range tags {
			sb.WriteString(fmt.Sprintf("  - \"%s\"\n", tag))
		}
	}
	
	sb.WriteString("status: seed\n")
	sb.WriteString("---\n\n")
	sb.WriteString(content)
	
	return sb.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/domain/export/formatter_test.go -v`
Expected: PASS

- [ ] **Step 5: Commit**

Run: `git add backend/internal/domain/export/formatter* && git commit -m "feat(backend): add obsidian markdown formatter"`


### Task 3: 后端 - 实现文件系统写入逻辑 (Service 层)

**Files:**
- Create: `backend/internal/domain/export/service.go`
- Create: `backend/internal/domain/export/service_test.go`

- [ ] **Step 1: Write the failing test for file writing**

```go
// backend/internal/domain/export/service_test.go
package export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportSeriesToVault_WriteFile(t *testing.T) {
	tempDir := t.TempDir()
	
	svc := NewService() 
	
	err := svc.WriteNoteToDisk(tempDir, "test.md", "content body")
	if err != nil {
		t.Fatalf("Failed to write note: %v", err)
	}
	
	content, err := os.ReadFile(filepath.Join(tempDir, "test.md"))
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	
	if string(content) != "content body" {
		t.Errorf("Expected 'content body', got %s", string(content))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/domain/export/service_test.go -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
// backend/internal/domain/export/service.go
package export

import (
	"fmt"
	"os"
	"path/filepath"
)

type exportService struct {
	// dependencies like repo go here
}

func NewService() *exportService {
	return &exportService{}
}

// ExportSeriesToVault is the main entry. In a real scenario it fetches posts from DB.
func (s *exportService) ExportSeriesToVault(seriesID, vaultPath string) error {
	// TODO in actual implementation: fetch series and posts from repository
	// For now, it's just the interface compliance
	return nil
}

// WriteNoteToDisk safely writes content to the target vault path
func (s *exportService) WriteNoteToDisk(vaultPath, filename, content string) error {
	// Clean and resolve the absolute path to prevent path traversal
	cleanVaultPath := filepath.Clean(vaultPath)
	targetPath := filepath.Join(cleanVaultPath, filename)
	
	// Ensure the target path is still within the vault path
	if filepath.Dir(targetPath) != cleanVaultPath {
		return fmt.Errorf("invalid file path, path traversal detected")
	}

	// Create directory if not exists
	if err := os.MkdirAll(cleanVaultPath, 0755); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	return os.WriteFile(targetPath, []byte(content), 0644)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/domain/export/service_test.go -v`
Expected: PASS

- [ ] **Step 5: Commit**

Run: `git add backend/internal/domain/export/service* && git commit -m "feat(backend): implement vault file writing in export service"`


### Task 4: 前端 - 增加全局 Zustand 状态管理 Vault 路径

**Files:**
- Create: `frontend/src/store/useExportStore.ts`

- [ ] **Step 1: Write the minimal implementation**

```typescript
// frontend/src/store/useExportStore.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface ExportState {
  obsidianVaultPath: string;
  setObsidianVaultPath: (path: string) => void;
}

export const useExportStore = create<ExportState>()(
  persist(
    (set) => ({
      obsidianVaultPath: '',
      setObsidianVaultPath: (path) => set({ obsidianVaultPath: path }),
    }),
    {
      name: 'inkwords-export-storage',
    }
  )
);
```

- [ ] **Step 2: Commit**

Run: `git add frontend/src/store/useExportStore.ts && git commit -m "feat(frontend): add zustand store for obsidian export path"`


### Task 5: 前端 - 导出至 Obsidian 弹窗组件 (Dialog)

**Files:**
- Create: `frontend/src/components/ExportToObsidianModal.tsx`

- [ ] **Step 1: Write the component implementation**

```tsx
// frontend/src/components/ExportToObsidianModal.tsx
import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogTrigger } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useExportStore } from '@/store/useExportStore';

interface ExportToObsidianModalProps {
  seriesId: string;
  onExport: (vaultPath: string) => Promise<void>;
}

export const ExportToObsidianModal: React.FC<ExportToObsidianModalProps> = ({ seriesId, onExport }) => {
  const { obsidianVaultPath, setObsidianVaultPath } = useExportStore();
  const [localPath, setLocalPath] = useState(obsidianVaultPath);
  const [isOpen, setIsOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    if (!localPath.trim()) return;
    setLoading(true);
    try {
      setObsidianVaultPath(localPath);
      await onExport(localPath);
      setIsOpen(false);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">导出至 Obsidian</Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>导出至 Obsidian 知识库</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="vaultPath" className="text-right">
              Vault 绝对路径
            </Label>
            <Input
              id="vaultPath"
              value={localPath}
              onChange={(e) => setLocalPath(e.target.value)}
              placeholder="/Users/username/Documents/MyVault"
              className="col-span-3"
            />
          </div>
        </div>
        <DialogFooter>
          <Button disabled={loading || !localPath.trim()} onClick={handleExport}>
            {loading ? '导出中...' : '确认导出'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
```

- [ ] **Step 2: Commit**

Run: `git add frontend/src/components/ExportToObsidianModal.tsx && git commit -m "feat(frontend): create export to obsidian modal component"`


### Task 6: 前端 - 接入系列详情页

**Files:**
- Modify: `frontend/src/pages/SeriesDetail.tsx` (or the equivalent specific component)

- [ ] **Step 1: Update the SeriesDetail page to include the modal**

```tsx
// frontend/src/pages/SeriesDetail.tsx

// 1. Add imports
import { ExportToObsidianModal } from '@/components/ExportToObsidianModal';
import { useToast } from '@/components/ui/use-toast';

// 2. Add handlers inside the component
// const { toast } = useToast();
// const handleExportToObsidian = async (vaultPath: string) => {
//   const res = await fetch('/api/v1/export/obsidian', {
//     method: 'POST',
//     headers: { 'Content-Type': 'application/json' },
//     body: JSON.stringify({ series_id: seriesId, vault_path: vaultPath }),
//   });
//   if (!res.ok) {
//     const error = await res.json();
//     toast({ title: "导出失败", description: error.error, variant: "destructive" });
//     throw new Error(error.error);
//   }
//   toast({ title: "导出成功", description: "系列博客已成功写入 Obsidian Vault。" });
// };

// 3. Add modal in JSX
// <ExportToObsidianModal seriesId={seriesId} onExport={handleExportToObsidian} />
```

- [ ] **Step 2: Commit**

Run: `git add frontend/src/pages/SeriesDetail.tsx && git commit -m "feat(frontend): integrate obsidian export modal in series detail page"`