import { useEffect, useRef } from 'react'

export function useSyncedScroll(content: string) {
  const editorRef = useRef<HTMLTextAreaElement>(null)
  const previewRef = useRef<HTMLDivElement>(null)
  const activePaneRef = useRef<'editor' | 'preview' | null>(null)
  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (scrollTimeoutRef.current) clearTimeout(scrollTimeoutRef.current)
    }
  }, [])

  const handleEditorScroll = () => {
    if (activePaneRef.current === 'preview') return

    activePaneRef.current = 'editor'
    if (scrollTimeoutRef.current) clearTimeout(scrollTimeoutRef.current)
    scrollTimeoutRef.current = setTimeout(() => {
      activePaneRef.current = null
    }, 50)

    const editor = editorRef.current
    const preview = previewRef.current
    if (!editor || !preview) return

    if (editor.scrollTop <= 0) {
      preview.scrollTop = 0
      return
    }

    if (editor.scrollTop + editor.clientHeight >= editor.scrollHeight - 1) {
      preview.scrollTop = preview.scrollHeight - preview.clientHeight
      return
    }

    const scrollPercentage = editor.scrollTop / (editor.scrollHeight - editor.clientHeight)
    if (isNaN(scrollPercentage)) return

    const elements = Array.from(preview.querySelectorAll('[data-source-line]')) as HTMLElement[]
    if (elements.length === 0) {
      preview.scrollTop = scrollPercentage * (preview.scrollHeight - preview.clientHeight)
      return
    }

    const totalLines = content.split('\n').length || 1
    const currentLine = scrollPercentage * totalLines

    let prevElement = elements[0]
    let nextElement = elements[elements.length - 1]

    for (let i = 0; i < elements.length; i++) {
      const el = elements[i]
      const line = parseInt(el.getAttribute('data-source-line') || '0', 10)
      if (line <= currentLine) {
        prevElement = el
      }
      if (line > currentLine) {
        nextElement = el
        break
      }
    }

    const prevLine = parseInt(prevElement.getAttribute('data-source-line') || '0', 10)
    const nextLine = parseInt(nextElement.getAttribute('data-source-line') || '0', 10)

    const getElementScrollTop = (el: HTMLElement) => {
      return el.getBoundingClientRect().top - preview.getBoundingClientRect().top + preview.scrollTop
    }

    let targetScrollTop = 0

    const firstLine = parseInt(elements[0].getAttribute('data-source-line') || '0', 10)
    const lastLine = parseInt(elements[elements.length - 1].getAttribute('data-source-line') || '0', 10)

    if (currentLine < firstLine) {
      const firstTop = getElementScrollTop(elements[0])
      targetScrollTop = firstTop * (currentLine / firstLine)
    } else if (currentLine > lastLine) {
      const lastTop = getElementScrollTop(elements[elements.length - 1])
      const maxScrollTop = preview.scrollHeight - preview.clientHeight
      const remainingTop = maxScrollTop - lastTop
      const remainingLines = totalLines - lastLine
      const lineRatio = remainingLines > 0 ? (currentLine - lastLine) / remainingLines : 1
      targetScrollTop = lastTop + remainingTop * lineRatio
    } else if (prevLine === nextLine) {
      targetScrollTop = getElementScrollTop(prevElement)
    } else {
      const lineRatio = (currentLine - prevLine) / (nextLine - prevLine)
      const prevTop = getElementScrollTop(prevElement)
      const nextTop = getElementScrollTop(nextElement)
      targetScrollTop = prevTop + (nextTop - prevTop) * lineRatio
    }

    preview.scrollTop = targetScrollTop
  }

  const handlePreviewScroll = () => {
    if (activePaneRef.current === 'editor') return

    activePaneRef.current = 'preview'
    if (scrollTimeoutRef.current) clearTimeout(scrollTimeoutRef.current)
    scrollTimeoutRef.current = setTimeout(() => {
      activePaneRef.current = null
    }, 50)

    const editor = editorRef.current
    const preview = previewRef.current
    if (!editor || !preview) return

    if (preview.scrollTop <= 0) {
      editor.scrollTop = 0
      return
    }

    if (preview.scrollTop + preview.clientHeight >= preview.scrollHeight - 1) {
      editor.scrollTop = editor.scrollHeight - editor.clientHeight
      return
    }

    const elements = Array.from(preview.querySelectorAll('[data-source-line]')) as HTMLElement[]
    if (elements.length === 0) {
      const scrollPercentage = preview.scrollTop / (preview.scrollHeight - preview.clientHeight)
      editor.scrollTop = scrollPercentage * (editor.scrollHeight - editor.clientHeight)
      return
    }

    const getElementScrollTop = (el: HTMLElement) => {
      return el.getBoundingClientRect().top - preview.getBoundingClientRect().top + preview.scrollTop
    }

    const currentScrollTop = preview.scrollTop

    let prevElement = elements[0]
    let nextElement = elements[elements.length - 1]

    for (let i = 0; i < elements.length; i++) {
      const el = elements[i]
      const top = getElementScrollTop(el)
      if (top <= currentScrollTop) {
        prevElement = el
      }
      if (top > currentScrollTop) {
        nextElement = el
        break
      }
    }

    const prevLine = parseInt(prevElement.getAttribute('data-source-line') || '0', 10)
    const nextLine = parseInt(nextElement.getAttribute('data-source-line') || '0', 10)
    const prevTop = getElementScrollTop(prevElement)
    const nextTop = getElementScrollTop(nextElement)

    let targetLine = prevLine
    const firstTop = getElementScrollTop(elements[0])
    const firstLine = parseInt(elements[0].getAttribute('data-source-line') || '0', 10)
    const lastTop = getElementScrollTop(elements[elements.length - 1])
    const lastLine = parseInt(elements[elements.length - 1].getAttribute('data-source-line') || '0', 10)
    const totalLines = content.split('\n').length || 1

    if (currentScrollTop < firstTop) {
      targetLine = firstTop > 0 ? firstLine * (currentScrollTop / firstTop) : 0
    } else if (currentScrollTop > lastTop) {
      const maxScrollTop = preview.scrollHeight - preview.clientHeight
      const remainingTop = maxScrollTop - lastTop
      const remainingLines = totalLines - lastLine
      const scrollRatio = remainingTop > 0 ? (currentScrollTop - lastTop) / remainingTop : 1
      targetLine = lastLine + remainingLines * scrollRatio
    } else if (prevTop !== nextTop) {
      const scrollRatio = Math.max(0, Math.min(1, (currentScrollTop - prevTop) / (nextTop - prevTop)))
      targetLine = prevLine + (nextLine - prevLine) * scrollRatio
    }

    const scrollPercentage = targetLine / totalLines

    editor.scrollTop = scrollPercentage * (editor.scrollHeight - editor.clientHeight)
  }

  return {
    editorRef,
    previewRef,
    handleEditorScroll,
    handlePreviewScroll,
  }
}

