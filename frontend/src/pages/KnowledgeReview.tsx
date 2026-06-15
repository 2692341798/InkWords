import { useEffect, useState } from 'react'
import { ReviewEntryCards } from '@/components/review/ReviewEntryCards'
import { ReviewHistoryList } from '@/components/review/ReviewHistoryList'
import { ReviewNotePicker } from '@/components/review/ReviewNotePicker'
import { ReviewSessionCard } from '@/components/review/ReviewSessionCard'
import { StepStrip, type StepStripItem } from '@/components/shared/StepStrip'
import { Button } from '@/components/ui/button'
import { useKnowledgeReview } from '@/hooks/useKnowledgeReview'
import { useReviewStore } from '@/store/reviewStore'
import { getKnowledgeReviewViewState } from './knowledgeReviewViewState'

// Why: 复习页要把“选择入口 / 手动挑题 / 开始会话”切成单一主步骤，
// 避免用户同时面对多个主操作区，降低启动阻力。
export function KnowledgeReview() {
  const reviewStore = useReviewStore()
  const { initialize, startSession, startAnswering, respond, requestHint, finish, clearSession } = useKnowledgeReview()
  const [isPickerOpen, setIsPickerOpen] = useState(false)
  const effectiveIsPickerOpen = isPickerOpen && !reviewStore.currentSession
  const hasHiddenSession = Boolean(reviewStore.currentSession) && !reviewStore.shouldResumeSessionOnOpen
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
  const currentEntrySummary = reviewStore.currentSession
    ? reviewStore.currentSession.title
    : effectiveIsPickerOpen
      ? '手动选择文章'
      : '随机抽题 / 手动挑题'
  const nextActionText =
    viewState.currentStep === 'entry' && hasHiddenSession
      ? '下一步：你有一个未完成的复习会话，可以继续当前进度，或放弃后重新选择新的入口。'
      : viewState.currentStep === 'entry'
      ? '下一步：先决定是随机抽一篇开始，还是手动挑一篇进入复习。'
      : viewState.currentStep === 'picker'
        ? '下一步：在候选列表中锁定一篇文章，系统会用推荐模式直接开启会话。'
        : '下一步：围绕当前主题持续作答或请求提示，直到形成最终反馈。'
  const currentModeLabel = reviewStore.currentSession?.mode === 'detailed_qa'
    ? '细致提问'
    : reviewStore.currentSession?.mode === 'light_recall'
      ? '轻提示复述'
      : reviewStore.selectedMode === 'detailed_qa'
        ? '细致提问'
        : '轻提示复述'

  const closeSession = () => {
    clearSession()
    setIsPickerOpen(false)
  }

  return (
    <div className="flex-1 overflow-y-auto bg-background custom-scrollbar">
      <div className="mx-auto flex max-w-5xl flex-col gap-6 px-6 py-8">
        <section className="overflow-hidden rounded-2xl border border-border bg-card shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
          <div className="grid gap-0 lg:grid-cols-[minmax(0,1.1fr)_300px]">
            <div className="px-8 py-8">
              <span className="inline-flex items-center rounded-md bg-secondary px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                知识漫游复习
              </span>
              <div className="mt-5 space-y-3">
                <h1 className="text-2xl font-semibold tracking-tight text-foreground">像整理笔记一样，把重点重新讲出来</h1>
                <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
                  这里优先保留阅读、复述和回看原文时真正需要的部分。先聚焦当前问题，需要时再从右侧或会话卡片里展开原文细节。
                </p>
              </div>
              <div className="mt-8 flex flex-wrap gap-3">
                <span className="rounded-md border border-border bg-secondary px-3 py-1 text-xs text-muted-foreground">极简阅读优先</span>
                <span className="rounded-md border border-border bg-secondary px-3 py-1 text-xs text-muted-foreground">原文抽屉式阅读</span>
              </div>
            </div>
            <div className="flex flex-col justify-between border-l border-border bg-secondary/30 px-6 py-8">
              <div>
                <p className="text-[11px] font-medium tracking-[0.2em] text-muted-foreground uppercase">当前摘要</p>
                <div className="mt-5 space-y-3">
                  <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                    <p className="text-xs text-muted-foreground">当前步骤</p>
                    <p className="mt-1 text-sm font-medium text-foreground">{currentStepMeta.title}</p>
                  </div>
                  <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                    <p className="text-xs text-muted-foreground">当前主题</p>
                    <p className="mt-1 text-sm font-medium text-foreground">{currentEntrySummary}</p>
                  </div>
                  <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                    <p className="text-xs text-muted-foreground">当前模式</p>
                    <p className="mt-1 text-sm font-medium text-foreground">{currentModeLabel}</p>
                  </div>
                </div>
              </div>
              <div className="mt-6 rounded-xl border border-dashed border-border bg-card px-4 py-4 text-sm leading-relaxed text-muted-foreground">
                {nextActionText}
              </div>
            </div>
          </div>
        </section>

        <section className="rounded-2xl border border-border bg-card p-6 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
          <div className="mb-6 flex items-center justify-between gap-4">
            <div>
              <p className="text-[11px] font-medium tracking-[0.2em] text-muted-foreground uppercase">复习流程</p>
              <h2 className="mt-2 text-lg font-medium text-foreground">先决定入口，再进入当前这一次复习</h2>
            </div>
            <span className="rounded-md border border-border bg-secondary px-3 py-1 text-xs font-medium text-muted-foreground">
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

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="space-y-6">
            {viewState.shouldShowEntryStep && (
              <>
                {hasHiddenSession && (
                  <section className="rounded-[28px] border border-indigo-200 bg-indigo-50 px-5 py-4">
                    <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-medium text-indigo-900">你有一个未完成的复习会话</p>
                        <p className="mt-1 text-sm leading-6 text-indigo-800">
                          当前主题：{reviewStore.currentSession?.title}，可继续上次进度，或放弃后重新选择新的入口。
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-3">
                        <Button
                          onClick={() => {
                            reviewStore.setShouldResumeSessionOnOpen(true)
                            setIsPickerOpen(false)
                          }}
                        >
                          继续当前会话
                        </Button>
                        <Button
                          variant="outline"
                          onClick={() => {
                            clearSession()
                            setIsPickerOpen(false)
                          }}
                        >
                          放弃当前会话
                        </Button>
                      </div>
                    </div>
                  </section>
                )}

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
              </>
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
                onStartAnswering={startAnswering}
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

          <aside className="h-fit rounded-2xl border border-border bg-card p-6 shadow-[0_2px_8px_rgba(0,0,0,0.02)] lg:sticky lg:top-6">
            <div className="rounded-xl border border-border bg-secondary/30 px-5 py-5">
              <p className="text-[11px] font-medium tracking-[0.2em] text-muted-foreground uppercase">任务摘要</p>
              <h2 className="mt-3 text-lg font-medium text-foreground">只保留必要的决策信息</h2>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">当前步骤、主题、模式和下一步动作持续固定显示，避免页面看起来像堆叠的后台面板。</p>
            </div>
            <div className="mt-5 space-y-3">
              <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                <span className="text-xs text-muted-foreground">当前步骤</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentStepMeta.title}</p>
              </div>
              <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                <span className="text-xs text-muted-foreground">当前主题</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentEntrySummary}</p>
              </div>
              <div className="rounded-xl border border-border bg-card px-4 py-3 shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
                <span className="text-xs text-muted-foreground">当前模式</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentModeLabel}</p>
              </div>
            </div>
            <div className="mt-5 rounded-xl border border-dashed border-border bg-card px-4 py-4 text-sm leading-relaxed text-muted-foreground">
              {nextActionText}
            </div>
          </aside>
        </div>
      </div>
    </div>
  )
}
