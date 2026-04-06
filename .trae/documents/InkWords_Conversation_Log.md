# 墨言博客助手 (InkWords) - AI 对话与决策摘要 (Conversation Log)
> **目的**：记录在 Vibe Coding 过程中，每一次核心对话的上下文、用户指令意图以及关键架构决策。以便在长周期的开发中，不论更换 AI 会话窗口还是重新梳理思路，都能快速找回项目背景。

### 对话 39：解决 Git 状态异常并完成项目提交到 GitHub
- **用户需求**：将项目提交到 GitHub。
- **AI 动作**：
  1. 诊断了因 Interactive Rebase 中断导致的暂存区和工作区文件状态错乱问题。
  2. 从 Stash 中成功恢复了丢失的架构文档（如 Map-Reduce 并发架构、可交互式大纲等核心记录）。
  3. 完成了项目的正式 Commit，并打上新的版本标签，推送至 GitHub。
- **决策/变更**：
  - 解决 Rebase 遗留状态并安全提交，保障代码和文档的绝对同步。

### 对话 38：真实场景/效果验证 (Reader Testing)
- **用户需求**：真实场景/效果验证 (Reader Testing)
- **AI 动作**：
  1. 为用户生成了 `.env.example` 模板。
  2. 提供了 3 个优质的开源项目（如 `lo`, `gjson`, `axios`）作为测试素材。
  3. 引导用户启动真实的生产容器集群并填入真实的 DeepSeek API 密钥，随时准备进行联合走查（Co-Review）。
- **决策/变更**：
  - 准备进入真实的生产验证阶段，确保环境配置完整。

### 对话 37：清理测试文件并将项目提交到 GitHub
- **用户需求**：清理测试文件，将项目提交到github。
- **AI 动作**：
  1. 删除了之前创建的 E2E 测试脚本、Docker 编排以及 Mock Server 冗余文件，确保代码库纯净。
  2. 同步更新项目七大核心文档，并调用 git 技能完成版本控制与提交。
- **决策/变更**：
  - 在完成端到端测试与并发 UI 的全流程收尾后，清理不需要的测试环境冗余文件。

### 对话 36：项目端到端测试与并发 UI Bug 修复
- **用户需求**：对我的项目进行端到端测试，请询问我来明确测试需求。
- **AI 动作**：
  1. 通过 `AskUserQuestion` 技能明确了测试重点、框架偏好与环境配置。
  2. 搭建了全隔离的 Playwright E2E 测试套件，并编写了 Mock LLM 服务器加速测试。
  3. 修复了测试中暴露的 Go Context 过早 Canceled 以及前端 SSE 断流后 UI 永久 Loading 死锁的严重 Bug。
- **决策/变更**：
  - 引入全隔离的 Mock 环境进行 E2E 测试，不仅能加快测试速度，还能暴露深度的并发状态 Bug。

### 对话 35：修复后端容器拉取 Git 仓库失败 (exec: "git": executable file not found in $PATH)
- **用户需求**：用户提供了一张截图，显示在生成分析任务时报错：`分析失败: 拉取仓库失败: failed to clone repository: exec: "git": executable file not found in $PATH, stderr: `。
- **AI 动作**：
  1. 识别出报错是因为后端 Go 程序运行在 `alpine` 容器内，而该运行时镜像缺少 `git` 依赖。
  2. 修改 `backend/Dockerfile` 的 `Stage 2: Runtime` 阶段，在 `apk add` 命令中补充安装了 `git`。
  3. 执行 `docker compose build backend` 验证镜像可以成功构建。
  4. 同步更新了开发日志和本对话记录。
- **决策/变更**：
  - 容器化部署中，除了构建阶段（Builder）需要特定工具链外，如果运行时（Runtime）的业务逻辑（如使用 `os/exec` 调用外部命令）依赖系统级工具，必须在运行镜像中显式安装。

### 对话 34：修复 SSE 流被代理或错误拦截时的迷惑报错 (Expected content-type...)
- **用户需求**：用户贴出浏览器控制台关于 `Expected content-type to be text/event-stream, Actual: text/plain` 的报错，可能因为后端没启动或代理错误，要求修复。
- **AI 动作**：
  1. 诊断出 `@microsoft/fetch-event-source` 库在建连时严格校验 `Content-Type`，当后端代理报错 `502` 或返回 JSON 错误信息时，库会静默抛出类型错误，掩盖真实原因。
  2. 修改前端 `src/hooks/useBlogStream.ts` 和 `src/components/Editor.tsx` 中所有的 `fetchEventSource` 调用，补充 `onopen` 拦截钩子。
  3. 当响应非 `text/event-stream` 时，主动通过 `response.json()` 或 `response.text()` 读取真实报错信息，并封装为 `StopStreamError` 抛出给外层提示用户。
  4. 为用户一键重启了前后端项目，并在随后更新了项目文档并提交至 GitHub。
- **决策/变更**：
  - 前端处理 SSE 流时，必须实现自定义的 `onopen` 拦截器来解析业务级或代理级的普通 HTTP 错误，而不是依赖底层的隐式报错，这对于排查网络连通性问题至关重要。

### 对话 33：整理项目基准文档并提交至 GitHub
- **用户需求**：要求确认提交内容并提交项目到 GitHub。
- **AI 动作**：
  1. 通过 `git status` 和 `git diff` 分析了最近的修改内容：包含前端侧边栏的批量导出 ZIP 功能、防溢出滚动条修复，以及后端的系列导出接口和项目文档更新。
  2. 补全并同步了所有的项目基准文档，包括在 PRD 中加入批量导出说明、在架构文档和日志文档中详细记录了功能开发细节。
  3. 通过 `AskUserQuestion` 与用户确认 Git 提交信息以及是否推送代码。
- **决策/变更**：
  - 严格执行 `git-and-release.md` 中的规范，在提交代码前更新和核对 7 份核心项目文档，完成文档即代码同步。

### 对话 32：修复历史博客列表拉长页面及新增批量导出功能
- **用户需求**：用户指出历史博客列表部分在展开文件夹时页面会被拉长，没有出现滚动条；同时找不到批量导出按钮，要求通过提问明确需求并修复。
- **AI 动作**：
  1. 调用 `AskUserQuestion` 明确了三个需求：滚动条表现（仅列表滚动）、导出按钮位置（侧边栏多选管理模式）、导出文件格式（ZIP 压缩包）。
  2. **修复滚动条溢出**：修改 `App.tsx` 中的全局样式为 `h-screen overflow-hidden`，让 `Sidebar.tsx` 内的历史记录列表容器 `flex-1 overflow-y-auto` 生效，防止被子元素撑爆。
  3. **实现批量导出功能**：
     - 在前端 `Sidebar.tsx` 新增“批量管理”模式开关（文件夹图标）。
     - 引入 `jszip` 库，在多选模式下允许用户勾选父节点或子节点博客，点击“导出 ZIP”后，将前端缓存中的所有选中文章按层级和序号命名并打包。
     - 在后端 `api/blog.go` 补充了 `/api/v1/blogs/:id/export` 导出整个系列文章为 ZIP 的接口，供在单篇博客页面一键导出该系列。
  4. 精简和合并了 `.trae/rules` 目录下的多份零散规则，将它们收拢到 3 份核心文档（架构、Vibe 工作流、Git规范）中。
- **决策/变更**：
  - 在侧边栏采用多选框交互实现批量导出是最直观的体验，前端引入 `jszip` 进行轻量级打包下载，后端辅以接口支持系列一键导出，覆盖了不同场景的需求。
  - 对于复杂的多文档项目规则，将其合并整理，有助于降低 AI 上下文读取压力，并增强核心约束的传达力度。

### 对话 31：增强内容生成兜底与 UI 滚动优化
- **用户需求**：用户反馈在生成“GSAP插件生态”系列时，第二部分内容不全（可能因大模型上限突然中断），并且生成的系列名称被硬编码为“Git源码解析系列”。同时，随着生成博客列表的增加，页面被拉长影响体验。要求通过提问明确需求并修复。
- **AI 动作**：
  1. 调用 `AskUserQuestion` 明确了三个决策：“后端优化提示词+前端增加继续生成兜底”、“AI自动提取主题+允许用户手动修改”、“主界面与侧边栏列表均增加局部滚动条”。
  2. 修复硬编码标题：修改 `GenerateOutline` 提示词，要求返回包含 `series_title` 的 JSON 对象。在前端 `Generator.tsx` 提供标题输入框，并在后端 `GenerateSeries` 时使用该自定义标题创建父节点。
  3. 修复内容不全：除了在提示词中强调“完整性约束”外，开发了兜底方案——在编辑器右上角增加“继续生成”按钮。点击后调用新增的 `POST /api/v1/blogs/:id/continue` 接口，大模型基于现有内容自动续写并通过 SSE 流式追加。
  4. 修复 UI 拉长：为 `Generator.tsx` 的大纲列表和 `Sidebar.tsx` 的当前生成任务列表分别添加了 `max-h-[40vh]` 和 `max-h-[30vh]` 以及 `custom-scrollbar` 局部滚动样式。
  5. 验证前后端编译通过后，更新了所有的项目基准文档，并准备执行 Git 提交与推送。
- **决策/变更**：
  - 大语言模型在输出超长文本时截断是偶发且难以完全避免的物理限制。除了优化提示词外，在产品 UI 层提供“继续生成（续写）”按钮是最佳的工程兜底实践。

### 对话 30：解决长文生成截断、内容空洞与切换页面卡死问题
- **用户需求**：用户反馈最后生成的几篇文章出现了截断（生成不全），且内容比较空洞；同时指出在生成过程中一旦切换页面，项目就会卡死。要求 AI 询问明确需求并修复，并完成所有项目文档的更新与 GitHub 推送。
- **AI 动作**：
  1. 调用 `AskUserQuestion` 明确了三个关键决策：“后台静默生成”、“仅点击按钮中止”以及“精准按需喂代码”。
  2. 修复切换页面卡死：将 `AbortController`、`analysisStep` 等流状态从局部组件剥离，移入全局 Zustand `streamStore.ts`，并移除了 `useEffect` 卸载时的强制中断逻辑。实现了完美的“后台静默生成”，切回页面进度无缝衔接。
  3. 优化大纲粒度：修改后端 Prompt，强制要求大型项目拆分为 5-10 篇细粒度博客，单篇字数约束为 1000-1500 字，避免单篇过长触发 LLM 最大输出 Token 截断。
  4. **架构重构（精准按需喂代码）**：
     - 修改 `Chapter` 结构体，增加 `Files []string` 字段。要求 AI 在生成大纲时，必须基于目录树为每个章节指定强相关的源码文件。
     - 修改 `GenerateSeries` 逻辑：后端接收到大纲后，临时 Clone 仓库，在串行生成每一章时，**仅读取**大纲中指定的几个文件源码作为上下文。
     - 这一举措将单次 API 调用的上下文从几十万字的“项目全局摘要”骤降至几千字的“核心源码”，彻底解决了大模型注意力分散导致的“内容空洞”和“假大空”问题。
  5. 修复端口冲突：清理了后台残留的 Vite 进程，确保前端稳定运行在 `5173` 端口，解决了此前引发的 GitHub OAuth 回调 `ERR_ABORTED` 问题。
  6. 在完成所有代码修改与验证后，通过本轮对话将所有的架构变更与开发日志写入项目基准文档，准备执行 Git Tag 发布流水线。
- **决策/变更**：
  - 将大模型的上下文供给策略由“全量粗放投喂”升级为“按图索骥的精准按需投喂”，是提升 AI 代码解析类文章质量的质变级架构调整。
  - 将长耗时的流式网络请求状态提升至全局 Store 管理，是现代 React 复杂应用中保障体验流畅性的标准范式。

### 对话 29：修复 React 状态延迟引起的博客互相覆盖 Bug 并清理脏数据
- **用户需求**：用户反馈在点击“历史博客”系列文章时，发现多篇文章内容变成了同一篇。要求查明原因，并在修复后通过提问明确如何处理脏数据。
- **AI 动作**：
  1. 深度排查发现：由于 `Editor.tsx` 组件在切换博客时复用，其内部的 `useDebounce` 状态未及时重置，导致 2 秒后旧的 `debouncedTitle` 和 `debouncedContent` 触发了对新 `selectedBlog.id` 的自动保存，从而引发文章内容被互相覆盖的 Bug。
  2. 修复方案：在 `App.tsx` 中为 `<Editor />` 组件添加 `key={selectedBlog.id}`，确保每次切换博客时彻底卸载并重新挂载编辑器组件，清空所有内部状态和延迟定时器。
  3. 优化体验：在 `Editor.tsx` 中新增了基于组件卸载的生命周期钩子，如果用户输入后立刻切换文章，会在卸载前立刻触发最后一次自动保存，防止丢失最新输入。
  4. 使用 `AskUserQuestion` 询问用户如何处理已经覆盖的脏数据，用户选择“帮忙清理脏数据”。
  5. 编写了 `backend/scripts/cleanup.go` 脚本，用于自动查询并清理数据库中由于该 Bug 产生的同一个父节点下标题重复的脏数据。
- **决策/变更**：
  - 明确在 React 中处理“不同实体复用同一表单/编辑器组件”时，强制使用 `key` 属性重新挂载是避免状态残留（特别是防抖/节流钩子）的最佳实践。
  - 通过组件卸载前同步执行乐观更新，完美平衡了“切换时立刻清空状态”与“不丢失最后输入内容”的双重需求。

### 对话 28：修复大项目分析生成卡死问题与添加超时/重试机制
- **用户需求**：用户反馈在分析大项目并生成系列博客时，有概率在生成第一、第二篇之后卡住。要求分析原因，并通过提问明确修改需求。
- **AI 动作**：
  1. 通过代码分析定位到 3 个卡死风险点：客户端断开导致的 `errChan` 永久阻塞（Goroutine 泄露）、`http.Client` 未设置超时导致的流式读取假死、以及串行生成和并发信号量耗尽引发的雪崩效应。
  2. 调用 `AskUserQuestion` 向用户明确了三个维度的处理策略：流式空闲超时、后台继续生成（防泄露）、自动重试。
  3. 修改 `internal/api/stream.go`：引入 `context.WithoutCancel` 分离 HTTP 生命周期与业务生命周期。当客户端（前端）断开时，启动后台协程排空（drain）`chunkChan` 和 `errChan`，确保生成任务在后台继续执行并落库，防止 Goroutine 阻塞泄露。
  4. 修改 `internal/service/decomposition.go` 中的 `GenerateSeries`：为单章生成引入最大 3 次的 `for` 循环自动重试机制；并使用 `time.Timer` 实现了 30 秒的流式空闲超时（Idle Timeout），一旦超时则主动 `cancel()` HTTP 请求并触发重试。
  5. 修改 `internal/service/generator.go` 中的 `GenerateBlogStream`：同样引入了 30 秒的流式空闲超时保护。
  6. 修改 `internal/service/decomposition.go` 中的 `generateLocalSummaryWithRetry`：为单次非流式的 HTTP 请求增加了 3 分钟的 Context 超时控制。
- **决策/变更**：
  - 彻底解决了由于网络波动或客户端异常断开导致的服务端“假死”和资源泄露问题。
  - 实现了“前端断开，后台继续”的健壮逻辑，极大提升了大项目（耗时较长）生成的最终成功率。

### 对话 27：使用 Playwright 进行全链路端到端 (E2E) 测试及 Bug 修复
- **用户需求**：要求使用 Playwright 对项目进行全方位的测试，并通过提问明确测试细节。
- **AI 动作**：
  1. 使用 `AskUserQuestion` 明确了测试范围（全局 E2E）、服务管理（自动启动全栈服务）以及测试产物（截图、控制台日志、网络请求监控）。
  2. 调用 `webapp-testing` 技能，查阅了 `with_server.py` 辅助脚本的使用说明。
  3. 编写了完整的 Playwright E2E 测试脚本 `playwright_e2e_test.py`，覆盖了导航、注册、上传 PDF、文件解析、开始生成、生成完毕、侧边栏点击查看的完整闭环。
  4. 运行测试时，发现并修复了两个前端 Bug：
     - `Generator.tsx` 中当 `store.outline` 为空数组 `[]` 时，`every` 方法返回 `true` 导致生成按钮被错误隐藏的问题。
     - `useBlogStream.ts` 中单文件生成完毕后未调用 `store.reset()`，导致 UI 无法正确重置的问题。
  5. 修复 Bug 后再次运行测试，成功跑通了整个自动化测试流程，所有测试产物（日志与截图）均保存至 `test_results/` 目录。
- **决策/变更**：
  - 引入了自动化的全栈 E2E 测试机制，极大提升了项目的回归测试效率。
  - 发现 `Array.prototype.every` 在空数组上的边缘 case 并予以修复，保障了单文件（非系列）生成的 UI 状态流转正确性。

### 对话 26：项目文档更新、Git 提交与发布版本 v1.0.0
- **用户需求**：要求将今天完成的所有功能（Markdown 双向同步、Vibe Coding 规范更新）写入项目基准文档，并提交至 GitHub，打上 `v1.0.0` 的版本标签。
- **AI 动作**：
  1. 使用 `AskUserQuestion` 明确了文档更新范围（滚动同步及其他）以及需要打的 Git 标签级别（`v1.0.0`）。
  2. 执行 `git diff` 梳理了代码变更细节。
  3. 更新 `InkWords_Architecture.md`，增加了 Rehype 自定义插件和双向滚动插值算法的架构描述。
  4. 更新 `InkWords_PRD.md`，补充了按元素精准双向滚动同步的产品需求。
  5. 更新 `InkWords_Development_Plan_and_Log.md`，记录了今天的开发日志与踩坑总结。
  6. 更新了本对话日志文件。
  7. 随后将执行 `git add .`、`git commit`（Angular 规范）、`git tag v1.0.0` 及 `git push --tags` 完成完整发布流。
- **决策/变更**：
  - 严格执行了今天刚刚写入 `.trae/rules` 的《Git 提交与发布规范》，通过询问、对比、详细描述、打标、推送的标准化流水线，完成了 V1.0.0 版本的商业化里程碑发布。

### 对话 25：添加 GitHub 提交规范与修改 Vibe Coding 流程
- **用户需求**：要求将“每次对话前都要向我询问来明确功能”写入 `vibe-coding-workflow.md`，并在 `.trae/rules/` 中新增关于 GitHub 提交前明确差异、详细 Commit、打标签再推送的强制规范。
- **AI 动作**：
  1. 修改了 `vibe-coding-workflow.md` 的共创模式要求，添加了强制询问的规定。
  2. 新建了 `git-and-release.md` 规则文件，记录了对比、编写规范与打标签的 3 步走策略。
- **决策/变更**：
  - 提升了 AI 助手在编码过程中的主动性与谨慎性，强制开启交互式需求确认，并标准化了代码的版本发布流程。

### 对话 24.5：实现 Markdown 编辑区与预览区按元素精准双向滚动同步
- **用户需求**：用户指出左侧 Markdown 编辑部分和右侧预览部分滚动不同步，要求 AI 先提问明确需求。
- **AI 动作**：
  1. 主动调用 `AskUserQuestion` 工具询问了具体表现（滚动位置不同步）以及期望效果（按元素精准对齐）。
  2. 编写了 `rehypeSourceLine` 插件并挂载到 `MarkdownEngine`，给右侧生成的 HTML 元素打上 `data-source-line` 标签。
  3. 在 `Editor.tsx` 中编写了复杂的双向插值滚动算法 `handleEditorScroll` 和 `handlePreviewScroll`，并处理了头部与底部的边界计算。
- **决策/变更**：
  - 放弃简单的按比例计算（这会导致图表区域严重错位），采用获取视口内真实 DOM 的 `offsetTop` 偏移量并与行号做比例插值的硬核方案，一劳永逸地解决了图文混排导致的高低失调问题。

### 对话 24：解决文章依旧无法查看的问题并优化生成后的右侧面板
- **用户需求**：用户反馈在修复“孤岛数据”问题后，生成的系列文章依然无法添加到“历史博客”并点击查看。截图显示左侧“当前生成任务”已经全部打勾完成，但右侧主区域依旧停留在“准备生成/开始生成”状态。
- **AI 动作**：
  1. 深度排查发现：上一轮（对话22）对 `backend/internal/service/decomposition.go` 的“持久化父节点”修复代码实际上**并未生效**。由于端口 `:8080` 被旧进程占用，Go 后端启动失败，一直在运行没有该修复逻辑的旧代码！
  2. 使用 `lsof` 找到旧后端的进程 PID，通过 `kill -9` 强杀后，重新启动了携带最新修复代码的 Go 后端服务。
  3. 考虑到用户数据库中已经存在了一批没有父节点的“孤岛”子文章，编写并执行了一条补救的 PostgreSQL SQL 语句，通过子节点的 `parent_id` 自动反向生成并恢复了 8 个缺失的父节点，让之前的生成成果也重见天日。
  4. 修复了前端的 UI 状态倒退问题：修改 `Generator.tsx`，当所有章节状态变为 `completed` 时，将右侧面板的“开始生成”按钮隐藏，并把文案修改为“系列博客生成完毕，请点击左侧查看”。避免给用户带来“任务没完成”的错觉。
- **决策/变更**：
  - 代码热重载失败或端口冲突往往是“改了代码不生效”的元凶。修复旧脏数据的同时，保证新的流程通过重新编译彻底阻断数据孤岛的产生。
  - 在大项目生成结束后保留左侧“当前生成任务”打勾面板，让用户直观地看到进度结果，并在右侧给予明确的“完成提示与引导”。

### 对话 23：优化生成页面的高度与隐藏 Mermaid 报错
- **用户需求**：用户反馈在生成博客文章时，预览模块会随着内容的增加越来越大，导致页面卡顿。同时希望隐藏生成过程中页面上直接暴露的报错信息（如 Mermaid 图表未生成完整时的 Syntax Error）。
- **AI 动作**：
  1. 调用 `AskUserQuestion` 向用户明确了修改偏好，确认了“最大高度限制为 500px”以及“完全隐藏报错”的方案。
  2. 修改 `frontend/src/components/Generator.tsx`，为打字机预览容器（包裹 `<MarkdownEngine />` 的 `div`）增加 Tailwind 属性 `max-h-[500px] overflow-y-auto`。
  3. 修改 `frontend/src/components/MarkdownEngine.tsx`，在 `mermaid.initialize` 配置和实例配置中注入 `suppressErrorRendering: true`，并在渲染失败的 `catch` 块中通过清空 `innerHTML` 彻底屏蔽原生红色报错图。
  4. 同步更新了架构与开发相关的日志文档。
- **决策/变更**：
  - 在流式生成（Streaming）场景中，所有代码块的“中间态”本质上都是非法语法的，应当从 UI 层面对所有的语法分析引擎进行报错静默处理，以此保障用户无缝的视觉体验。

### 对话 22：修复生成的系列博客无法添加到“历史博客”的问题
- **用户需求**：用户反馈通过分析代码仓库生成的博客不能正常添加到“历史博客”列表中（只能在“当前生产任务”栏看到），且无法点击查看内容（只能看预览）。用户要求 AI 先通过提问明确修改需求，再进行修复。
- **AI 动作**：
  1. 调用 `AskUserQuestion` 工具向用户提出了 3 个问题，明确了用户的期望：任务完成后“自动刷新”列表、历史记录采用“树状折叠展示”、点击生产任务时“联动打开并在左侧树中高亮”。
  2. 排查后端 `backend/internal/service/decomposition.go`，发现 `GenerateSeries` 虽为子章节分配了 `parentID`，但漏掉了将父节点本身存入数据库的步骤，导致整组文章成为孤岛数据。修复并持久化父节点。
  3. 排查前端 `frontend/src/components/Sidebar.tsx`，重写了“当前生产任务”卡片的 `onClick` 事件，支持在嵌套的 `blogs` 树中递归查找刚生成的真实博客节点。
  4. 查找到匹配的节点后，提取其 `parentId` 加入到 `expandedNodes` 集合中自动展开目录树，并触发 `selectBlog` 在右侧主区域打开完整内容。
- **决策/变更**：
  - 严格遵循了“共创模式（Co-Authoring Mode）”偏好，在动手修改核心逻辑前，先用多选项弹窗收集了用户的交互期望，避免了闭门造车。
  - 彻底打通了从“流式生成完毕” -> “自动入库” -> “列表刷新” -> “自动定位展开并高亮” 的全链路体验。

### 对话 21：项目文档更新与代码提交
- **用户需求**：要求更新项目的所有基准文档（PRD、API、架构、数据库、开发日志、对话日志），更新完成后提交代码至 GitHub，并在查阅现有标签后打上新标签。
- **AI 动作**：
  1. 查阅并更新了 `InkWords_API.md`，将分析接口 `/api/v1/project/analyze` 更新为流式接口 `/api/v1/stream/analyze`。
  2. 查阅并更新了 `InkWords_Architecture.md`，在“核心业务流转设计”中补充了分析阶段的 SSE 进度推送机制。
  3. 查阅并更新了 `InkWords_PRD.md`，在“大项目拆解流程”中补充了前端打字机效果及 SSE 进度条的需求描述。
  4. 查阅并更新了 `InkWords_Database.md`，确认数据库表结构无需变更，保持最新状态。
  5. 检查了 `InkWords_Development_Plan_and_Log.md` 与本文件，确认日志已是最新。
  6. 接下来将执行 `git add`, `git commit`, `git push` 以及打标签操作。
- **决策/变更**：
  - 确保“文档即代码”的一致性，将今日修复的“串行打字机”、“SSE进度流”、“解决隐藏挂起 Bug”等核心机制正式写入基准文档，为后续版本迭代提供准确参考。

### 对话 20：修复 SSE 流被浏览器后台标签页机制中断的问题 (ERR_ABORTED)
- **用户需求**：用户贴出了 `net::ERR_ABORTED ... at onVisibilityChange` 的报错日志，在分析/生成大项目期间，如果切换了浏览器标签页，SSE 流会被直接切断并报错。
- **AI 动作**：
  1. 识别到这是 `@microsoft/fetch-event-source` 库的默认行为：当标签页失去焦点（Visibility Change）时，它会主动中断并挂起连接。
  2. 修改 `frontend/src/hooks/useBlogStream.ts` 中所有的 `fetchEventSource` 调用（包括 `/analyze` 和两个 `/generate` 场景），统一加上 `openWhenHidden: true` 参数。
- **决策/变更**：
  - 由于大模型生成或 Git 仓库克隆的耗时较长，用户极大可能会在此期间切到别的标签页干别的事情。关闭隐藏时的断流保护能确保任务在后台继续执行，提高整体体验与成功率。

### 对话 19：优化 Git 仓库分析过程，引入真实 SSE 进度流
- **AI 动作**：
  1. 通过 `AskUserQuestion` 与用户确认了采用“真实 SSE 进度流 + 轻量化克隆”的优化策略。
  2. 修改 `backend/internal/api/stream.go`，新增了 `/api/v1/stream/analyze` 路由处理器，专门用于将原本 HTTP POST 的短连接改造为长连接的 SSE 流。
  3. 修改 `backend/internal/service/decomposition.go`，新增了 `AnalyzeStream` 方法。在克隆仓库、生成大纲的各个核心节点，通过通道主动向下发包含 `step` 和 `message` 的 JSON 数据；并在最后一步 `step: 3` 时将大纲和提取的源码 `data` 下发给前端。
  4. 修改 `frontend/src/hooks/useBlogStream.ts` 中的 `analyzeGit`，弃用原有的 `fetch` 改用 `fetchEventSource`。通过解析下发的事件流，动态更新 `analysisStep` 和 `analysisMessage` 状态。
  5. 优化了 `frontend/src/components/Generator.tsx` 中的进度展示，移除了原有的 `useEffect` 假定时器，现在分析进度文字会随着后端 SSE 的真实状态同步改变。
- **决策/变更**：
  - 将大项目分析阶段也从“黑盒等待”升级为了“白盒透明”，用户可以看到当前是在“克隆拉取”还是在“AI生成大纲”，缓解了因大项目耗时过长带来的焦虑感。
  - 保留了本地文件的假进度条（因为文件解析非常快），仅对耗时极长的大模型分析步骤开启 SSE 改造。
- **用户需求**：用户指出在输入 Git 仓库生成系列博客时：1. 没有进度条（生成过程不明显，卡在0字符很久然后突然完成）；2. 不希望同时生成多篇文章；3. 无法点击卡片，想知道如何查看真实文章。
- **AI 动作**：
  1. 通过 `AskUserQuestion` 与用户确认了采用“修复打字机 + 恢复完成状态点击”的解决方案。
  2. 修改 `backend/internal/service/decomposition.go`：修复了极其隐蔽的 Bug —— 之前在 Go 中使用 `fmt.Sprintf` 拼接 SSE 的 JSON 返回数据，大模型输出的文本中带有回车、引号等特殊字符导致前端 `JSON.parse` 报错被静默吞掉，从而表现为“假死，0 字符，很久后突然完成”。现已改用标准的 `json.Marshal(map[string]interface{})`。
  3. 修改 `backend/internal/service/decomposition.go` 中针对每一篇文章生成期间的 SSE 事件发送逻辑：将 LLM 返回的每个 chunk 打包为 `{"status":"streaming", "content":"..."}` 事件推送给前端。
  4. 修改 `frontend/src/hooks/useBlogStream.ts`，接收到 `streaming` 事件时实时更新 Zustand 的 `generatedContent`；当某章 `completed` 时主动调用 `fetchBlogs()` 刷新历史列表。
  5. 修改 `frontend/src/components/Generator.tsx`，合并了 `file` 和 `git` 的生成中状态，在系列生成时统一使用 `MarkdownEngine` 实时渲染当前正在生成的章节内容。
  6. 修改 `frontend/src/components/Sidebar.tsx`，恢复了卡片的 `onClick` 事件，但仅限于该章节状态变为 `completed`（变绿）时触发。点击后不再渲染“草稿假数据”，而是使用递归算法从最新的 `blogs` 树中查找对应的真实历史文章，并自动跳转至编辑器面板打开它。
- **决策/变更**：
  - 取消了系列文章的并发生成（为了提供沉浸式的打字机反馈体验并防止内容过短）。
  - 将单文件生成与系列文章生成的“生成中”UI体验完全对齐，利用串行化和精确的 JSON 序列化彻底解决了状态展示混乱、打字机罢工的问题。
  - 通过匹配历史文章的方式恢复卡片点击，解决了之前“草稿数据”被当做最终结果的误导问题。

### 对话 17：新增“新建博客 / 返回首页”功能
- **用户需求**：用户反馈在点击进入生成的博客详情（编辑器）后，无法退出当前状态并返回到首页进行下一篇博客的创作，要求修复并通过提问方式明确交互功能。
- **AI 动作**：
  1. 通过 `AskUserQuestion` 提供了三个交互位置选项（侧边栏顶部按钮、Logo可点击、编辑器顶部按钮），用户选择了“侧边栏顶部按钮 (推荐)”。
  2. 修改 `frontend/src/components/Sidebar.tsx`，在左侧边栏顶部的 Logo 下方增加了一个醒目的“新建博客 / 返回首页”按钮。
  3. 绑定点击事件 `selectBlog(null)`，清空当前选中的博客状态，从而触发 `App.tsx` 重新渲染 `<Generator />` 首页组件。
  4. 同步更新了开发日志和对话记录，保持“文档即代码”的一致性。
- **决策/变更**：
  - 采用显式的全局常驻按钮来处理“退出阅读/编辑态”的交互，比单纯的 Logo 点击更符合直觉，提升了用户在不同博客间切换及新建任务的体验。

### 对话 15：修复 GitHub 仓库解析内容少与单文件上传进度条体验
- **用户需求**：用户指出解析 GitHub 仓库时生成的内容过少且未按要求拆分系列博客；另外上传文件生成博客时，前端缺少流式生成的进度条提示。
- **AI 动作**：
  1. 通过提问明确了用户的具体痛点（GitHub源码读取不完整/内容太简略/未拆分系列，上传文件需要全链路综合进度提示）。
  2. 修改 `backend/internal/parser/git_fetcher.go`，在读取源码时优先将目录结构（Tree）拼接到最上方，保证即便后续因过长截断，大模型也能了解仓库整体结构。
  3. 修改 `backend/internal/service/decomposition.go`，在 `GenerateOutline` 提示词中**强制要求**大型项目至少拆分为3个章节；并在 `GenerateSeries` 的提示词中要求大模型必须深入分析、引用核心代码、提供复现步骤，甚至要求它根据目录结构合理推演缺失部分。
  4. 修改 `frontend/src/store/streamStore.ts` 增加 `generatedContent` 状态来保存单篇博客生成的流式文本。
  5. 修改 `frontend/src/hooks/useBlogStream.ts` 在 `generateSingle` 接收到 `chunk` 事件时，拼接实时内容。
  6. 修改 `frontend/src/components/Generator.tsx`，在单文件生成过程中使用 `MarkdownEngine` 实时渲染生成的 Markdown 文本流，实现打字机效果。
- **决策/变更**：
  - GitHub 仓库过大截断是不可避免的物理限制，通过“注入目录结构”加“强化 Prompt 深度”来对抗截断导致的内容空洞。
  - 单文件生成的体验从“Loading Spinner 盲等”升级为“实时 Markdown 渲染”，提升了反馈感知。

### 对话 16：修复 Mermaid 图表渲染与 PDF 解析报错
- **用户需求**：用户贴出了控制台中出现的 `UnknownDiagramError: No diagram type detected matching given configuration for text: undefined` 以及 PDF 解析时出现的 `failed to create PDF reader: not a PDF file: missing %%EOF` 报错日志。
- **AI 动作**：
  1. 修复了 `frontend/src/components/MarkdownEngine.tsx`，在传递给 Mermaid 渲染引擎之前增加了非空校验和对 `undefined` 文本流的过滤防护，防止因为打字机流式渲染期间不完整的图表语法导致崩溃。
  2. 修复了 `backend/internal/parser/doc_parser.go` 中 PDF 的解析流程。在临时文件拷贝完成后，主动调用 `tempFile.Sync()` 强制刷盘，并执行 `Seek(0, 0)`，防止第三方 PDF 库因为读取指针位置错误或文件未完全写入导致的 `missing %%EOF`（文件不完整）错误。
  3. 为 `backend/internal/api/project.go` 增加了对空文件的阻断拦截（`header.Size == 0`）。
  4. 为 `frontend/src/hooks/useBlogStream.ts` 补充了针对文件解析报错的用户友好提示框（`alert`），防止前端页面只在控制台打错而没有任何反馈。
- **决策/变更**：
  - 流式生成的 Markdown 渲染组件必须具备极强的容错性，因为中间态的代码块极大概率是不完整的语法。
  - Go 语言中利用 `io.Copy` 处理文件时必须严格控制文件指针和缓冲区刷盘，尤其是在第三方依赖直接使用该文件句柄时。

### 对话 14：修复 SSE 连接与 AbortError 日志报错
- **用户需求**：用户贴出了控制台中出现的 `AbortError: BodyStreamBuffer was aborted` 报错日志，要求进行分析与修复。并且要求修复完毕后同步文档并推送到 GitHub。
- **AI 动作**：
  1. 分析并定位到问题原因是 SSE 网络流由于服务端在返回 `[DONE]` 后迅速关闭或组件卸载时的残余流导致的。
  2. 修改 `useBlogStream.ts`，引入 `useRef` 保存 `AbortController` 实例，在组件卸载 (`useEffect` 清理函数) 和每次发起新生成前主动调用 `abort()`。
  3. 在 `fetchEventSource` 的 `onerror` 和外层 `catch` 中增加了对 `AbortError` 的静默过滤，将其转化为受控的 `StopStreamError` 以阻止默认报错和重连。
  4. 顺带修复了 `Generator.tsx` 和 `useBlogStream.ts` 中的 Lint 报错（`react-hooks/set-state-in-effect` 及 `any` 类型使用），目前项目 `npm run lint` 已零错误。
  5. 补充并更新了本文件和开发日志。
- **决策/变更**：
  - 主动接管 `fetch-event-source` 的中止信号生命周期，明确区分正常结束/组件卸载导致的流切断与网络真实断开的错误。

### 对话 13：添加用户注销（退出登录）功能
- **用户需求**：要求为项目添加支持用户注销的功能，完成后更新项目文档、提交 GitHub 并打上对应标签。
- **AI 动作**：
  1. 修改前端 `frontend/src/components/Sidebar.tsx` 组件，在底部添加了带有注销图标的「退出登录」按钮。
  2. 实现了点击按钮后清除 `localStorage` 中的 `token` 并重定向到根路径强制刷新页面，显示回登录组件。
  3. 更新了开发日志和对话记录。
  4. 使用 Git 提交代码并打上对应功能的 Tag。
- **决策/变更**：
  - 确认了系统使用的是无状态的 JWT 方案，因此注销逻辑完全由前端承担（清理本地凭证），无需后端 API 参与。

### 对话 12：修复 GitHub 登录重定向与前端列表加载 Bug
- **用户需求**：要求修复 GitHub 授权登录后卡在后端 JSON 响应页面无法返回前端的问题，以及前端页面加载时 `fetchBlogs` 抛出 JSON 解析错误的问题。
- **AI 动作**：
  1. 修改后端 `internal/api/auth.go` 的 `OAuthCallback`，将 JSON 响应替换为 `c.Redirect` 重定向回前端（携带 `?token=` 或 `?error=`）。
  2. 修改前端 `Login.tsx`，使用 `useEffect` 解析并提示 URL 中的 `error` 参数，随后清理 URL。
  3. 修改前端 `blogStore.ts`，将 `fetchBlogs` 中的直接 `res.json()` 改为先获取 `res.text()`，增加判空和 `try-catch` 保护，防止空响应导致报错。
  4. 更新了 PRD、API、开发日志等相关基准文档。
- **决策/变更**：
  - 明确第三方 OAuth 回调的闭环必须通过 HTTP 重定向交还控制权给前端，由前端提取状态（Token 或 Error）并处理后续路由和用户提示。

### 对话 11：常规登录实现与依赖/图标修复
- **用户需求**：要求更新开发日志和对话记录，补充今日的 Bug 修复与新特性开发：移除 `rehype-mermaid`、修复 `lucide-react` 的 Github 图标以及后端常规登录与注册功能的实现。
- **AI 动作**：
  1. 在 `InkWords_Development_Plan_and_Log.md` 中追加了对应的 Bugfix & Feature 记录。
  2. 在 `InkWords_Conversation_Log.md` 中补充了本次对话与决策的摘要。
- **决策/变更**：
  - **架构微调**：为了严格执行“纯净无样式”的图表渲染约束，移除了 `rehype-mermaid` 依赖，避免了样式冲突。
  - **功能完善**：在已有的第三方 OAuth 授权之外，补充了传统的账号密码登录与注册机制，完善了用户鉴权闭环。

### 对话 10：修复编辑器输入覆盖 Bug
- **用户需求**：要求修复 `Editor.tsx` 中由于自动保存导致 `selectedBlog` 更新，从而触发 `useEffect` 覆盖用户正在输入的内容的 Bug。同时要求**不更新**个人任务文件 `tasks.md` 或 `checklist.md`。
- **AI 动作**：
  1. 将 `Editor.tsx` 中同步博客内容的 `useEffect` 依赖项从 `selectedBlog` 修改为 `selectedBlog?.id`。
  2. 禁用了相关的 `eslint-disable-next-line react-hooks/exhaustive-deps` 检查以消除 linter 警告。
  3. 顺手修复了 `MarkdownEngine.tsx` 中的 `node` 未使用报错。
  4. 运行 `npm run lint` 验证所有问题均已修复。
  5. 更新了项目开发日志 `InkWords_Development_Plan_and_Log.md` 以及本对话日志。
- **决策/变更**：
  - 核心决策是**只在切换文章时同步状态**，而不再每次保存后覆盖本地组件状态，以保障用户的输入流畅性。

---

## 2026-04-04 (项目规划与脚手架搭建)

### 对话 1：项目初步规划 (Plan Mode)
- **用户需求**：要求开发一个基于 DeepSeek API 的博客写作助手，前端使用 React 18，后端使用 Go，支持 Word/PDF/MD 及 Git 仓库解析，大项目自动拆解，Mermaid 图表无样式，并要求配置严格的工程 rules。
- **AI 动作**：在 Plan 模式下起草了初步的《开发计划与架构设计》。
- **决策/变更**：用户多次反馈修改意向，要求强调“小白友好、可复现、枯燥概念加代码”的核心生成逻辑。

### 对话 2：文档共创流程引入 (Doc Co-Authoring)
- **用户需求**：用户拒绝将所有计划写在一个文件里，要求采用“结构化共创流程”，逐个拆分出产品需求(PRD)、数据库、架构文档。
- **AI 动作**：启动 `doc-coauthoring` 技能，通过多轮提问，与用户依次完成了：
  1. **PRD**：确立 5000 字拆分限制、50MB 体积限制、阅后即焚策略。
  2. **Architecture**：确立 Monorepo、SSE 流推送、并发调度策略及 JWT+OAuth 体系。
  3. **Database**：确立了父子系列博客关联 (`parent_id`) 及雪花 ID 策略。
- **决策/变更**：
  - 架构设计过程中，用户发现缺少“账号体系”，临时决定引入 GitHub/Wechat 第三方 OAuth 一键登录。
  - 数据库选型阶段，用户要求**将 MySQL 替换为 PostgreSQL 14+**，并采用 `TEXT` 和 `UUID` 类型。

### 对话 3：补充核心基准文档与重构 Rules
- **用户需求**：发现缺少前后端 API 定义，要求补全。随后要求将 `.cursorrules` 按规则类型拆分，并强调自己极度偏好在 Vibe Coding 中使用**共创模式**。
- **AI 动作**：
  - 补充了第 4 份文档《API 接口规范文档》。
  - 将 `.cursorrules` 删除，拆分为 `.trae/rules/` 下的三个 Markdown 文件（架构规范、业务约束、Vibe协作规范）。
  - 在规则中确立了**“强制阅读上下文 (Context First)”**的第一铁律。

### 对话 4：开发计划与里程碑
- **用户需求**：基于已有文档，要求给出开发计划和每日开发日志。
- **AI 动作**：使用共创模式编写了《开发计划与日志》，划分了 MVP、Alpha、Beta、V1.0 四个里程碑，并对各模块进行了耗时预估和强单测约束。

### 对话 5：创建对话日志文件 
- **用户需求**：在 `.trae/documents/` 目录下添加一个文件，专门用于简单记录每次的对话。
- **AI 动作**：创建了本文件 (`InkWords_Conversation_Log.md`)，并回溯记录了今日从项目启动到脚手架搭建的完整对话脉络与关键决策点。

### 对话 6：完成 MVP 骨架与基建开发
- **用户需求**：要求继续按照开发计划执行。
- **AI 动作**：
  1. 使用 Spec 模式执行并完成了“基础架构与数据库搭建”(Task 1, 2)
  2. 使用 Spec 模式执行并完成了“第三方 OAuth2.0 与 JWT 签发”(Task 3)
  3. 使用 Spec 模式执行并完成了“基础 PDF/MD 解析器”(Task 4)
- **决策/变更**：
  - 遇到 PDF 流解析问题，决定采用生成临时文件然后 `defer os.Remove` 的策略，确保兼顾 `io.ReaderAt` 接口需求与“阅后即焚”的安全需求。

### 对话 7：封装 DeepSeek 与 SSE 流式推流通道 (当前)
- **用户需求**：继续开发“封装 DeepSeek 客户端，建立前后端 SSE 实时推流渲染通道”功能。
- **AI 动作**：
  1. 编写了前后端 `spec.md` 及相关任务，并执行开发。
  2. 封装了 `DeepSeekClient` 以及 Gin 的 SSE 推流接口 `/api/v1/stream/generate`。
  3. 前端集成了 Zustand、`fetch-event-source`，以及基于 `react-markdown` 的视图组件。
  4. 利用自建 Remark 插件移除了 Mermaid 图表代码块内的样式注入。
  5. 自动更新了 `InkWords_API.md`、`InkWords_Architecture.md` 和 `InkWords_PRD.md`，随后又将开发记录同步至 `InkWords_Development_Plan_and_Log.md` 和本文件。
- **决策/变更**：
  - 因为原生 EventSource 不支持 POST 请求，且基于 Ticket GET 状态拉取长文本设计过于复杂，**更改架构决策**为前端携带提取出的纯文本（`source_content`），直接向后端发起 `POST` 请求获取 SSE 数据流（使用 `@microsoft/fetch-event-source`）。

### 对话 8：阶段二 Alpha (大项目智能拆解) 开发完成
- **用户需求**：要求根据开发计划进行阶段二的开发（大项目智能拆解），并在完成后更新所有架构与设计文档。
- **AI 动作**：
  1. 查阅了 `doc-coauthoring` 和 `find-skills` 技能说明。
  2. 使用 Spec 模式编写了 `05-project-decomposition` 的规范、任务与检查单。
  3. 实现了 `GitFetcher`，支持 Git 仓库的拉取（`git clone --depth 1`）、过滤与文本提取，并确保阅后即焚。
  4. 实现了 `DecompositionService` 和 `POST /api/v1/project/analyze` 路由，支持长文本的非流式大纲生成。
  5. 实现了基于 Goroutine 池的并发调度生成机制，确保同一系列文章能够并行生成并正确落库（绑定相同的 `ParentID`）。
  6. 更新了 SSE 推流接口，支持接收大纲并下发各个章节的进度事件。
  7. 批量更新了所有 `.trae/documents` 下的基础文档并提交至 GitHub。
- **决策/变更**：
  - 采用 Goroutine 池并结合信号量 (Semaphore) 控制并发量，防止大量并发请求导致大模型 API 频率超限或内存激增。
  - SSE 推流在系列生成模式下，不再单纯推送 chunk，而是推送每个章节的整体生成状态，以便前端展示全局进度。

### 对话 9：前端工作台 (Workspace) UI 搭建与联调
- **用户需求**：要求启动项目并解决初始化的 React 空白页面问题，构建实际的前端工作台。
- **AI 动作**：
  1. 诊断并修复了因缺失数据库导致的后端启动失败问题（创建 `inkwords` 数据库并初始化 `.env`）。
  2. 开发了前端双栏布局界面，左侧边栏用于展示大项目大纲与实时进度（Loading / 完成图标），右侧用于输入 Git URL 和触发分析与生成。
  3. 完善了 Zustand 状态库 `streamStore.ts` 以及自定义 Hooks `useBlogStream.ts`，打通了前后端的 `Analyze` 与 `Generate` 接口。
  4. 修复了 Vite 预构建缓存导致 `lucide-react` 找不到的 404 错误（使用 `npm run dev -- --force` 重启）。
  5. 修复了 `MarkdownEngine` 中的 ESLint 报错，并清理了冗余代码。
- **决策/变更**：
  - 在 `vite.config.ts` 中配置了跨域代理，将前端 `/api` 转发到后端 `8080` 端口，避免了开发环境的 CORS 问题。

---

## 2026-04-06 架构踩坑与修复记录

### 1. GitHub OAuth 授权挂起与资源泄露问题
**背景**：国内网络环境下，后端服务器访问 `github.com` API 经常超时，导致浏览器长时间等待后主动切断连接 (`net::ERR_ABORTED`)。但在原先实现中，由于请求使用了 `context.Background()`，即便浏览器已经断开，Go 后端的请求协程依然挂起并继续等待，白白消耗系统资源。
**决策与解决**：将 Gin 的 `c.Request.Context()` 透传至底层的 OAuth2 `Exchange` 请求中。一旦浏览器主动断连，Go 服务能瞬间捕获到 context 取消信号，立即终止挂起的 HTTP 请求并回收资源。

### 2. SSE 长连接代理超时中断问题 (net::ERR_ABORTED)
**背景**：在调用 DeepSeek 模型生成长篇博客或使用 Git Clone 拉取大型仓库时，由于耗时往往超过几分钟，导致 Nginx / Vite 的默认 Proxy（通常闲置 120 秒超时）会主动切断 SSE 连接，导致前端抛出中止错误。
**决策与解决**：
- 后端 (`stream.go`)：在所有的 SSE 接口中增加 15 秒级别的心跳机制（`<-time.After(15 * time.Second)`），在没有数据返回的真空期主动发送 `event: ping` 伪数据。
- 前端 (`vite.config.ts`)：显式增加代理的 `timeout: 120000` 并在前端捕获和处理特定的 `StopStreamError`，自动丢弃 `ping` 包并平滑处理意外断连。

### 3. 大模型 Token 长度限制与自动无缝续写
**背景**：DeepSeek 单次输出的最大 Token 数量有严格限制（通常为 4k - 8k），生成长文极易被截断，且直接暴露给前端用户体验极差。
**决策与解决**：
- 底层重构 (`deepseek.go`)：在解析 SSE chunk 时检测 `FinishReason`，将其透传回业务层。
- 业务层重构 (`generator.go` & `decomposition.go`)：引入内部自动重试与续写循环。如果检测到模型输出是因为达到长度上限（`length`）而终止，业务层会自动追加提示词（“刚才你的回答被截断了，请严格从上文最后一个字符开始无缝续写...”），并再次发起请求。这些追加的内容将无缝拼接后继续通过外层的 SSE 管道推流给前端。用户侧完全感知不到模型的截断限制。

### 对话 10：前端文章删除的二次确认与竞态Bug修复
- **用户需求**：修复点击删除按钮后未等待确认即删除文章的 Bug，且二次反馈确认框弹出时文章已被删除。
- **AI 动作**：
  1. 在 `Generator.tsx`（章节删除）和 `Sidebar.tsx`（批量删除）中为删除按钮显式增加 `type="button"` 和 `window.confirm`。
  2. 诊断出事件冒泡及并发连点（双击）导致数据先行删除的竞态条件（Race Condition）。
  3. 在 `Sidebar.tsx` 中引入 `useRef` 级别的确认锁 (`isConfirmingRef`) 拦截排队的确认框。
  4. 在 `Generator.tsx` 章节删除中增加 `e.detail > 1` 拦截，从浏览器底层原生事件阻断双击/多击。
  5. 加入 `e.preventDefault()` 和 `e.stopPropagation()` 阻断事件冒泡。
- **决策/变更**：
  - 前端涉及破坏性操作（如删除）的 `onClick` 事件，不仅需要 `window.confirm`，还必须考虑 React 异步状态更新期间用户连续点击造成的重复触发问题，通过 `useRef` 和原生 `e.detail` 进行双重防御。
