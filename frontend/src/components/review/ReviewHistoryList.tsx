import type { ReviewHistoryItem } from '@/services/review'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'

interface ReviewHistoryListProps {
  items: ReviewHistoryItem[]
  isLoading: boolean
}

export function ReviewHistoryList({ items, isLoading }: ReviewHistoryListProps) {
  return (
    <Panel className="p-6">
      <SectionHeader title="最近复习记录" description="帮助你快速回顾最近练过什么、当前停在哪一步。" />

      <div className="mt-5 space-y-3">
        {isLoading ? (
          <div className="rounded-xl border border-dashed border-border bg-secondary/35 px-4 py-6 text-sm text-muted-foreground">
            正在加载最近记录...
          </div>
        ) : items.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border bg-secondary/35 px-4 py-6 text-sm text-muted-foreground">
            还没有复习记录，先完成一轮知识漫游，记录会自动出现在这里。
          </div>
        ) : (
          items.map((item) => (
            <article
              key={item.session_id}
              className="rounded-xl border border-border bg-card px-4 py-4"
            >
              <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="text-sm font-medium text-foreground">{item.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{item.source_title || '未标注系列'}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <StatusPill>{item.mode === 'detailed_qa' ? '细致提问' : '轻提示复述'}</StatusPill>
                  <StatusPill>
                    {item.status === 'completed' ? '已完成' : item.status === 'in_progress' ? '进行中' : '已创建'}
                  </StatusPill>
                </div>
              </div>
              <p className="mt-3 text-sm leading-6 text-muted-foreground">{item.summary}</p>
              <p className="mt-2 text-xs text-muted-foreground">
                {item.reviewed_at ? `最近时间：${new Date(item.reviewed_at).toLocaleString()}` : '最近时间：暂无'}
              </p>
            </article>
          ))
        )}
      </div>
    </Panel>
  )
}
