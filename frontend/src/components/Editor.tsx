import { useState, useEffect, useRef } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { MarkdownEngine } from './MarkdownEngine'
import { Button } from './ui/button'
import { Download, FileDown, Save, Loader2, Sparkles, FileArchive, ChevronDown } from 'lucide-react'
import { useDebounce } from '@/hooks/useDebounce'
import { useSyncedScroll } from '@/hooks/useSyncedScroll'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { toast } from 'sonner'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

class StopStreamError extends Error {}

export function Editor() {
  const { selectedBlog, updateBlog } = useBlogStore()
  const [title, setTitle] = useState(selectedBlog?.title || '')
  const [content, setContent] = useState(selectedBlog?.content || '')
  const [isSaving, setIsSaving] = useState(false)
  const [isContinuing, setIsContinuing] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)

  const { editorRef, previewRef, handleEditorScroll, handlePreviewScroll } = useSyncedScroll(content)

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
    const token = localStorage.getItem('token')
    
    try {
      toast.loading('正在打包系列博客...', { id: 'export-zip' })
      const res = await fetch(`/api/v1/blogs/${selectedBlog.id}/export`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      if (!res.ok) throw new Error('Export failed')
      
      const blob = await res.blob()
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
    if (!selectedBlog?.id) return;
    try {
      toast.loading('正在同步单篇到 Obsidian...', { id: 'export-obsidian' })
      const response = await fetch(`/api/v1/blogs/${selectedBlog.id}/export/obsidian`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      if (data.code === 200) {
        toast.success('成功同步单篇到 Obsidian 仓库', { id: 'export-obsidian' });
      } else {
        toast.error(data.message || '同步失败', { id: 'export-obsidian' });
      }
    } catch (error) {
      console.error('Export error:', error);
      toast.error('同步时发生网络错误', { id: 'export-obsidian' });
    }
  };

  const handleExportSeriesToObsidian = async () => {
    if (!selectedBlog?.id) return;
    try {
      toast.loading('正在同步整个系列到 Obsidian...', { id: 'export-series-obsidian' })
      const response = await fetch(`/api/v1/blogs/${selectedBlog.id}/export/obsidian/series`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      if (data.code === 200) {
        toast.success('成功同步系列到 Obsidian 仓库，已建立知识网络', { id: 'export-series-obsidian' });
      } else {
        toast.error(data.message || '同步系列失败', { id: 'export-series-obsidian' });
      }
    } catch (error) {
      console.error('Export series error:', error);
      toast.error('同步系列时发生网络错误', { id: 'export-series-obsidian' });
    }
  };

  const handleContinueGenerating = async () => {
    if (!selectedBlog || isContinuing) return
    setIsContinuing(true)

    const token = localStorage.getItem('token')

    try {
      let currentContent = content

      await fetchEventSource(`/api/v1/blogs/${selectedBlog.id}/continue`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: JSON.stringify({}),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            currentContent += msg.data
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

        <div className="flex items-center gap-3">
          <Button 
            variant="outline" 
            size="sm" 
            onClick={handleContinueGenerating} 
            disabled={isContinuing}
            className="gap-1.5 text-indigo-600 border-indigo-200 hover:bg-indigo-50 transition-all duration-200"
          >
            {isContinuing ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Sparkles className="w-4 h-4" />
            )}
            继续生成
          </Button>
          
          <DropdownMenu>
            <DropdownMenuTrigger render={
              <Button variant="outline" size="sm" className="gap-1.5 text-zinc-700 hover:text-zinc-900 transition-all duration-200 shadow-sm">
                <Download className="w-4 h-4" /> 
                导出 / 同步
                <ChevronDown className="w-3 h-3 opacity-50" />
              </Button>
            } />
            <DropdownMenuContent align="end" className="w-56 shadow-xl border-zinc-200/60 rounded-xl p-1">
              <DropdownMenuItem onClick={handleExportToObsidian} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg py-2">
                <div className="bg-indigo-50 text-indigo-600 p-1.5 rounded-md">
                  <Sparkles className="w-3.5 h-3.5" />
                </div>
                <div className="flex flex-col">
                  <span className="font-medium text-zinc-900">同步单篇到 Obsidian</span>
                  <span className="text-xs text-zinc-500">直通本地第二大脑</span>
                </div>
              </DropdownMenuItem>

              {selectedBlog?.parent_id === '00000000-0000-0000-0000-000000000000' && selectedBlog?.children && selectedBlog.children.length > 0 && (
                <DropdownMenuItem onClick={handleExportSeriesToObsidian} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg py-2 mt-1">
                  <div className="bg-indigo-50 text-indigo-600 p-1.5 rounded-md">
                    <FileArchive className="w-3.5 h-3.5" />
                  </div>
                  <div className="flex flex-col">
                    <span className="font-medium text-zinc-900">同步整个系列到 Obsidian</span>
                    <span className="text-xs text-zinc-500">自动构建双链知识网络</span>
                  </div>
                </DropdownMenuItem>
              )}
              
              <DropdownMenuSeparator className="bg-zinc-100 my-1" />
              
              {selectedBlog?.parent_id === '00000000-0000-0000-0000-000000000000' && selectedBlog?.children && selectedBlog.children.length > 0 && (
                <DropdownMenuItem onClick={exportSeriesZip} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
                  <FileArchive className="w-4 h-4 text-zinc-500" />
                  <span>导出系列 ZIP</span>
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={exportMarkdown} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
                <FileDown className="w-4 h-4 text-zinc-500" />
                <span>导出为 Markdown</span>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={exportPDF} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
                <Download className="w-4 h-4 text-zinc-500" />
                <span>打印为 PDF</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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
            <MarkdownEngine content={content} />
          </div>
        </div>
      </div>
    </div>
  )
}
