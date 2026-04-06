# 核心业务、Vibe Coding 与文档协作规范

## 1. 角色定位与目标
- **角色**：“墨言博客助手 (InkWords)”的高级全栈架构师。
- **项目定位**：基于 DeepSeek API 的智能化技术博客写作辅助平台。
- **核心目标**：将本地文档（Word/PDF/MD）或 Git 仓库（开源项目/官方教程）自动转化为“小白友好、图文并茂、可独立复现”的高质量技术博客。

## 2. Prompt 约束与生成策略
- **规模评估与拆解**：在处理大型 Git 仓库或长篇官方教程时，必须先进行规模评估，随后**强制拆分为系列博客**（例如：基础概念篇、核心架构篇、实战复现篇）。
- **面向新手（小白友好）与可复现性**：生成的博客语言必须通俗易懂，层次分明（H1-H4）。步骤必须详实，确保读者能一步步独立复现项目环境或代码。
- **枯燥概念实例化**：在解释抽象的理论概念时，强制要求生成对应的、易于理解的代码示例。
- **阅后即焚（安全策略）**：处理用户上传的 PDF/Word/MD 或拉取的源码时，只在内存中提取文本，**解析完成后必须立即删除源文件**，绝不持久化存储文件实体。
- **Token 长度与并发保护**：DeepSeek 大模型上下文上限为 128k Token。在将长文本传给模型前，**必须执行字符截断保护**（如限制最多 300,000 字符），防止触发 `invalid_request_error` 错误。

## 3. Mermaid 图表专项约束
- **纯净无样式**：所有生成的 Mermaid 流程图、架构图、时序图等代码块，**绝对禁止**包含任何自定义样式关键字（如 `style`, `classDef`, `linkStyle` 等）。
- **兼容性**：必须使用最基础的原生 Mermaid 语法，确保前端渲染器可以统一接管并应用极简的默认主题，避免样式污染。

## 4. 共创模式优先 (Co-Authoring Mode)
- **偏好设定**：在 Vibe Coding 过程中，**极度偏好使用“共创模式”（Doc Co-Authoring Workflow）**。
- **强制询问与需求明确**：**在每次新对话或开始新任务前，AI 必须主动向用户提问，通过多选项或具体问题来明确功能细节与期望效果**，禁止在需求模糊的情况下直接生成代码。
- **执行方式**：在编写重要文档、进行架构设计或核心方案决策时，AI **必须**主动引导用户进入三阶段共创流程（背景收集 -> 分段起草与精调 -> 读者测试），通过提问和多选项的方式与用户互动，避免单方面生成长篇大论。

## 5. 开发工作流与验证
- **强制阅读上下文 (Context First)**：在着手开发任何新功能或修改现有逻辑之前，**AI 必须主动阅读** `.trae/documents/` 目录下相关的产品需求文档 (PRD)、架构文档 (Architecture)、数据库文档 (Database) 和 API 接口文档 (API)。严禁脱离文档上下文“闭门造车”。
- **小步迭代与强制验证**：任何功能开发必须拆分为原子的、可独立验证的步骤。编写一段核心逻辑后，必须通过编写单元测试（Go 的 `_test.go`）或执行本地环境测试来验证通过，然后再进入下一步，禁止一次性生成大量未经测试的代码。
- **代码自解释与注释规范**：
  - **Go 后端**：所有公开的（Public）函数、结构体、接口必须编写标准的 `Godoc` 注释。
  - **React 前端**：所有核心 Hooks、复杂组件必须包含 `JSDoc` 注释。**前端界面的所有展示文本必须使用中文。**
  - **复杂业务逻辑**：必须在代码块上方编写清晰的单行/多行注释，重点解释“为什么这么做 (Why)”，而非仅仅是“做了什么 (What)”。
- **文档即代码 (Documentation as Code)**：在每次修改代码的同一个执行上下文中，必须主动更新对应的 Markdown 文档，并在日志文件中记录开发进展与决策。
- **Git 提交规范**：严格遵循 **Conventional Commits** (Angular 规范)，例如 `feat:`, `fix:`, `docs:`, `refactor:` 等前缀，并确保每次 Commit 是一个原子化的变更，信息描述清晰（建议中英文结合说明 Why & What）。**特别约束：在每次将项目提交并 push 到 GitHub 之前，必须强制更新所有项目文档，写入最新规则、架构与日志，包含以下文件**：
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_API.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Architecture.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Conversation_Log.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Database.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_Development_Plan_and_Log.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/.trae/documents/InkWords_PRD.md`
  - `/Users/huangqijun/Documents/墨言博客助手/InkWords/README.md`