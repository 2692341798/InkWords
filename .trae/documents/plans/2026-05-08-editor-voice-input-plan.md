# 编辑器语音输入（浏览器转写）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在写博客编辑器中增加“语音输入”按钮，使用浏览器 SpeechRecognition 实时转写并插入到正文光标处，且不影响自动保存、继续生成与滚动同步。

**Architecture:** 新增 `useSpeechRecognition` hook 封装 Web Speech API；在 `Editor.tsx` 增加“语音输入”按钮与一套“可替换区间”的实时插入算法，通过 `ref` 记录锚点与上次插入长度，保证 interim 反复更新不会重复堆叠。

**Tech Stack:** React + TypeScript + Zustand；`lucide-react` 图标；`sonner` toast。

---

## File Map

- Create: `frontend/src/hooks/useSpeechRecognition.ts`
- Modify: `frontend/src/components/Editor.tsx`
- (Later docs) Modify:
  - `.trae/documents/InkWords_Conversation_Log.md`
  - `.trae/documents/InkWords_Development_Plan_and_Log.md`
  - `.trae/documents/InkWords_Architecture.md`（如需要补充“前端编辑器增强”条目）

---

### Task 1: 新增 useSpeechRecognition Hook（浏览器语音识别封装）

**Files:**
- Create: `frontend/src/hooks/useSpeechRecognition.ts`

- [ ] Step 1: 定义类型与兼容层

```ts
type SpeechRecognitionCtor = new () => SpeechRecognition

function getSpeechRecognitionCtor(): SpeechRecognitionCtor | null {
  const w = window as unknown as {
    SpeechRecognition?: SpeechRecognitionCtor
    webkitSpeechRecognition?: SpeechRecognitionCtor
  }
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null
}
```

- [ ] Step 2: 实现 hook 对外 API

```ts
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

export type SpeechRecognitionCallbacks = {
  onInterimText?: (text: string) => void
  onFinalText?: (text: string) => void
  onErrorText?: (message: string) => void
}

export function useSpeechRecognition(callbacks: SpeechRecognitionCallbacks) {
  const ctor = useMemo(() => (typeof window === 'undefined' ? null : getSpeechRecognitionCtor()), [])
  const isSupported = !!ctor

  const recognitionRef = useRef<SpeechRecognition | null>(null)
  const shouldKeepListeningRef = useRef(false)

  const [isListening, setIsListening] = useState(false)

  const stop = useCallback(() => {
    shouldKeepListeningRef.current = false
    setIsListening(false)
    recognitionRef.current?.stop()
  }, [])

  const start = useCallback(() => {
    if (!ctor) {
      callbacks.onErrorText?.('当前浏览器不支持语音输入，请使用 Chrome/Edge')
      return
    }

    if (!recognitionRef.current) {
      const recognition = new ctor()
      recognition.lang = 'zh-CN'
      recognition.continuous = true
      recognition.interimResults = true

      recognition.onresult = (event) => {
        let interim = ''
        let final = ''

        for (let i = event.resultIndex; i < event.results.length; i++) {
          const result = event.results[i]
          const text = result?.[0]?.transcript ?? ''
          if (!text) continue
          if (result.isFinal) final += text
          else interim += text
        }

        if (interim) callbacks.onInterimText?.(interim)
        if (final) callbacks.onFinalText?.(final)
      }

      recognition.onerror = (event) => {
        shouldKeepListeningRef.current = false
        setIsListening(false)
        callbacks.onErrorText?.(`语音识别失败：${event.error}`)
      }

      recognition.onend = () => {
        if (shouldKeepListeningRef.current) {
          try {
            recognition.start()
          } catch {
          }
        } else {
          setIsListening(false)
        }
      }

      recognitionRef.current = recognition
    }

    shouldKeepListeningRef.current = true
    setIsListening(true)

    try {
      recognitionRef.current.start()
    } catch {
    }
  }, [callbacks, ctor])

  useEffect(() => stop, [stop])

  return { isSupported, isListening, start, stop }
}
```

- [ ] Step 3: 自检

Run: `cd frontend && npm run build`  
Expected: build 成功，无 TypeScript 报错。

---

### Task 2: Editor 增加语音按钮与“实时插入区间”算法

**Files:**
- Modify: `frontend/src/components/Editor.tsx`

- [ ] Step 1: 引入图标与 hook

在 `lucide-react` 引入 `Mic`/`MicOff`（或使用现有图标集合中的等价图标），并引入 `useSpeechRecognition`。

- [ ] Step 2: 定义 voice session ref 与插入工具函数

```ts
type VoiceSession = {
  isActive: boolean
  anchorStart: number
  anchorEnd: number
  lastInsertedLength: number
}

const voiceSessionRef = useRef<VoiceSession>({
  isActive: false,
  anchorStart: 0,
  anchorEnd: 0,
  lastInsertedLength: 0,
})

function normalizeInsertionText(text: string) {
  return text.replace(/\s+/g, ' ')
}
```

插入/替换函数（核心）：

```ts
function replaceVoiceSegment(params: {
  base: string
  anchorStart: number
  anchorEnd: number
  lastInsertedLength: number
  nextText: string
}) {
  const { base, anchorStart, anchorEnd, lastInsertedLength, nextText } = params
  const start = anchorStart
  const previousEnd = anchorStart + lastInsertedLength
  const before = base.slice(0, start)
  const after = base.slice(Math.max(anchorEnd, previousEnd))
  const merged = `${before}${nextText}${after}`
  return { merged, newInsertedLength: nextText.length }
}
```

- [ ] Step 3: 初始化锚点（开始识别时）

在 `start()` 前，从 `editorRef.current` 读取选区：

```ts
const textarea = editorRef.current
if (!textarea) return
voiceSessionRef.current = {
  isActive: true,
  anchorStart: textarea.selectionStart ?? 0,
  anchorEnd: textarea.selectionEnd ?? textarea.selectionStart ?? 0,
  lastInsertedLength: 0,
}
```

- [ ] Step 4: 处理 interim（实时写入）

```ts
const handleInterim = (raw: string) => {
  const text = normalizeInsertionText(raw)
  if (!text) return
  const session = voiceSessionRef.current
  if (!session.isActive) return

  setContent((prev) => {
    const { merged, newInsertedLength } = replaceVoiceSegment({
      base: prev,
      anchorStart: session.anchorStart,
      anchorEnd: session.anchorEnd,
      lastInsertedLength: session.lastInsertedLength,
      nextText: text,
    })
    session.lastInsertedLength = newInsertedLength
    return merged
  })
}
```

并在下一次渲染后把光标移动到插入末尾（用 `requestAnimationFrame`）：

```ts
requestAnimationFrame(() => {
  const textarea = editorRef.current
  if (!textarea) return
  const session = voiceSessionRef.current
  const pos = session.anchorStart + session.lastInsertedLength
  textarea.focus()
  textarea.setSelectionRange(pos, pos)
})
```

- [ ] Step 5: 处理 final（落定 + 追加空格 + 更新锚点）

```ts
const handleFinal = (raw: string) => {
  const text = `${normalizeInsertionText(raw)} `
  const session = voiceSessionRef.current
  if (!session.isActive) return

  setContent((prev) => {
    const { merged, newInsertedLength } = replaceVoiceSegment({
      base: prev,
      anchorStart: session.anchorStart,
      anchorEnd: session.anchorEnd,
      lastInsertedLength: session.lastInsertedLength,
      nextText: text,
    })

    session.lastInsertedLength = 0
    session.anchorStart = session.anchorStart + newInsertedLength
    session.anchorEnd = session.anchorStart
    return merged
  })
}
```

- [ ] Step 6: 停止识别时清理 session

```ts
voiceSessionRef.current.isActive = false
voiceSessionRef.current.lastInsertedLength = 0
```

- [ ] Step 7: 加入互斥禁用逻辑与中文 toast 提示

要求：
- `isContinuing` 时语音按钮禁用
- `isListening` 时“继续生成”按钮禁用
- 不支持或报错时 toast 提示（中文）

---

### Task 3: 验证（手工 + 构建）

**Files:**
- No new files

- [ ] Step 1: 通过 Docker Compose 一键重启并构建

Run (repo root):

```bash
docker compose down && docker compose up -d --build
```

Expected: 前后端与 Nginx 正常启动；通过 `http://localhost` 访问页面。

- [ ] Step 2: 手工验收清单

- 进入任意博客编辑页（手写草稿或历史博客）。
- 点击“语音输入”允许麦克风权限后，口述文字能实时写入正文。
- 停顿不会停止；只有手动点“停止语音”才停止。
- 语音输入期间“继续生成”不可点击；继续生成期间“语音输入”不可点击。
- 等待 2 秒后刷新页面，文字仍存在（防抖保存链路未破坏）。
- 滚动同步仍正常（至少不报错/不明显错位）。

---

### Task 4: 文档同步（Docs-as-Code）

**Files:**
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- (Optional) Modify: `.trae/documents/InkWords_Architecture.md`

- [ ] Step 1: 在 Conversation Log 追加本次决策要点与 spec/plan 链接
- [ ] Step 2: 在 Development Plan and Log 记录实现进度与验证方式

