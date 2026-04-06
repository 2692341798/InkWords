# [Generator UI and Prompt Fixes] Spec

## Why
在博客生成过程中，用户遇到了 4 个问题：
1. 并发 Worker 数量增加后，前端 Worker 卡片由于硬编码的固定布局导致比例挤压变形。
2. 生成过程中的步骤 2 和步骤 3 名称存在重复或相似的文本（"评估大模型并生成项目大纲..."），导致界面混淆。
3. 单篇生成的博客正文中，Git 仓库链接因为未作为 Prompt 的一部分输入，被大模型生成为了占位符 `<你的墨言项目仓库地址>`。
4. 第一篇博客结尾出现了与下一篇无关的预告，因为大模型不知道后续章节的标题。

## What Changes
- 修改 `Generator.tsx` 中 Worker 卡片的渲染逻辑，不再硬编码 `[0,1,2,3,4]` 而是遍历 `Object.keys(store.workers)`，并采用网格自适应布局。
- 修改 `Generator.tsx` 中 Step 2 的占位文本，由 "评估大模型并生成项目大纲..." 改为 "并发分析代码分块..."。
- 修改 `decomposition.go` 中 `GenerateSeries` 的 Prompt，将 `gitURL` 注入到上下文中。
- 修改 `decomposition.go` 中 `GenerateSeries` 的 Prompt，附加下一章节的大纲信息，引导正确的预告。

## Impact
- Affected specs: 无
- Affected code: `frontend/src/components/Generator.tsx`, `backend/internal/service/decomposition.go`

## MODIFIED Requirements
### Requirement: Generator UI and Prompting
The system SHALL provide adaptive worker cards, distinct step names, real git URLs in generated content, and accurate next-chapter previews.

#### Scenario: Success case
- **WHEN** user initiates a project analysis with up to 20 workers
- **THEN** the UI neatly wraps the cards without squashing them
- **WHEN** analysis progresses from step 2 to 3
- **THEN** step 2 says "并发分析代码分块..." and step 3 says "生成项目全局大纲..."
- **WHEN** the blog is generated
- **THEN** it contains the real GitHub URL instead of a placeholder, and a preview matching the next chapter title.
