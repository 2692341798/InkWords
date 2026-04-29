# 系列博客批量导出 PDF（合并版）设计

## 1. 背景与目标

InkWords 已支持：
- 系列博客导出 ZIP（Markdown 打包，后端：`GET /api/v1/blogs/:id/export`）
- 侧边栏批量导出 ZIP（前端 JSZip 打包）
- 单篇/系列同步到 Obsidian（后端写入挂载目录）

本需求新增：支持在侧边栏批量模式中，对“系列父节点”进行 **合并导出 PDF**。

### 目标（DoD）
- 用户在侧边栏批量模式中勾选 1+ 个系列父节点后，点击「导出 PDF」。
- 每个系列生成 1 个 PDF（封面 + 目录 + 正文），浏览器逐个触发下载（多次下载）。
- 生成过程需要鉴权；失败时给出中文提示，不影响后续系列的下载尝试。

### 非目标（Not in scope）
- 在目录页显示页码（先不做）。
- 生成单篇 PDF（可后续复用同一能力扩展）。
- PDF 内代码高亮、图片下载代理、复杂排版模板系统（先做最小可用版本）。

## 2. 方案选择

选择 **后端 Headless Chromium 打印 HTML → PDF**：
- 前端无法可靠自动“保存为 PDF”，且批量体验差。
- 后端生成 PDF 后通过 HTTP 直接返回二进制，前端可以标准化触发下载。
- 版式更接近预览（HTML/CSS），后续可迭代样式与主题。

## 3. 用户交互（B 方案）

入口：`frontend/src/components/Sidebar.tsx` 批量操作栏新增按钮「导出 PDF」。

行为：
- 批量模式下，用户勾选多个系列父节点。
- 点击「导出 PDF」后，按顺序逐个请求后端接口并触发下载：
  - 对每个系列：下载 1 个 PDF
  - 过程中展示 loading / 进度 toast（例如：`正在导出：2/5`）
  - 某个系列失败：toast 提示失败原因，继续处理下一个系列

## 4. API 设计

### 4.1 新增接口

`GET /api/v1/blogs/:id/export/pdf`

说明：
- `:id` 为系列父节点 blog_id（与现有 zip 导出接口对齐）
- 复用 JWT 鉴权中间件

响应：
- `200`：`Content-Type: application/pdf`
- `Content-Disposition: attachment; filename="<系列标题>.pdf"`
- `404`：系列不存在或无权限
- `500`：生成失败（返回 JSON 错误，前端提示）

### 4.2 兼容性

不修改既有接口；新增一个独立路径，避免与 ZIP 行为耦合。

## 5. 后端实现设计

### 5.1 代码位置（最小改动）
- 路由注册：`backend/cmd/server/main.go`
  - `blogGroup.GET("/:id/export/pdf", blogAPI.ExportSeriesPDF)`
- Handler：`backend/internal/api/blog.go`
  - 新增 `ExportSeriesPDF` 方法，模式参考 `ExportSeries`
- Service：`backend/internal/service/` 新增 `pdf_export.go`
  - 新增 `ExportSeriesToPDF(ctx, parentID, userID) (pdfPath string, filename string, err error)` 或 `([]byte, filename, err)`（优先以临时文件方式避免内存峰值）

### 5.2 数据获取

复用 `GetSeriesBlogs(ctx, parentID, userID)`：
- `blogs[0]` 为父节点（系列导读/概览）
- `blogs[1:]` 为子章节（按 `chapter_sort ASC`）

### 5.3 Markdown → HTML

引入 `github.com/yuin/goldmark`（GFM 扩展）将 Markdown 渲染为 HTML。

约束：
- 输出 HTML 需要进行基本的安全处理（只允许后端生成内容，不接收用户传入 HTML）。
- Mermaid 不在服务端渲染为图（保持代码块样式即可），避免引入额外 headless 渲染复杂度。

### 5.4 HTML 模板（封面 + 目录 + 正文）

HTML 结构建议：
- `cover`：系列标题、导出时间
- `toc`：列出章节（含导读/概览与各章节），可做锚点跳转（不含页码）
- `content`：每篇文章一章，章节间 `page-break-before: always`

排版：
- A4 输出、合理页边距（CSS `@page` + `margin`）
- 使用基础字体（需确保容器内有中文字体）

### 5.5 HTML → PDF（Chromium）

使用容器内 Chromium 执行：
- 写入临时 HTML 文件（`os.CreateTemp`）
- 执行命令：
  - `chromium --headless --disable-gpu --no-sandbox --print-to-pdf=<tmp.pdf> <tmp.html>`
- 完成后：
  - API 层通过 `c.FileAttachment(tmp.pdf, filename)` 发送
  - `defer` 删除临时文件（HTML/PDF）

并发控制：
- 单次请求只生成一个系列 PDF。
- 前端批量为顺序调用；后端无需额外 goroutine 池。
- 如后续要支持并发批量导出，可在 service 内引入 semaphore 限制同时生成数。

## 6. 容器与部署改动

### 6.1 后端镜像（runtime stage）

需要安装：
- `chromium`
- 中文字体（如 `font-noto-cjk`）及基础字体依赖
- Chromium 运行常见依赖（`nss`, `freetype`, `harfbuzz` 等）

修改文件：
- [backend/Dockerfile](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/backend/Dockerfile)

## 7. 前端实现设计

修改文件：
- [Sidebar.tsx](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/frontend/src/components/Sidebar.tsx)

新增：
- 批量操作栏按钮「导出 PDF」
- 处理函数 `handleBatchExportSeriesPDF`
  - 基于 `selectedSeriesRoots`（已存在）循环调用接口
  - 每次下载：`const blob = await res.blob()` → `URL.createObjectURL` → `<a download=...>`
  - Toast：开始/进度/成功/失败（中文）

## 8. 错误处理

后端：
- 系列不存在/无权限：返回 404 JSON（与现有错误风格一致）
- Chromium 缺失/执行失败：返回 500 JSON，message 不泄漏内部路径

前端：
- 单个系列失败：提示“某系列导出失败：原因”，继续导出下一个
- 全部成功：提示“已开始下载 N 份 PDF”

## 9. 验证计划（可复现）

后端：
- `go test ./...`（至少覆盖新增的渲染/命令执行封装的单元测试）
- 通过 docker 运行后，调用接口检查响应头：
  - `Content-Type: application/pdf`
  - `Content-Disposition` filename 正确

端到端：
- `docker compose down && docker compose up -d --build`
- 访问 `http://localhost`：
  - 生成或选中已有系列
  - 侧边栏进入批量模式，勾选 2 个系列父节点
  - 点击「导出 PDF」确认浏览器触发多次下载

## 10. 后续扩展
- 目录页页码（需要评估 Chromium 对 `target-counter(page)` 的支持或引入 paged media 引擎）
- “多系列打包 ZIP（PDF）”作为可选模式
- 单篇博客导出 PDF（复用同一 HTML 模板与渲染链路）

