import type { ComponentType } from 'react'
import { BookOpen, Clock3, Shuffle, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ReviewCardResponse } from '@/services/review'

interface ReviewEntryCardsProps {
  todayCard: ReviewCardResponse | null
  randomCard: ReviewCardResponse | null
  isLoadingToday: boolean
  isLoadingRandom: boolean
  onRefreshToday: () => Promise<void> | void
  onRefreshRandom: () => Promise<void> | void
  onStartToday: () => Promise<void> | void
  onStartRandom: () => Promise<void> | void
  onOpenPicker: () => Promise<void> | void
}

const fallbackCard = {
  review_reason: '先抽一篇可复习的知识卡，再开始这次主动回忆。',
  estimated_minutes: 5,
}

function ReviewCard({
  title,
  description,
  icon,
  detail,
  loading,
  actionLabel,
  refreshLabel,
  onAction,
  onRefresh,
}: {
  title: string
  description: string
  icon: ComponentType<{ className?: string }>
  detail: ReviewCardResponse | null
  loading: boolean
  actionLabel: string
  refreshLabel: string
  onAction: () => Promise<void> | void
  onRefresh: () => Promise<void> | void
}) {
  const Icon = icon

  return (
    <article className="rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm">
      <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 text-zinc-700">
        <Icon className="h-5 w-5" />
      </div>
      <div className="space-y-2">
        <h2 className="text-base font-semibold text-zinc-900">{title}</h2>
        <p className="text-sm leading-6 text-zinc-600">{description}</p>
      </div>
      <div className="mt-5 rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 p-4">
        <p className="text-sm font-medium text-zinc-900">{detail?.title ?? '还没有抽到本次题卡'}</p>
        <p className="mt-2 text-sm leading-6 text-zinc-600">
          {detail?.review_reason ?? fallbackCard.review_reason}
        </p>
        <div className="mt-3 flex items-center gap-2 text-xs text-zinc-500">
          <Clock3 className="h-3.5 w-3.5" />
          <span>预计 {detail?.estimated_minutes ?? fallbackCard.estimated_minutes} 分钟</span>
        </div>
      </div>
      <div className="mt-4 flex gap-3">
        <Button className="flex-1" onClick={onAction} disabled={loading}>
          {actionLabel}
        </Button>
        <Button variant="outline" className="flex-1" onClick={onRefresh} disabled={loading}>
          {refreshLabel}
        </Button>
      </div>
    </article>
  )
}

export function ReviewEntryCards(props: ReviewEntryCardsProps) {
  return (
    <section className="grid gap-4 md:grid-cols-3">
      <ReviewCard
        title="开始今日复习"
        description="优先推荐今天最值得回顾的一篇内容。"
        icon={Sparkles}
        detail={props.todayCard}
        loading={props.isLoadingToday}
        actionLabel="开始今日复习"
        refreshLabel="刷新推荐"
        onAction={props.onStartToday}
        onRefresh={props.onRefreshToday}
      />
      <ReviewCard
        title="随机抽一篇"
        description="适合快速进入状态，打散固定复习路径。"
        icon={Shuffle}
        detail={props.randomCard}
        loading={props.isLoadingRandom}
        actionLabel="用这篇开始"
        refreshLabel="再抽一篇"
        onAction={props.onStartRandom}
        onRefresh={props.onRefreshRandom}
      />
      <article className="rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-zinc-100 text-zinc-700">
          <BookOpen className="h-5 w-5" />
        </div>
        <div className="space-y-2">
          <h2 className="text-base font-semibold text-zinc-900">选择文章复习</h2>
          <p className="text-sm leading-6 text-zinc-600">按关键词筛选并手动挑选本次要训练的主题。</p>
        </div>
        <div className="mt-5 rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 p-4 text-sm leading-6 text-zinc-600">
          当你想定向练某个主题，而不是跟着系统漫游时，从候选列表里直接点选即可。
        </div>
        <div className="mt-4">
          <Button className="w-full" variant="outline" onClick={props.onOpenPicker}>
            打开候选列表
          </Button>
        </div>
      </article>
    </section>
  )
}
