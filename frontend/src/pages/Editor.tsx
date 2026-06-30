import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { toast } from 'sonner'
import { useSyncedScroll } from '@/hooks/useSyncedScroll'
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition'
import { replaceVoiceSegment } from '@/lib/voiceInsertion'
import { usePolishStream } from '@/hooks/usePolishStream'
import { extractPolishedBody } from '@/lib/polishDraft'
import { normalizeMarkdown } from '@/lib/markdownNormalize'
import { useContinueStream } from '@/hooks/useContinueStream'
import { useEditorAutosave } from '@/hooks/useEditorAutosave'
import { blogService } from '@/services/blog'
import { EditorHeader } from '@/components/editor/EditorHeader'
import { EditorBody } from '@/components/editor/EditorBody'

export function Editor() {
  const { selectedBlog, updateBlog } = useBlogStore()
  const [title, setTitle] = useState(selectedBlog?.title || '')
  const [content, setContent] = useState(selectedBlog?.content || '')
  const [activePreviewTab, setActivePreviewTab] = useState<'preview' | 'polish'>('preview')

  const { isSaving, lastSaved } = useEditorAutosave({
    selectedBlog,
    title,
    content,
    updateBlog,
  })

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

  const { isContinuing, handleContinueGenerating } = useContinueStream({
    blogId: selectedBlog?.id ?? '',
    getContent: () => content,
    setContent,
    onDone: (finalContent) => {
      if (selectedBlog) {
        updateBlog(selectedBlog.id, { content: finalContent })
      }
    },
  })

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

  // Why: 切换博客时清空润色预览状态，避免旧博客的润色草稿残留
  // setActivePreviewTab 是切换博客后视觉状态重置的必要副作用
  useEffect(() => {
    cancelPolishAndClear()
    queueMicrotask(() => setActivePreviewTab('preview'))
  }, [selectedBlog?.id, cancelPolishAndClear])

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

  // 继续生成的触发：额外检查语音和润色状态
  const handleTriggerContinue = useCallback(() => {
    if (!selectedBlog || isVoiceListening || isPolishing) return
    handleContinueGenerating()
  }, [selectedBlog, isVoiceListening, isPolishing, handleContinueGenerating])

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
        onContinueGenerating={handleTriggerContinue}
        onExportToObsidian={handleExportToObsidian}
        onExportSeriesToObsidian={handleExportSeriesToObsidian}
        onExportSeriesZip={exportSeriesZip}
        onExportMarkdown={exportMarkdown}
        onExportPDF={() => window.print()}
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
