import { useEffect, useState } from 'react'
import { ReviewEntryCards } from '@/components/review/ReviewEntryCards'
import { ReviewHistoryList } from '@/components/review/ReviewHistoryList'
import { ReviewNotePicker } from '@/components/review/ReviewNotePicker'
import { ReviewSessionCard } from '@/components/review/ReviewSessionCard'
import { StepStrip, type StepStripItem } from '@/components/shared/StepStrip'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
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
    <PageShell>
        <PageHeader
          title="像整理笔记一样，把重点重新讲出来"
          description="先聚焦当前问题，需要时再展开原文细节，让复习过程保持安静和可持续。"
          meta={<StatusPill tone="success">知识漫游复习</StatusPill>}
          actions={<StatusPill>当前步骤：{currentStepMeta.title}</StatusPill>}
        />

        <Panel className="p-6">
          <StepStrip
            title="当前流程"
            description={currentStepMeta.description}
            steps={steps}
            currentStepIndex={viewState.currentStepIndex}
            variant="progress"
          />
        </Panel>

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="space-y-6">
            {viewState.shouldShowEntryStep && (
              <>
                {hasHiddenSession && (
                  <section className="rounded-xl border border-[color-mix(in_srgb,var(--brand)_22%,var(--border))] bg-[var(--brand-soft)] px-5 py-4">
                    <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-medium text-foreground">你有一个未完成的复习会话</p>
                        <p className="mt-1 text-sm leading-6 text-muted-foreground">
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
                onRespond={respond}
                onStartAnswering={startAnswering}
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

          <aside className="summary-rail">
            <SectionHeader eyebrow="任务摘要" title="必要决策信息" description="当前步骤、主题、模式和下一步动作固定显示。" />
            <div className="mt-5 space-y-3">
              <div className="summary-row">
                <span className="text-xs text-muted-foreground">当前步骤</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentStepMeta.title}</p>
              </div>
              <div className="summary-row">
                <span className="text-xs text-muted-foreground">当前主题</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentEntrySummary}</p>
              </div>
              <div className="summary-row">
                <span className="text-xs text-muted-foreground">当前模式</span>
                <p className="mt-1 text-sm font-medium text-foreground">{currentModeLabel}</p>
              </div>
            </div>
            <div className="mt-5 rounded-xl border border-dashed border-border bg-secondary/35 px-4 py-4 text-sm leading-relaxed text-muted-foreground">
              {nextActionText}
            </div>
          </aside>
        </div>
    </PageShell>
  )
}
