import { useEffect, useState } from 'react'
import { ReviewEntryCards } from '@/components/review/ReviewEntryCards'
import { ReviewHistoryList } from '@/components/review/ReviewHistoryList'
import { ReviewNotePicker } from '@/components/review/ReviewNotePicker'
import { ReviewSessionCard } from '@/components/review/ReviewSessionCard'
import { StepStrip, type StepStripItem } from '@/components/shared/StepStrip'
import { useKnowledgeReview } from '@/hooks/useKnowledgeReview'
import { useReviewStore } from '@/store/reviewStore'
import { getKnowledgeReviewViewState } from './knowledgeReviewViewState'

// Why: 复习页要把“选择入口 / 手动挑题 / 开始会话”切成单一主步骤，
// 避免用户同时面对多个主操作区，降低启动阻力。
export function KnowledgeReview() {
  const reviewStore = useReviewStore()
  const { initialize, startSession, respond, requestHint, finish, clearSession } = useKnowledgeReview()
  const [isPickerOpen, setIsPickerOpen] = useState(false)
  const effectiveIsPickerOpen = isPickerOpen && !reviewStore.currentSession
  const viewState = getKnowledgeReviewViewState({
    hasSession: Boolean(reviewStore.currentSession),
    isPickerOpen: effectiveIsPickerOpen,
    shouldEnterSession: reviewStore.shouldResumeSessionOnOpen,
  })

  useEffect(() => {
    void initialize()
  }, [initialize])

  const steps: StepStripItem[] = [
    { key: 'entry', title: '选择入口', description: '先决定今天从哪种复习方式开始。' },
    { key: 'picker', title: '选择文章', description: '只在手动挑题时展示候选列表。' },
    { key: 'session', title: '开始会话', description: '进入当前主题，持续复述、提示与反馈。' },
  ]

  const currentStepMeta = steps[viewState.currentStepIndex]
  const currentEntrySummary = viewState.currentStep === 'session' && reviewStore.currentSession
    ? reviewStore.currentSession.title
    : effectiveIsPickerOpen
      ? '手动选择文章'
      : '随机抽题 / 手动挑题'
  const nextActionText =
    viewState.currentStep === 'entry'
      ? '下一步：先决定是随机抽一篇开始，还是手动挑一篇进入复习。'
      : viewState.currentStep === 'picker'
        ? '下一步：在候选列表中锁定一篇文章，系统会用推荐模式直接开启会话。'
        : '下一步：围绕当前主题持续作答或请求提示，直到形成最终反馈。'

  const closeSession = () => {
    clearSession()
    setIsPickerOpen(false)
  }

  return (
    <div className="flex-1 overflow-y-auto bg-zinc-50 custom-scrollbar">
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-12">
        <section className="rounded-3xl border border-zinc-200 bg-white px-8 py-10 shadow-sm">
          <div className="space-y-4">
            <span className="inline-flex items-center rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700">
              知识漫游复习
            </span>
            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-tight text-zinc-900">把知识库里的重点内容重新讲出来</h1>
              <p className="max-w-3xl text-sm leading-6 text-zinc-600">
                这里会承接随机抽题和手动选文两种入口，并把会话提示、追问和最近记录收敛在同一个复习工作台里。
              </p>
            </div>
          </div>
        </section>

        <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
          <div className="mb-5 flex justify-end">
            <span className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600">
              当前步骤：{currentStepMeta.title}
            </span>
          </div>
          <StepStrip
            title="当前流程"
            description={currentStepMeta.description}
            steps={steps}
            currentStepIndex={viewState.currentStepIndex}
            variant="progress"
          />
        </section>

        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.4fr)_340px]">
          <div className="space-y-6">
            {viewState.shouldShowEntryStep && (
              <ReviewEntryCards
                recommendationCard={reviewStore.recommendationCard}
                isLoadingRecommendation={reviewStore.isLoadingRecommendation}
                onRefreshRecommendation={async () => {
                  // Why: 重新抽题代表用户要放弃上一轮主题，先清掉潜在的旧会话，
                  // 避免“候选已刷新，但页面还被旧 session 推进到第 3 步”。
                  clearSession()
                  await reviewStore.refreshRecommendation()
                }}
                onStartRecommendation={async () => {
                  if (!reviewStore.recommendationCard) {
                    await reviewStore.loadRecommendation()
                  }
                  const card = useReviewStore.getState().recommendationCard
                  if (card) {
                    await startSession(card, 'manual_random')
                  }
                }}
                onStartQuestionRecommendation={async () => {
                  if (!reviewStore.recommendationCard) {
                    await reviewStore.loadRecommendation()
                  }
                  const card = useReviewStore.getState().recommendationCard
                  if (card) {
                    // Why: “提问开始”是一个显式模式入口，点击后要先把页面状态切到细致提问，
                    // 再复用现有推荐卡片开局流程，确保后续会话和右侧摘要都保持一致。
                    reviewStore.setSelectedMode('detailed_qa')
                    await startSession(card, 'manual_random', 'detailed_qa')
                  }
                }}
                onOpenPicker={async () => {
                  clearSession()
                  setIsPickerOpen(true)
                  await reviewStore.loadNotes()
                }}
              />
            )}

            {viewState.shouldShowPickerStep && (
              <ReviewNotePicker
                notes={reviewStore.noteOptions}
                isLoading={reviewStore.isLoadingNotes}
                onSearch={(query) => reviewStore.loadNotes(query)}
                onModeSync={(mode) => reviewStore.setSelectedMode(mode)}
                onSelect={async (card) => {
                  await startSession(card, 'manual_select')
                }}
                onBack={() => setIsPickerOpen(false)}
              />
            )}

            {viewState.shouldShowSessionStep && (
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
                onClose={closeSession}
              />
            )}

            {viewState.shouldShowHistory && (
              <ReviewHistoryList
                items={reviewStore.historyItems}
                isLoading={reviewStore.isLoadingHistory}
              />
            )}
          </div>

          <aside className="h-fit rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
            <div>
              <h2 className="text-lg font-semibold text-zinc-900">任务摘要</h2>
              <p className="mt-1 text-sm leading-6 text-zinc-600">
                右侧持续告诉用户当前所处的复习阶段，以及接下来唯一应该做的动作。
              </p>
            </div>
            <div className="mt-5 space-y-3">
              <div className="flex items-start justify-between gap-4 rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3">
                <span className="text-sm text-zinc-500">当前步骤</span>
                <span className="text-right text-sm font-medium text-zinc-900">{currentStepMeta.title}</span>
              </div>
              <div className="flex items-start justify-between gap-4 rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3">
                <span className="text-sm text-zinc-500">当前主题</span>
                <span className="text-right text-sm font-medium text-zinc-900">{currentEntrySummary}</span>
              </div>
              <div className="flex items-start justify-between gap-4 rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3">
                <span className="text-sm text-zinc-500">当前模式</span>
                <span className="text-right text-sm font-medium text-zinc-900">
                  {reviewStore.selectedMode === 'detailed_qa' ? '细致提问' : '轻提示复述'}
                </span>
              </div>
            </div>
            <div className="mt-5 rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-4 text-sm leading-6 text-zinc-600">
              {nextActionText}
            </div>
          </aside>
        </div>
      </div>
    </div>
  )
}
