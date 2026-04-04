import { useState, useEffect } from 'react'
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

  // Update local state when selectedBlog changes
  useEffect(() => {
    if (selectedBlog) {
      setTitle(selectedBlog.title || '')
      setContent(selectedBlog.content || '')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedBlog?.id])

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
            className="flex-1 w-full p-6 resize-none bg-zinc-50 border-none focus:outline-none focus:ring-0 font-mono text-sm text-zinc-700 leading-relaxed"
            placeholder="使用 Markdown 开始编写您的博客..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            spellCheck={false}
          />
        </div>

        {/* Preview Pane */}
        <div className="flex-1 bg-white overflow-y-auto print:block print:w-full print:overflow-visible">
          <div className="max-w-3xl mx-auto p-8 print:p-0">
            <h1 className="text-3xl font-bold text-zinc-900 mb-8">{title}</h1>
            <MarkdownEngine content={content} />
          </div>
        </div>
      </div>
    </div>
  )
}
