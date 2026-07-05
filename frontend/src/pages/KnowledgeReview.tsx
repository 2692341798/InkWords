import { useEffect, useState } from 'react'
import { ReviewEntryCards } from '@/components/review/ReviewEntryCards'
import { ReviewHistoryList } from '@/components/review/ReviewHistoryList'
import { ReviewNotePicker } from '@/components/review/ReviewNotePicker'
import { ReviewSessionCard } from '@/components/review/ReviewSessionCard'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, StatusPill } from '@/components/ui/workspace'
import { useKnowledgeReview } from '@/hooks/useKnowledgeReview'
import { useReviewStore } from '@/store/reviewStore'

export function KnowledgeReview() {
  const reviewStore = useReviewStore()
  const { initialize, startSession, completeReading, respond, requestHint, finish, clearSession, isPendingAction } = useKnowledgeReview()
  const [isPickerOpen, setIsPickerOpen] = useState(false)
  const showSession = Boolean(reviewStore.currentSession && reviewStore.shouldResumeSessionOnOpen)

  useEffect(() => { void initialize() }, [initialize])

  if (showSession) {
    return (
      <PageShell className="min-h-screen">
        <ReviewSessionCard
          session={reviewStore.currentSession}
          selectedMode={reviewStore.selectedMode}
          latestStageFeedback={reviewStore.latestStageFeedback}
          latestHint={reviewStore.latestHint}
          finalFeedback={reviewStore.finalFeedback}
          pending={isPendingAction}
          onModeChange={reviewStore.setSelectedMode}
          onCompleteReading={completeReading}
          onRespond={respond}
          onRequestHint={requestHint}
          onFinish={finish}
          onClose={() => { clearSession(); setIsPickerOpen(false) }}
        />
      </PageShell>
    )
  }

  return (
    <PageShell>
      <PageHeader
        title="选一篇，读完，再用自己的话讲出来"
        description="先完整浏览原文。进入复述后原文会收起，帮助你真正检验理解，而不是照着摘抄。"
        meta={<StatusPill tone="success">知识漫游复习</StatusPill>}
      />

      {reviewStore.currentSession ? (
        <section className="mb-6 rounded-xl border border-[color-mix(in_srgb,var(--brand)_22%,var(--border))] bg-[var(--brand-soft)] px-5 py-4">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div><p className="text-sm font-medium text-foreground">继续上次的复习</p><p className="mt-1 text-sm text-muted-foreground">{reviewStore.currentSession.title}</p></div>
            <div className="flex gap-3"><Button onClick={() => reviewStore.setShouldResumeSessionOnOpen(true)}>继续</Button><Button variant="outline" onClick={clearSession}>放弃</Button></div>
          </div>
        </section>
      ) : null}

      {!isPickerOpen ? (
        <>
          <ReviewEntryCards
            recommendationCard={reviewStore.recommendationCard}
            isLoadingRecommendation={reviewStore.isLoadingRecommendation}
            onRefreshRecommendation={async () => { clearSession(); await reviewStore.refreshRecommendation() }}
            onStartRecommendation={async () => {
              if (!reviewStore.recommendationCard) await reviewStore.loadRecommendation()
              const card = useReviewStore.getState().recommendationCard
              if (card) await startSession(card, 'manual_random')
            }}
            onOpenPicker={async () => { clearSession(); setIsPickerOpen(true); await reviewStore.loadNotes() }}
          />
          <div className="mt-8"><ReviewHistoryList items={reviewStore.historyItems} isLoading={reviewStore.isLoadingHistory} /></div>
        </>
      ) : (
        <ReviewNotePicker
          notes={reviewStore.noteOptions}
          isLoading={reviewStore.isLoadingNotes}
          onSearch={(query) => reviewStore.loadNotes(query)}
          onModeSync={reviewStore.setSelectedMode}
          onSelect={async (card) => { await startSession(card, 'manual_select') }}
          onBack={() => setIsPickerOpen(false)}
        />
      )}
    </PageShell>
  )
}
