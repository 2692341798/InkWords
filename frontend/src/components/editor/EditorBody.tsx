import { useDeferredValue, type RefObject } from 'react'
import { Button } from '../ui/button'
import { Loader2 } from 'lucide-react'
import { MarkdownEngine } from '../MarkdownEngine'

type EditorBodyProps = {
  content: string
  onContentChange: (value: string) => void
  editorRef: RefObject<HTMLTextAreaElement | null>
  previewRef: RefObject<HTMLDivElement | null>
  handleEditorScroll: () => void
  handlePreviewScroll: () => void
  activePreviewTab: 'preview' | 'polish'
  setActivePreviewTab: (value: 'preview' | 'polish') => void
  isPolishing: boolean
  polishedDraft: string
  normalizedPolishedDraft: string
  onApplyPolish: () => void
  onCancelPolish: () => void
  onRetryPolish: () => void
}

export function EditorBody(props: EditorBodyProps) {
  const {
    content,
    onContentChange,
    editorRef,
    previewRef,
    handleEditorScroll,
    handlePreviewScroll,
    activePreviewTab,
    setActivePreviewTab,
    isPolishing,
    polishedDraft,
    normalizedPolishedDraft,
    onApplyPolish,
    onCancelPolish,
    onRetryPolish,
  } = props

  const deferredContent = useDeferredValue(content)
  const deferredPolishedDraft = useDeferredValue(normalizedPolishedDraft)

  return (
    <div className="flex-1 flex overflow-hidden print:overflow-visible print:block bg-background">
      <div className="flex-1 border-r border-border flex flex-col print:hidden">
        <textarea
          ref={editorRef}
          onScroll={handleEditorScroll}
          className="flex-1 w-full p-8 resize-none bg-secondary/20 border-none focus:outline-none focus:ring-0 font-mono text-[15px] text-foreground leading-[1.8] custom-scrollbar"
          placeholder="使用 Markdown 开始编写您的博客..."
          value={content}
          onChange={(e) => onContentChange(e.target.value)}
          spellCheck={false}
        />
      </div>

      <div
        ref={previewRef}
        onScroll={handlePreviewScroll}
        className="flex-1 bg-background overflow-y-auto print:block print:w-full print:overflow-visible relative custom-scrollbar"
      >
        <div className="sticky top-0 z-10 border-b border-border/60 bg-background/80 backdrop-blur print:hidden">
          <div className="max-w-3xl mx-auto px-8 py-3 flex items-center justify-between gap-4">
            <div className="inline-flex rounded-lg bg-secondary p-1">
              <button
                type="button"
                onClick={() => setActivePreviewTab('preview')}
                className={
                  activePreviewTab === 'preview'
                    ? 'px-3 py-1.5 text-sm font-medium rounded-md bg-card shadow-sm text-foreground'
                    : 'px-3 py-1.5 text-sm font-medium rounded-md text-muted-foreground hover:text-foreground'
                }
              >
                预览
              </button>
              <button
                type="button"
                onClick={() => setActivePreviewTab('polish')}
                className={
                  activePreviewTab === 'polish'
                    ? 'px-3 py-1.5 text-sm font-medium rounded-md bg-card shadow-sm text-foreground flex items-center gap-2'
                    : 'px-3 py-1.5 text-sm font-medium rounded-md text-muted-foreground hover:text-foreground flex items-center gap-2'
                }
              >
                润色预览
                {isPolishing ? <Loader2 className="w-3 h-3 animate-spin" /> : null}
              </button>
            </div>

            {activePreviewTab === 'polish' ? (
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  onClick={onApplyPolish}
                  disabled={isPolishing || !polishedDraft.trim()}
                  className="gap-1.5 bg-emerald-600 hover:bg-emerald-600/90 text-white"
                >
                  应用润色结果
                </Button>
                <Button variant="outline" size="sm" onClick={onCancelPolish}>
                  取消
                </Button>
                <Button variant="outline" size="sm" onClick={onRetryPolish} disabled={isPolishing}>
                  重新润色
                </Button>
              </div>
            ) : null}
          </div>
        </div>

        <div className="max-w-3xl mx-auto p-8 print:p-0">
          <MarkdownEngine content={activePreviewTab === 'polish' ? deferredPolishedDraft : deferredContent} />
        </div>
      </div>
    </div>
  )
}
