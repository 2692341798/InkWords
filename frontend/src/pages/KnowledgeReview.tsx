import { useEffect } from 'react'
import { ReviewEntryCards } from '@/components/review/ReviewEntryCards'
import { ReviewHistoryList } from '@/components/review/ReviewHistoryList'
import { ReviewNotePicker } from '@/components/review/ReviewNotePicker'
import { ReviewSessionCard } from '@/components/review/ReviewSessionCard'
import { useKnowledgeReview } from '@/hooks/useKnowledgeReview'
import { useReviewStore } from '@/store/reviewStore'

// Why: 知识漫游复习需要把推荐入口、手动选文、会话输入和最近记录放在同一视图里，用户才能在一次停留中完成“开始 -> 输出 -> 复盘”的闭环。
export function KnowledgeReview() {
  const reviewStore = useReviewStore()
  const { initialize, startSession, respond, requestHint, finish } = useKnowledgeReview()

  useEffect(() => {
    void initialize()
  }, [initialize])

  return (
    <div className="flex-1 overflow-y-auto bg-zinc-50 custom-scrollbar">
      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-12">
        <section className="rounded-3xl border border-zinc-200 bg-white px-8 py-10 shadow-sm">
          <div className="space-y-4">
            <span className="inline-flex items-center rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700">
              知识漫游复习
            </span>
            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-tight text-zinc-900">把知识库里的重点内容重新讲出来</h1>
              <p className="max-w-3xl text-sm leading-6 text-zinc-600">
                这里会承接今日推荐、随机抽题和手动选文三种入口，并把会话提示、追问和最近记录收敛在同一个复习工作台里。
              </p>
            </div>
          </div>
        </section>

        <ReviewEntryCards
          todayCard={reviewStore.todayCard}
          randomCard={reviewStore.randomCard}
          isLoadingToday={reviewStore.isLoadingToday}
          isLoadingRandom={reviewStore.isLoadingRandom}
          onRefreshToday={() => reviewStore.loadToday()}
          onRefreshRandom={() => reviewStore.loadRandom()}
          onStartToday={async () => {
            if (!reviewStore.todayCard) {
              await reviewStore.loadToday()
            }
            const card = useReviewStore.getState().todayCard
            if (card) {
              await startSession(card, 'today')
            }
          }}
          onStartRandom={async () => {
            if (!reviewStore.randomCard) {
              await reviewStore.loadRandom()
            }
            const card = useReviewStore.getState().randomCard
            if (card) {
              await startSession(card, 'manual_random')
            }
          }}
          onOpenPicker={() => reviewStore.loadNotes()}
        />

        <ReviewNotePicker
          notes={reviewStore.noteOptions}
          isLoading={reviewStore.isLoadingNotes}
          onSearch={(query) => reviewStore.loadNotes(query)}
          onModeSync={(mode) => reviewStore.setSelectedMode(mode)}
          onSelect={async (card) => {
            await startSession(card, 'manual_select')
          }}
        />

        <ReviewSessionCard
          session={reviewStore.currentSession}
          selectedMode={reviewStore.selectedMode}
          latestStageFeedback={reviewStore.latestStageFeedback}
          latestHint={reviewStore.latestHint}
          finalFeedback={reviewStore.finalFeedback}
          onModeChange={reviewStore.setSelectedMode}
          onRespond={respond}
          onRequestHint={requestHint}
          onFinish={finish}
        />

        <ReviewHistoryList
          items={reviewStore.historyItems}
          isLoading={reviewStore.isLoadingHistory}
        />
      </div>
    </div>
  )
}
