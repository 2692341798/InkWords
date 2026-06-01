import type { ComponentType } from 'react'
import { BookOpen, Clock3, Shuffle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ReviewCardResponse } from '@/services/review'

interface ReviewEntryCardsProps {
  recommendationCard: ReviewCardResponse | null
  isLoadingRecommendation: boolean
  onRefreshRecommendation: () => Promise<void> | void
  onStartRecommendation: () => Promise<void> | void
  onStartQuestionRecommendation: () => Promise<void> | void
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
  questionActionLabel,
  refreshLabel,
  onAction,
  onQuestionAction,
  onRefresh,
}: {
  title: string
  description: string
  icon: ComponentType<{ className?: string }>
  detail: ReviewCardResponse | null
  loading: boolean
  actionLabel: string
  questionActionLabel: string
  refreshLabel: string
  onAction: () => Promise<void> | void
  onQuestionAction: () => Promise<void> | void
  onRefresh: () => Promise<void> | void
}) {
  const Icon = icon
  const supportsDetailedQuestionMode = detail?.available_modes.includes('detailed_qa') ?? false

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
      <div className="mt-4 flex flex-col gap-3 sm:flex-row">
        <Button className="sm:flex-1" onClick={onAction} disabled={loading}>
          {actionLabel}
        </Button>
        <Button
          variant="outline"
          className="sm:flex-1"
          onClick={onQuestionAction}
          disabled={loading || !supportsDetailedQuestionMode}
        >
          {questionActionLabel}
        </Button>
        <Button variant="outline" className="sm:flex-1" onClick={onRefresh} disabled={loading}>
          {refreshLabel}
        </Button>
      </div>
    </article>
  )
}

export function ReviewEntryCards(props: ReviewEntryCardsProps) {
  return (
    <section className="grid gap-4 md:grid-cols-2">
      <ReviewCard
        title="随机抽一篇"
        description="适合快速进入状态，先从一篇随机文章开始复习。"
        icon={Shuffle}
        detail={props.recommendationCard}
        loading={props.isLoadingRecommendation}
        actionLabel="用这篇开始"
        questionActionLabel="提问开始"
        refreshLabel="再抽一篇"
        onAction={props.onStartRecommendation}
        onQuestionAction={props.onStartQuestionRecommendation}
        onRefresh={props.onRefreshRecommendation}
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
