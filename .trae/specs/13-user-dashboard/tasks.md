# User Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a user dashboard to show token usage, cost, generated articles, word count, tech stack charts, and allow avatar/profile updates.

**Architecture:** 
- Backend: GORM model updates (`WordCount`, `TechStacks` in `Blog`), Gin routes for avatar upload, profile update, and stats aggregation. Generator service updates to extract tech stacks.
- Frontend: React component `Dashboard.tsx` with Recharts for tech stack frequency visualization.

**Tech Stack:** Go, Gin, GORM, PostgreSQL, React, Recharts, Tailwind CSS.

---

### Task 1: Update Database Models & Routes Setup

**Files:**
- Modify: `backend/internal/model/blog.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add new fields to Blog model**
Modify `backend/internal/model/blog.go` to include WordCount and TechStacks.

```go
package model

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Blog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;index:idx_user_parent_chapter;not null" json:"user_id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index:idx_user_parent_chapter" json:"parent_id"`
	ChapterSort int            `gorm:"type:integer;index:idx_user_parent_chapter" json:"chapter_sort"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	SourceType  string         `gorm:"type:varchar(50);not null" json:"source_type"`
	Status      int16          `gorm:"type:smallint;default:0" json:"status"`
	WordCount   int            `gorm:"type:integer;default:0" json:"word_count"`
	TechStacks  datatypes.JSON `gorm:"type:jsonb" json:"tech_stacks"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
```

- [ ] **Step 2: Add static file serving for avatars**
Modify `backend/cmd/server/main.go` to serve the `uploads` directory.
Add `r.Static("/uploads", "./uploads")` before the API routes.

- [ ] **Step 3: Commit**
```bash
cd backend
go get gorm.io/datatypes
cd ..
git add backend/internal/model/blog.go backend/cmd/server/main.go backend/go.mod backend/go.sum
git commit -m "feat: add WordCount and TechStacks to Blog model and serve uploads"
```

### Task 2: User Profile & Avatar API

**Files:**
- Create: `backend/internal/api/profile.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Implement Profile APIs**
Create `backend/internal/api/profile.go` with `UpdateProfile` and `UploadAvatar` handlers. Handle file upload to `uploads/avatars/` and update `avatar_url` and `username`.

- [ ] **Step 2: Register routes in main.go**
Add `PUT /api/user/profile` and `POST /api/user/avatar` to the protected route group.

- [ ] **Step 3: Commit**
```bash
git add backend/internal/api/profile.go backend/cmd/server/main.go
git commit -m "feat: implement user profile update and avatar upload APIs"
```

### Task 3: Implement User Stats API

**Files:**
- Create: `backend/internal/api/stats.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Implement Stats API**
Create `backend/internal/api/stats.go` to calculate `tokens_used`, `estimated_cost` (tokens / 1,000,000), `total_articles`, `total_words`, and aggregate `TechStacks` from the `Blog` model.

- [ ] **Step 2: Register Stats route**
Modify `backend/cmd/server/main.go` to add `GET /api/user/stats`.

- [ ] **Step 3: Commit**
```bash
git add backend/internal/api/stats.go backend/cmd/server/main.go
git commit -m "feat: implement user stats API for dashboard"
```

### Task 4: Update Generator Service for TechStacks

**Files:**
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/llm/deepseek.go`

- [ ] **Step 1: Update Generator to Calculate Word Count**
In `generator.go`, after the blog is fully generated, calculate the length of the generated Markdown text (e.g. `len([]rune(content))`) and update the `WordCount` of the blog.

- [ ] **Step 2: Use LLM to Extract Tech Stacks**
Add a lightweight LLM call to extract an array of tech stacks.
```go
// Add to DeepSeek client or call directly in generator
prompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符：\n\n" + content
// Call LLM, parse JSON response into a string slice, then marshal to datatypes.JSON
```

- [ ] **Step 3: Save to Database**
Save the `WordCount` and `TechStacks` to the database. Update the user's `TokensUsed`.

- [ ] **Step 4: Commit**
```bash
git add backend/internal/service/generator.go backend/internal/llm/deepseek.go
git commit -m "feat: extract tech stacks and calculate word count after generation"
```

### Task 5: Frontend Recharts & Dashboard Component

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/src/components/Dashboard.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Install Recharts**
```bash
cd frontend
npm install recharts lucide-react
```

- [ ] **Step 2: Create Dashboard Component**
Create `frontend/src/components/Dashboard.tsx` with Recharts BarChart, data cards (Tokens, Cost, Articles, Words), and an Avatar/Profile edit section.

- [ ] **Step 3: Add to Router & Sidebar**
Modify `frontend/src/App.tsx` to include the route. Modify `Sidebar.tsx` to link to it.

- [ ] **Step 4: Commit**
```bash
git add frontend/package.json frontend/package-lock.json frontend/src/components/Dashboard.tsx frontend/src/components/Sidebar.tsx frontend/src/App.tsx
git commit -m "feat: add user dashboard UI with recharts and profile editing"
```

### Task 6: Document Updates (Required by Workspace Rules)

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: Update API & Database docs**
Add `/api/user/stats`, `/api/user/profile`, `/api/user/avatar` to `InkWords_API.md`.
Update `Blog` model in `InkWords_Database.md` to include `word_count` and `tech_stacks`.

- [ ] **Step 2: Update PRD & Arch & Logs**
Update PRD to include Dashboard features.
Record this development log in `InkWords_Development_Plan_and_Log.md`.

- [ ] **Step 3: Commit**
```bash
git add .trae/documents/*.md README.md
git commit -m "docs: update core documents for user dashboard feature"
```
