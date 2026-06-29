import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader } from '@/components/ui/workspace'
import type { ReviewCardResponse, ReviewMode, ReviewNoteOption } from '@/services/review'

interface ReviewNotePickerProps {
  notes: ReviewNoteOption[]
  isLoading: boolean
  onSearch: (query: string) => Promise<void> | void
  onModeSync: (mode: ReviewMode) => void
  onSelect: (card: ReviewCardResponse) => Promise<void> | void
  onBack?: () => void
}

export function ReviewNotePicker({ notes, isLoading, onSearch, onModeSync, onSelect, onBack }: ReviewNotePickerProps) {
  const [query, setQuery] = useState('')

  return (
    <Panel className="p-6">
      <SectionHeader
        title="选择文章复习"
        description="只展示适合复习的概念卡，避免抽到索引页或空白页。"
      />
      <div className="mt-4 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div className="flex w-full flex-col gap-3 md:max-w-xl md:flex-row">
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="按标题关键字搜索，例如：并发"
            className="h-10 flex-1 rounded-lg border border-border bg-secondary/35 px-3 text-sm text-foreground outline-none transition focus:border-[var(--brand)] focus:bg-card"
          />
          <div className="flex gap-3">
            <Button variant="outline" onClick={() => onSearch(query)} disabled={isLoading}>
              搜索
            </Button>
            {onBack ? (
              <Button variant="outline" onClick={onBack}>
                返回入口
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      <div className="mt-5 space-y-3">
        {notes.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border bg-secondary/35 px-4 py-6 text-sm text-muted-foreground">
            还没有候选文章，点击上方搜索或先让系统拉取一批可复习主题。
          </div>
        ) : (
          notes.map((note) => (
            <article
              key={note.note_path}
              className="flex flex-col gap-3 rounded-xl border border-border bg-card px-4 py-4 md:flex-row md:items-center md:justify-between"
            >
              <div className="space-y-1">
                <p className="text-sm font-medium text-foreground">{note.title}</p>
                <p className="text-sm text-muted-foreground">{note.source_title || '未标注系列'}</p>
                <p className="text-xs text-muted-foreground">
                  {note.last_reviewed_at ? `最近复习：${new Date(note.last_reviewed_at).toLocaleString()}` : '最近复习：还没有记录'}
                </p>
              </div>
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={() => onModeSync(note.preferred_mode)}
                >
                  推荐模式：{note.preferred_mode === 'detailed_qa' ? '细致提问' : '轻提示复述'}
                </Button>
                <Button
                  onClick={() => {
                    onModeSync(note.preferred_mode)
                    void onSelect({
                      note_path: note.note_path,
                      title: note.title,
                      source_title: note.source_title,
                      review_reason: '这是你主动选择的一篇复习文章。',
                      estimated_minutes: 5,
                      available_modes: ['light_recall', 'detailed_qa'],
                    })
                  }}
                >
                  开始复习
                </Button>
              </div>
            </article>
          ))
        )}
      </div>
    </Panel>
  )
}
