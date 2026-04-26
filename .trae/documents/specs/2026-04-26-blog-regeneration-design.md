# 博客再生功能优化设计文档

## 1. 目标与背景
目前在 InkWords 墨言博客助手中，如果解析一个已解析过的 GitHub 仓库（且源码有更新），系统虽然能够对比出哪些章节需要 `skip`、哪些需要 `regenerate`，但在处理 `regenerate` 章节时，LLM 会完全从头生成新的博客内容，没有利用该章节旧版本的内容作为上下文。这不仅浪费了 Token 资源，还可能导致旧版本中沉淀的优秀解释或行文风格丢失。
本设计的目标是在 `regenerate` 阶段，将旧版本的博客内容作为上下文注入 LLM，使其在最新源码的基础上进行“松散参考重写 (Loose Reference Rewrite)”。

## 2. 核心设计与数据流

### 2.1 提取旧版内容
- **位置**：`backend/internal/service/decomposition_generate.go` -> `GenerateSeries` 方法。
- **逻辑**：在遍历 `outline`（大纲）准备生成章节时，如果遇到 `chapter.Action == "regenerate"` 且 `chapter.ID` 不为空：
  - 通过 `db.DB.First(&oldBlog, "id = ?", chapter.ID)` 从数据库中获取该章节旧版本的 `Content`。
  - 将旧版内容截断（若过长，如截取前 50,000 字符），避免占用过多 Token 上限。

### 2.2 构建 LLM Prompt (System Prompt Injection)
- **策略**：采用 System Prompt 注入旧版本内容。
- **Prompt 构造**：
  在原有的 System Prompt (`你是一个高级技术博客作者。`) 后追加：
  ```
  【注意：本章节为旧版博客的更新重写】
  以下是该章节在旧版本项目中的博客内容，供你作为松散参考。
  你可以参考旧内容中解释抽象概念的比喻、业务知识点或行文风格，但必须以本次提供的最新源码为准进行重写或调整，如果最新代码逻辑发生了改变，请以最新代码为准。
  旧版本内容：
  ---
  {OldContent}
  ---
  ```
- 对于 `Action == "new"` 或无旧版内容的章节，保持原有逻辑，不注入上述内容。

### 2.3 LLM 生成与数据库更新
- LLM 根据最新的源码 (`chapterSourceContent`) 和注入的旧版上下文生成新的 Markdown 内容。
- 生成完毕后，现有的逻辑已支持对指定 ID 的博客执行 `db.Updates` 操作，因此生成的新内容将覆盖旧内容。

## 3. 错误处理与降级
- **数据库查询失败**：如果 `regenerate` 标记的章节无法在数据库中找到对应的 `oldBlog`（可能被意外删除），则平滑降级为普通的 `new` 生成流程，不再注入旧版内容，且不会中断整个系列的生成。
- **Token 截断**：由于旧版内容与最新源码共同构成了 Prompt，必须严格控制总字符数。新源码与旧版内容各自进行字符上限截断，确保不会引发 DeepSeek 的 `invalid_request_error`（单次请求超出上下文限制）。

## 4. 验证计划
- 在本地数据库中构造一条旧版博客数据，模拟触发 `regenerate`。
- 检查后端日志或打印生成的 Prompt，确认旧版内容已成功注入。
- 确认生成出的新博客不仅包含了最新代码，还能看出受到旧版解释的影响。
