import type { ReviewHistoryItem } from '@/services/review'

interface ReviewHistoryListProps {
  items: ReviewHistoryItem[]
  isLoading: boolean
}

export function ReviewHistoryList({ items, isLoading }: ReviewHistoryListProps) {
  return (
    <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900">最近复习记录</h2>
        <p className="mt-1 text-sm leading-6 text-zinc-600">帮助你快速回顾最近练过什么、当前停在哪一步。</p>
      </div>

      <div className="mt-5 space-y-3">
        {isLoading ? (
          <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-6 text-sm text-zinc-500">
            正在加载最近记录...
          </div>
        ) : items.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-6 text-sm text-zinc-500">
            还没有复习记录，先完成一轮知识漫游，记录会自动出现在这里。
          </div>
        ) : (
          items.map((item) => (
            <article
              key={item.session_id}
              className="rounded-2xl border border-zinc-200 px-4 py-4"
            >
              <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="text-sm font-medium text-zinc-900">{item.title}</p>
                  <p className="mt-1 text-sm text-zinc-600">{item.source_title || '未标注系列'}</p>
                </div>
                <div className="flex flex-wrap gap-2 text-xs text-zinc-500">
                  <span className="rounded-full bg-zinc-100 px-3 py-1">{item.mode === 'detailed_qa' ? '细致提问' : '轻提示复述'}</span>
                  <span className="rounded-full bg-zinc-100 px-3 py-1">
                    {item.status === 'completed' ? '已完成' : item.status === 'in_progress' ? '进行中' : '已创建'}
                  </span>
                </div>
              </div>
              <p className="mt-3 text-sm leading-6 text-zinc-700">{item.summary}</p>
              <p className="mt-2 text-xs text-zinc-500">
                {item.reviewed_at ? `最近时间：${new Date(item.reviewed_at).toLocaleString()}` : '最近时间：暂无'}
              </p>
            </article>
          ))
        )}
      </div>
    </section>
  )
}
