import { useState, useEffect, useRef } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { MarkdownEngine } from './MarkdownEngine'
import { Button } from './ui/button'
import { Download, FileDown, Save, Loader2 } from 'lucide-react'
import { useDebounce } from '@/hooks/useDebounce'

export function Editor() {
  const { selectedBlog, updateBlog } = useBlogStore()
  const [title, setTitle] = useState(selectedBlog?.title || '')
  const [content, setContent] = useState(selectedBlog?.content || '')
  const [isSaving, setIsSaving] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)

  const editorRef = useRef<HTMLTextAreaElement>(null)
  const previewRef = useRef<HTMLDivElement>(null)
  const activePaneRef = useRef<'editor' | 'preview' | null>(null)
  const scrollTimeoutRef = useRef<number | null>(null)

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

  if (!selectedBlog) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-zinc-400">
        <p>请在左侧选择一篇博客以进行编辑</p>
      </div>
    )
  }

  const exportMarkdown = () => {
    const blob = new Blob([`# ${title}\n\n${content}`], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${title || '未命名博客'}.md`
    a.click()
    URL.revokeObjectURL(url)
  }

  const exportPDF = () => {
    window.print()
  }

  const handleEditorScroll = () => {
    if (activePaneRef.current === 'preview') return;
    
    activePaneRef.current = 'editor';
    if (scrollTimeoutRef.current) clearTimeout(scrollTimeoutRef.current);
    scrollTimeoutRef.current = setTimeout(() => {
      activePaneRef.current = null;
    }, 50);

    const editor = editorRef.current;
    const preview = previewRef.current;
    if (!editor || !preview) return;

    if (editor.scrollTop <= 0) {
      preview.scrollTop = 0;
      return;
    }

    if (editor.scrollTop + editor.clientHeight >= editor.scrollHeight - 1) {
      preview.scrollTop = preview.scrollHeight - preview.clientHeight;
      return;
    }

    const scrollPercentage = editor.scrollTop / (editor.scrollHeight - editor.clientHeight);
    if (isNaN(scrollPercentage)) return;

    const elements = Array.from(preview.querySelectorAll('[data-source-line]')) as HTMLElement[];
    if (elements.length === 0) {
      preview.scrollTop = scrollPercentage * (preview.scrollHeight - preview.clientHeight);
      return;
    }

    const totalLines = content.split('\n').length || 1;
    const currentLine = scrollPercentage * totalLines;

    let prevElement = elements[0];
    let nextElement = elements[elements.length - 1];

    for (let i = 0; i < elements.length; i++) {
      const el = elements[i];
      const line = parseInt(el.getAttribute('data-source-line') || '0', 10);
      if (line <= currentLine) {
        prevElement = el;
      }
      if (line > currentLine) {
        nextElement = el;
        break;
      }
    }

    const prevLine = parseInt(prevElement.getAttribute('data-source-line') || '0', 10);
    const nextLine = parseInt(nextElement.getAttribute('data-source-line') || '0', 10);

    const getElementScrollTop = (el: HTMLElement) => {
      return el.getBoundingClientRect().top - preview.getBoundingClientRect().top + preview.scrollTop;
    };

    let targetScrollTop = 0;

    const firstLine = parseInt(elements[0].getAttribute('data-source-line') || '0', 10);
    const lastLine = parseInt(elements[elements.length - 1].getAttribute('data-source-line') || '0', 10);
    
    if (currentLine < firstLine) {
      const firstTop = getElementScrollTop(elements[0]);
      targetScrollTop = firstTop * (currentLine / firstLine);
    } else if (currentLine > lastLine) {
      const lastTop = getElementScrollTop(elements[elements.length - 1]);
      const maxScrollTop = preview.scrollHeight - preview.clientHeight;
      const remainingTop = maxScrollTop - lastTop;
      const remainingLines = totalLines - lastLine;
      const lineRatio = remainingLines > 0 ? (currentLine - lastLine) / remainingLines : 1;
      targetScrollTop = lastTop + remainingTop * lineRatio;
    } else if (prevLine === nextLine) {
      targetScrollTop = getElementScrollTop(prevElement);
    } else {
      const lineRatio = (currentLine - prevLine) / (nextLine - prevLine);
      const prevTop = getElementScrollTop(prevElement);
      const nextTop = getElementScrollTop(nextElement);
      targetScrollTop = prevTop + (nextTop - prevTop) * lineRatio;
    }

    preview.scrollTop = targetScrollTop;
  }

  const handlePreviewScroll = () => {
    if (activePaneRef.current === 'editor') return;
    
    activePaneRef.current = 'preview';
    if (scrollTimeoutRef.current) clearTimeout(scrollTimeoutRef.current);
    scrollTimeoutRef.current = setTimeout(() => {
      activePaneRef.current = null;
    }, 50);

    const editor = editorRef.current;
    const preview = previewRef.current;
    if (!editor || !preview) return;

    if (preview.scrollTop <= 0) {
      editor.scrollTop = 0;
      return;
    }

    if (preview.scrollTop + preview.clientHeight >= preview.scrollHeight - 1) {
      editor.scrollTop = editor.scrollHeight - editor.clientHeight;
      return;
    }

    const elements = Array.from(preview.querySelectorAll('[data-source-line]')) as HTMLElement[];
    if (elements.length === 0) {
      const scrollPercentage = preview.scrollTop / (preview.scrollHeight - preview.clientHeight);
      editor.scrollTop = scrollPercentage * (editor.scrollHeight - editor.clientHeight);
      return;
    }

    const getElementScrollTop = (el: HTMLElement) => {
      return el.getBoundingClientRect().top - preview.getBoundingClientRect().top + preview.scrollTop;
    };

    const currentScrollTop = preview.scrollTop;

    let prevElement = elements[0];
    let nextElement = elements[elements.length - 1];

    for (let i = 0; i < elements.length; i++) {
      const el = elements[i];
      const top = getElementScrollTop(el);
      if (top <= currentScrollTop) {
        prevElement = el;
      }
      if (top > currentScrollTop) {
        nextElement = el;
        break;
      }
    }

    const prevLine = parseInt(prevElement.getAttribute('data-source-line') || '0', 10);
    const nextLine = parseInt(nextElement.getAttribute('data-source-line') || '0', 10);
    const prevTop = getElementScrollTop(prevElement);
    const nextTop = getElementScrollTop(nextElement);

    let targetLine = prevLine;
    const firstTop = getElementScrollTop(elements[0]);
    const firstLine = parseInt(elements[0].getAttribute('data-source-line') || '0', 10);
    const lastTop = getElementScrollTop(elements[elements.length - 1]);
    const lastLine = parseInt(elements[elements.length - 1].getAttribute('data-source-line') || '0', 10);
    const totalLines = content.split('\n').length || 1;

    if (currentScrollTop < firstTop) {
      targetLine = firstTop > 0 ? firstLine * (currentScrollTop / firstTop) : 0;
    } else if (currentScrollTop > lastTop) {
      const maxScrollTop = preview.scrollHeight - preview.clientHeight;
      const remainingTop = maxScrollTop - lastTop;
      const remainingLines = totalLines - lastLine;
      const scrollRatio = remainingTop > 0 ? (currentScrollTop - lastTop) / remainingTop : 1;
      targetLine = lastLine + remainingLines * scrollRatio;
    } else if (prevTop !== nextTop) {
      const scrollRatio = Math.max(0, Math.min(1, (currentScrollTop - prevTop) / (nextTop - prevTop)));
      targetLine = prevLine + (nextLine - prevLine) * scrollRatio;
    }

    const scrollPercentage = targetLine / totalLines;

    editor.scrollTop = scrollPercentage * (editor.scrollHeight - editor.clientHeight);
  }

  return (
    <div className="flex-1 flex flex-col h-screen bg-white print:h-auto print:block">
      {/* Editor Header */}
      <div className="h-14 border-b border-zinc-200 flex items-center justify-between px-6 shrink-0 print:hidden">
        <div className="flex items-center gap-4 flex-1">
          <input
            type="text"
            className="text-lg font-semibold bg-transparent border-none focus:outline-none focus:ring-0 text-zinc-800 placeholder-zinc-400 w-1/2"
            placeholder="输入博客标题..."
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
          <div className="text-xs text-zinc-400 flex items-center gap-1">
            {isSaving ? (
              <>
                <Loader2 className="w-3 h-3 animate-spin" /> 保存中...
              </>
            ) : lastSaved ? (
              <>
                <Save className="w-3 h-3" /> 已保存 {lastSaved.toLocaleTimeString()}
              </>
            ) : null}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={exportMarkdown} className="gap-1.5 text-zinc-600">
            <Download className="w-4 h-4" /> 导出 MD
          </Button>
          <Button variant="outline" size="sm" onClick={exportPDF} className="gap-1.5 text-zinc-600">
            <FileDown className="w-4 h-4" /> 导出 PDF
          </Button>
        </div>
      </div>

      {/* Split Pane */}
      <div className="flex-1 flex overflow-hidden print:overflow-visible print:block">
        {/* Editor Pane */}
        <div className="flex-1 border-r border-zinc-200 flex flex-col print:hidden">
          <textarea
            ref={editorRef}
            onScroll={handleEditorScroll}
            className="flex-1 w-full p-6 resize-none bg-zinc-50 border-none focus:outline-none focus:ring-0 font-mono text-sm text-zinc-700 leading-relaxed"
            placeholder="使用 Markdown 开始编写您的博客..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            spellCheck={false}
          />
        </div>

        {/* Preview Pane */}
        <div 
          ref={previewRef}
          onScroll={handlePreviewScroll}
          className="flex-1 bg-white overflow-y-auto print:block print:w-full print:overflow-visible relative"
        >
          <div className="max-w-3xl mx-auto p-8 print:p-0">
            <h1 className="text-3xl font-bold text-zinc-900 mb-8">{title}</h1>
            <MarkdownEngine content={content} />
          </div>
        </div>
      </div>
    </div>
  )
}
