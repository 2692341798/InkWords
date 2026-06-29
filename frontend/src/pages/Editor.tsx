import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { toast } from 'sonner'
import { useDebounce } from '@/hooks/useDebounce'
import { useSyncedScroll } from '@/hooks/useSyncedScroll'
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition'
import { replaceVoiceSegment } from '@/lib/voiceInsertion'
import { usePolishStream } from '@/hooks/usePolishStream'
import { extractPolishedBody } from '@/lib/polishDraft'
import { normalizeMarkdown } from '@/lib/markdownNormalize'
import { fetchEventSourceWithAuth } from '@/services/sse'
import { blogService } from '@/services/blog'
import {
  buildContinueTaskPayload,
  buildGenerationTaskRequest,
  createGenerationTask,
  extractTaskChunkContent,
} from '@/services/generationTasks'
import { EditorHeader } from '@/components/editor/EditorHeader'
import { EditorBody } from '@/components/editor/EditorBody'

class StopStreamError extends Error {}

export function Editor() {
  const { selectedBlog, updateBlog } = useBlogStore()
  const [title, setTitle] = useState(selectedBlog?.title || '')
  const [content, setContent] = useState(selectedBlog?.content || '')
  const [isSaving, setIsSaving] = useState(false)
  const [isContinuing, setIsContinuing] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [activePreviewTab, setActivePreviewTab] = useState<'preview' | 'polish'>('preview')

  const {
    isPolishing,
    draft: polishedDraft,
    start: startPolish,
    cancelAndClear: cancelPolishAndClear,
  } = usePolishStream()

  const normalizedPolishedDraft = useMemo(() => normalizeMarkdown(polishedDraft), [polishedDraft])

  const { editorRef, previewRef, handleEditorScroll, handlePreviewScroll } = useSyncedScroll(content)

  const voiceSessionRef = useRef({
    isActive: false,
    anchorStart: 0,
    anchorEnd: 0,
    lastInsertedLength: 0,
  })

  const normalizeVoiceText = useCallback((text: string) => {
    return text.replace(/\s+/g, ' ')
  }, [])

  const moveCaretToVoiceEnd = useCallback(() => {
    requestAnimationFrame(() => {
      const textarea = editorRef.current
      if (!textarea) return
      const session = voiceSessionRef.current
      const pos = session.anchorStart + session.lastInsertedLength
      textarea.focus()
      textarea.setSelectionRange(pos, pos)
    })
  }, [editorRef])

  const applyVoiceText = useCallback(
    (rawText: string, isFinal: boolean) => {
      const session = voiceSessionRef.current
      if (!session.isActive) return

      const normalized = normalizeVoiceText(rawText)
      if (!normalized) return

      const nextText = isFinal ? `${normalized} ` : normalized

      setContent((prev) => {
        const res = replaceVoiceSegment({
          base: prev,
          anchorStart: session.anchorStart,
          anchorEnd: session.anchorEnd,
          lastInsertedLength: session.lastInsertedLength,
          nextText,
        })

        session.lastInsertedLength = res.newInsertedLength

        if (isFinal) {
          session.anchorStart = session.anchorStart + res.newInsertedLength
          session.anchorEnd = session.anchorStart
          session.lastInsertedLength = 0
        }

        return res.merged
      })

      if (!isFinal) {
        moveCaretToVoiceEnd()
      } else {
        requestAnimationFrame(() => {
          const textarea = editorRef.current
          if (!textarea) return
          const session = voiceSessionRef.current
          textarea.focus()
          textarea.setSelectionRange(session.anchorStart, session.anchorStart)
        })
      }
    },
    [moveCaretToVoiceEnd, normalizeVoiceText, editorRef],
  )

  const voiceCallbacks = useMemo(
    () => ({
      onInterimText: (text: string) => applyVoiceText(text, false),
      onFinalText: (text: string) => applyVoiceText(text, true),
      onErrorText: (message: string) => {
        voiceSessionRef.current.isActive = false
        voiceSessionRef.current.lastInsertedLength = 0
        toast.error(message)
      },
    }),
    [applyVoiceText],
  )

  const { isSupported: isVoiceSupported, isListening: isVoiceListening, start: startVoice, stop: stopVoice } =
    useSpeechRecognition(voiceCallbacks)

  const handleToggleVoiceInput = useCallback(() => {
    if (isContinuing || isPolishing) return

    if (!isVoiceSupported) {
      toast.error('当前浏览器不支持语音输入，请使用 Chrome/Edge')
      return
    }

    if (isVoiceListening) {
      voiceSessionRef.current.isActive = false
      voiceSessionRef.current.lastInsertedLength = 0
      stopVoice()
      return
    }

    const textarea = editorRef.current
    if (!textarea) {
      toast.error('正文输入框未就绪')
      return
    }

    voiceSessionRef.current = {
      isActive: true,
      anchorStart: textarea.selectionStart ?? 0,
      anchorEnd: textarea.selectionEnd ?? textarea.selectionStart ?? 0,
      lastInsertedLength: 0,
    }

    startVoice()
  }, [isContinuing, isPolishing, isVoiceListening, isVoiceSupported, startVoice, stopVoice, editorRef])

  useEffect(() => {
    cancelPolishAndClear()
    setActivePreviewTab('preview')
  }, [selectedBlog?.id, cancelPolishAndClear])

  // Track latest state for unmount save
  const currentStateRef = useRef({ selectedBlog, title, content })
  useEffect(() => {
    currentStateRef.current = { selectedBlog, title, content }
  }, [selectedBlog, title, content])

  // Save on unmount if there are unsaved changes
  useEffect(() => {
    return () => {
      const { selectedBlog: b, title: t, content: c } = currentStateRef.current
      if (b && (t !== b.title || c !== b.content)) {
        updateBlog(b.id, { title: t, content: c })
      }
    }
  }, [updateBlog])

  // Debounced values for auto-saving
  const debouncedTitle = useDebounce(title, 2000)
  const debouncedContent = useDebounce(content, 2000)

  useEffect(() => {
    if (selectedBlog && (debouncedTitle !== selectedBlog.title || debouncedContent !== selectedBlog.content)) {
      const save = async () => {
        setIsSaving(true)
        try {
          await updateBlog(selectedBlog.id, {
            title: debouncedTitle,
            content: debouncedContent
          })
          setLastSaved(new Date())
        } finally {
          setIsSaving(false)
        }
      }
      save()
    }
  }, [debouncedTitle, debouncedContent, selectedBlog, updateBlog])

  const handleStartPolish = useCallback(() => {
    if (!selectedBlog) return
    if (isContinuing || isVoiceListening || isPolishing) return
    if (!content.trim()) {
      toast.error('正文为空，无法润色')
      return
    }
    setActivePreviewTab('polish')
    startPolish(selectedBlog.id, title, content)
  }, [content, isContinuing, isPolishing, isVoiceListening, selectedBlog, startPolish, title])

  const handleCancelPolish = useCallback(() => {
    cancelPolishAndClear()
    setActivePreviewTab('preview')
  }, [cancelPolishAndClear])

  const handleRetryPolish = useCallback(() => {
    if (!selectedBlog) return
    if (isContinuing || isVoiceListening) return
    if (!content.trim()) {
      cancelPolishAndClear()
      toast.error('正文为空，无法润色')
      return
    }
    cancelPolishAndClear()
    setActivePreviewTab('polish')
    startPolish(selectedBlog.id, title, content)
  }, [cancelPolishAndClear, content, isContinuing, isVoiceListening, selectedBlog, startPolish, title])

  const handleApplyPolish = useCallback(() => {
    const draft = normalizedPolishedDraft.trim()
    if (!draft) {
      toast.error('润色草稿为空')
      return
    }
    setContent(extractPolishedBody(draft))
    cancelPolishAndClear()
    setActivePreviewTab('preview')
    toast.success('已应用润色结果')
  }, [cancelPolishAndClear, normalizedPolishedDraft])

  if (!selectedBlog) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-zinc-400">
        <p>请在左侧选择一篇博客以进行编辑</p>
      </div>
    )
  }

  const exportMarkdown = () => {
    try {
      const blob = new Blob([`# ${title}\n\n${content}`], { type: 'text/markdown;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${title || '未命名博客'}.md`
      a.click()
      URL.revokeObjectURL(url)
      toast.success('成功导出 Markdown 文件')
    } catch {
      toast.error('导出失败')
    }
  }

  const exportSeriesZip = async () => {
    if (!selectedBlog) return

    try {
      toast.loading('正在打包系列博客...', { id: 'export-zip' })
      const blob = await blogService.exportSeriesZip(selectedBlog.id)
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${title || 'series'}.zip`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
      toast.success('成功导出系列 ZIP', { id: 'export-zip' })
    } catch (err) {
      console.error('Failed to export zip:', err)
      toast.error('导出系列失败', { id: 'export-zip' })
    }
  }

  const exportPDF = () => {
    window.print()
  }

  const handleExportToObsidian = async () => {
    if (!selectedBlog?.id) return
    try {
      toast.loading('正在同步单篇到 Obsidian...', { id: 'export-obsidian' })
      await blogService.exportBlogToObsidian(selectedBlog.id)
      toast.success('成功同步单篇到 Obsidian 仓库', { id: 'export-obsidian' })
    } catch (error) {
      console.error('Export error:', error)
      toast.error('同步时发生网络错误', { id: 'export-obsidian' })
    }
  }

  const handleExportSeriesToObsidian = async () => {
    if (!selectedBlog?.id) return
    try {
      toast.loading('正在同步整个系列到 Obsidian...', { id: 'export-series-obsidian' })
      await blogService.exportSeriesToObsidian(selectedBlog.id)
      toast.success('成功同步系列到 Obsidian 仓库，已建立知识网络', { id: 'export-series-obsidian' })
    } catch (error) {
      console.error('Export series error:', error)
      toast.error('同步系列时发生网络错误', { id: 'export-series-obsidian' })
    }
  }

  const handleContinueGenerating = async () => {
    if (!selectedBlog || isContinuing || isVoiceListening || isPolishing) return
    setIsContinuing(true)

    try {
      let currentContent = content

      const task = await createGenerationTask(
        buildGenerationTaskRequest('continue', buildContinueTaskPayload(selectedBlog.id)),
      )

      await fetchEventSourceWithAuth(task.stream_url, {
        method: 'GET',
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json()
            throw new StopStreamError(data.error || '请求失败')
          }
          const text = await response.text()
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            currentContent += extractTaskChunkContent(msg.data)
            setContent(currentContent)
          } else if (msg.event === 'done') {
            updateBlog(selectedBlog.id, { content: currentContent })
            setIsContinuing(false)
            throw new StopStreamError('done')
          } else if (msg.event === 'error') {
            console.error('Continue generating error:', msg.data)
            setIsContinuing(false)
            throw new StopStreamError(msg.data)
          }
        },
        onclose() {
          setIsContinuing(false)
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const maybeError = err as { name?: unknown; message?: unknown }
          const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
          const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
          if (name === 'AbortError' || message.includes('AbortError') || message.includes('aborted') || message.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('Continue generating fetch error:', err)
          setIsContinuing(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message === 'done' || err.message === 'aborted') {
          return
        }
      }
      const maybeError = err as { name?: unknown; message?: unknown }
      const name = typeof maybeError?.name === 'string' ? maybeError.name : ''
      const message = typeof maybeError?.message === 'string' ? maybeError.message : ''
      if (name === 'AbortError' || message.includes('AbortError') || message.includes('aborted')) return
      
      console.error('Failed to continue generating:', err)
      setIsContinuing(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col h-screen bg-white print:h-auto print:block">
      <EditorHeader
        selectedBlog={selectedBlog}
        title={title}
        onTitleChange={setTitle}
        isSaving={isSaving}
        lastSaved={lastSaved}
        isVoiceListening={isVoiceListening}
        isContinuing={isContinuing}
        isPolishing={isPolishing}
        onToggleVoiceInput={handleToggleVoiceInput}
        onStartPolish={handleStartPolish}
        onContinueGenerating={handleContinueGenerating}
        onExportToObsidian={handleExportToObsidian}
        onExportSeriesToObsidian={handleExportSeriesToObsidian}
        onExportSeriesZip={exportSeriesZip}
        onExportMarkdown={exportMarkdown}
        onExportPDF={exportPDF}
      />

      <EditorBody
        content={content}
        onContentChange={setContent}
        editorRef={editorRef}
        previewRef={previewRef}
        handleEditorScroll={handleEditorScroll}
        handlePreviewScroll={handlePreviewScroll}
        activePreviewTab={activePreviewTab}
        setActivePreviewTab={setActivePreviewTab}
        isPolishing={isPolishing}
        polishedDraft={polishedDraft}
        normalizedPolishedDraft={normalizedPolishedDraft}
        onApplyPolish={handleApplyPolish}
        onCancelPolish={handleCancelPolish}
        onRetryPolish={handleRetryPolish}
      />
    </div>
  )
}
