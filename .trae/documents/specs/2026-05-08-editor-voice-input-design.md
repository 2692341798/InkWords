# InkWords 编辑器语音输入（浏览器转写）设计规格

## 1. 背景与目标

InkWords 当前的手写写作入口会进入双栏编辑器 [Editor.tsx](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/components/Editor.tsx)，正文输入基于受控 `textarea`，并带有：

- 2 秒防抖自动保存（Zustand `updateBlog`）
- AI 继续生成（SSE 追加内容）
- 编辑区/预览区滚动同步

本需求希望在“写博客”编辑页新增语音输入能力，提升口述写作效率。

参考（知识库）：
- [[concepts/前端组件体系：Editor 与 Markdown 渲染]]

### 1.1 目标（要做）

- 在编辑器中新增“语音输入”按钮，支持开始/停止语音识别。
- 语音识别结果以“语音打字”的形式实时写入正文（插入到光标处）。
- 不新增后端接口，不引入外部转写服务，优先使用浏览器语音识别能力。

### 1.2 非目标（不做）

- 不支持录音文件导入/上传转写。
- 不支持标题输入框语音转写（仅正文）。
- 不做语种切换、多语言 UI。

## 2. 方案选型

### 2.1 采用浏览器 SpeechRecognition / Web Speech API

- 通过 `window.SpeechRecognition` / `window.webkitSpeechRecognition` 使用浏览器内置语音识别。
- 语言固定为 `zh-CN`。

权衡：
- 优点：无需后端改动；上线快；本地 `http://localhost` 可用。
- 缺点：兼容性取决于浏览器（通常 Chrome/Edge 更好）；不同系统/浏览器的识别体验不一致。

## 3. 交互与 UI

### 3.1 按钮位置

在 Editor Header 右侧按钮区新增“语音输入”按钮，与现有“继续生成/导出”等按钮保持一致风格：

- 默认态：显示“语音输入”（麦克风图标）
- 录音中：显示“停止语音”（麦克风关闭图标/红色强调）

### 3.2 行为约束

- 当 AI 继续生成进行中（`isContinuing=true`）时，禁用语音输入按钮，避免同时对 `content` 进行双来源写入造成竞态。
- 当语音输入进行中（`isListening=true`）时，禁用“继续生成”按钮，避免同样的竞态。
- 若浏览器不支持 SpeechRecognition，则按钮可见但不可用，点击后用 toast 提示“当前浏览器不支持语音输入，请使用 Chrome/Edge”。

## 4. 数据流与状态设计

### 4.1 新增 Hook：useSpeechRecognition

新增 `frontend/src/hooks/useSpeechRecognition.ts`，封装识别生命周期与回调：

- `isSupported: boolean`
- `isListening: boolean`
- `start(): void`
- `stop(): void`
- `error: string | null`

内部行为要点：

- 兼容 `SpeechRecognition` 与 `webkitSpeechRecognition`。
- 设置：
  - `lang = 'zh-CN'`
  - `continuous = true`（仅手动停止）
  - `interimResults = true`（实时边说边写）
- 事件处理：
  - `onresult`：把识别结果拆分为 `interimText` 与 `finalText` 两类回调出去。
  - `onend`：如果仍处于 `isListening=true`（用户未手动停止），则自动 `start()` 以实现“只手动停止”策略。
  - `onerror`：终止并回调错误信息（同时置 `isListening=false`）。

### 4.2 Editor 内容插入模型（实时边说边写）

由于是“插入光标处 + 实时更新”，需要解决两点：

1. **用户开始说话时记录插入锚点**：在 `start()` 之前读取 `editorRef.current.selectionStart/End`，记为 `anchorStart/anchorEnd`。
2. **interim 持续更新时要可替换**：同一次语音会话中，interim 会反复变化；需要在内容里保留“可替换区间”，避免每次都向后追加导致重复文字。

推荐实现策略（文本区间替换）：

- 在 Editor 组件内维护一个 `voiceSessionRef`：
  - `anchorStart` / `anchorEnd`：开始识别时的选区
  - `lastInsertedLength`：上一次插入（interim+final）的长度
  - `isActive`：是否处于语音写入会话中
- 每次收到 `interimText`：
  - 从 `content` 中移除上次插入的临时区间（基于 `anchorStart` 与 `lastInsertedLength`）
  - 重新插入最新的 `interimText`（不落盘到后端，仍会被 2 秒防抖保存机制捕获）
  - 更新 `lastInsertedLength`
  - 同步更新 `textarea` 光标到插入区间末尾
- 每次收到 `finalText`：
  - 先按同样方式替换掉 interim（如果存在）
  - 再插入 `finalText`（可选择在末尾补一个空格/换行，默认补一个空格以利于继续口述）
  - 更新 `anchorStart` 为当前插入末尾，以便下一句继续追加在正确位置

边界处理：
- 若用户在语音输入期间手动移动光标或编辑内容：本版本以“语音会话锚点”为准继续替换，可能出现用户预期外的插入位置；可在后续迭代中加入“检测到用户手动编辑则自动停止语音输入”。

## 5. 错误处理与提示文案（中文）

- 无权限（用户拒绝麦克风）：toast 提示“未获得麦克风权限，请在浏览器设置中允许后重试”。
- 不支持：toast 提示“当前浏览器不支持语音输入，请使用 Chrome/Edge”。
- 识别异常：toast 提示“语音识别失败：<原因>”。
- 录音中状态：按钮文案/样式明确提示正在录音。

## 6. 验证方案（Definition of Done）

### 6.1 手工验证（必做）

在 `http://localhost` 打开编辑器：

- 点击“语音输入”后浏览器弹出权限申请；允许后开始识别。
- 口述一段文字，正文实时出现；停顿不会自动停止，除非手动点击“停止语音”。
- 点击“停止语音”后不再写入。
- 语音输入期间“继续生成”按钮不可用；反之亦然。
- 等待 2 秒，验证自动保存仍工作（刷新页面内容仍在）。
- 滚动同步仍可用（语音写入过程中不崩溃）。

### 6.2 构建验证

- 前端 `npm run build` 通过（或通过 Docker Compose 一键构建）。

## 7. 影响范围（预估文件）

- `frontend/src/components/Editor.tsx`：新增按钮与插入逻辑
- `frontend/src/hooks/useSpeechRecognition.ts`：新增

