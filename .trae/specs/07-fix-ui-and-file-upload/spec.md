# Fix UI and File Upload Spec

## Why
目前项目在前端交互和基础体验上存在几个关键问题，严重影响了核心功能的可用性：
1. 缺少登录/注册页面，用户无法使用已开发的后端 OAuth 接口进行登录，只能看到工作台。
2. 不支持文档上传（PDF、Word、Markdown），违背了 PRD 中“小文件/单文档生成流程”的核心功能。
3. GitHub 仓库分析过程缓慢且状态不清晰，用户无法判断系统是否卡死。
4. 侧边栏生成的系列博客卡片无法点击进行预览或编辑。
5. Markdown 编辑器存在状态同步 Bug，导致用户在输入过程中内容被自动保存覆盖，造成光标跳动和输入丢失，无法正常编辑文档。

## What Changes
- **新增**：前端登录页面/模态框，对接后端已有的 GitHub/WeChat OAuth2 接口。
- **新增**：在 `Generator.tsx` 中增加文件拖拽/点击上传区域，支持 PDF、Word、Markdown 格式文件。
- **改进**：优化仓库分析的 Loading 状态，增加更详细的阶段提示或进度条。
- **修复**：使 `Sidebar.tsx` 中正在生成/已生成的博客章节卡片可以被点击（点击后在右侧 `Editor` 中打开对应的博客节点）。
- **修复**：修复 `Editor.tsx` 中 `useEffect` 引起的状态互相覆盖问题，确保 Markdown 编辑顺畅无阻。

## Impact
- Affected specs: 登录认证流程、文件上传解析流程、流式生成交互体验。
- Affected code:
  - `frontend/src/components/Generator.tsx`
  - `frontend/src/components/Sidebar.tsx`
  - `frontend/src/components/Editor.tsx`
  - `frontend/src/App.tsx`
  - `frontend/src/store/streamStore.ts`
  - `frontend/src/store/blogStore.ts`

## ADDED Requirements
### Requirement: 登录注册支持
系统需提供用户登录/注册界面，必须登录后才能进入主工作台。
#### Scenario: 强制登录拦截
- **WHEN** 用户未登录（无有效 Token）访问页面时
- **THEN** 显示全屏的登录组件，提供 GitHub 等第三方授权按钮。

### Requirement: 文件上传支持
用户能够上传 PDF/Word/Markdown 文件以生成单篇博客。
#### Scenario: 上传文件并生成
- **WHEN** 用户在 Generator 页面拖拽或选择支持的文件并上传
- **THEN** 后端接收文件并解析文本，前端直接进入流式生成单篇博客的状态。

## MODIFIED Requirements
### Requirement: 侧边栏卡片可点击与编辑器修复
正在生成或已生成的任务卡片需要能够点击以进入二次编辑界面；编辑器需要支持流畅输入。
#### Scenario: 点击任务卡片
- **WHEN** 用户点击侧边栏的生成任务卡片
- **THEN** 右侧编辑器区域加载该卡片对应的数据，且用户可以自由编辑（不因自动保存而打断输入）。
