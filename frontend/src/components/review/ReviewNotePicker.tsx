import { useState } from 'react'
import { Button } from '@/components/ui/button'
import type { ReviewCardResponse, ReviewMode, ReviewNoteOption } from '@/services/review'

interface ReviewNotePickerProps {
  notes: ReviewNoteOption[]
  isLoading: boolean
  onSearch: (query: string) => Promise<void> | void
  onModeSync: (mode: ReviewMode) => void
  onSelect: (card: ReviewCardResponse) => Promise<void> | void
}

export function ReviewNotePicker({ notes, isLoading, onSearch, onModeSync, onSelect }: ReviewNotePickerProps) {
  const [query, setQuery] = useState('')

  return (
    <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-zinc-900">选择文章复习</h2>
          <p className="mt-1 text-sm leading-6 text-zinc-600">只展示适合复习的概念卡，避免抽到索引页或空白页。</p>
        </div>
        <div className="flex w-full gap-3 md:max-w-xl">
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="按标题关键字搜索，例如：并发"
            className="h-10 flex-1 rounded-xl border border-zinc-200 bg-zinc-50 px-3 text-sm text-zinc-900 outline-none transition focus:border-indigo-300 focus:bg-white"
          />
          <Button variant="outline" onClick={() => onSearch(query)} disabled={isLoading}>
            搜索
          </Button>
        </div>
      </div>

      <div className="mt-5 space-y-3">
        {notes.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-6 text-sm text-zinc-500">
            还没有候选文章，点击上方搜索或先让系统拉取一批可复习主题。
          </div>
        ) : (
          notes.map((note) => (
            <article
              key={note.note_path}
              className="flex flex-col gap-3 rounded-2xl border border-zinc-200 px-4 py-4 md:flex-row md:items-center md:justify-between"
            >
              <div className="space-y-1">
                <p className="text-sm font-medium text-zinc-900">{note.title}</p>
                <p className="text-sm text-zinc-600">{note.source_title || '未标注系列'}</p>
                <p className="text-xs text-zinc-500">
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
    </section>
  )
}
