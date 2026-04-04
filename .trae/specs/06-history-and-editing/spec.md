# 历史记录与编辑 Spec

## Why
在完成单篇生成和系列博客的大纲生成与保存后，当前系统缺乏让用户查看历史生成记录和进行二次编辑的界面。为了提供完整的“创作-保存-修改-导出”体验，我们需要开发历史记录功能及类似 Notion 的 Markdown 编辑器，支持自动保存和文件导出。

## What Changes
- 后端增加拉取博客历史记录接口，支持按系列展示。
- 后端增加更新单篇博客内容接口。
- 前端侧边栏新增历史记录拉取与展示组件。
- 前端右侧新增 Markdown 二次编辑器组件。
- 前端实现自动保存（Debounce）更新文章机制。
- 前端实现文章导出（MD/PDF）功能。

## Impact
- Affected specs: 05-project-decomposition, 01-init-foundation
- Affected code: `backend/internal/api/blog.go`, `frontend/src/store/streamStore.ts`, `frontend/src/components/Sidebar.tsx`, `frontend/src/components/Editor.tsx`

## ADDED Requirements
### Requirement: 历史记录展示
系统应能在左侧侧边栏展示用户历史生成的博客记录，并支持树状结构显示（系列博客的父子层级）。

#### Scenario: 成功加载历史记录
- **WHEN** 用户登录并进入工作台
- **THEN** 系统自动拉取历史记录并在侧边栏展示

### Requirement: Markdown 二次编辑与自动保存
系统应提供一个 Markdown 编辑器，支持对生成的博客进行二次编辑。

#### Scenario: 编辑并自动保存
- **WHEN** 用户在编辑器中修改内容，并在 2 秒内无新输入
- **THEN** 系统自动调用后端更新接口，保存最新内容

### Requirement: 导出功能
用户可以将当前浏览或编辑的博客导出为 Markdown 或 PDF 文件。

#### Scenario: 导出 Markdown
- **WHEN** 用户点击“导出为 MD”按钮
- **THEN** 浏览器触发下载对应博客内容的 .md 文件