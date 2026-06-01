interface KnowledgeReviewViewStateInput {
  hasSession: boolean
  isPickerOpen: boolean
  shouldEnterSession: boolean
}

/**
 * Why: 复习页的入口卡、候选列表和会话面板属于不同步骤，
 * 通过页面级状态集中裁剪显示范围，避免用户同时看到多个主任务区块。
 */
export function getKnowledgeReviewViewState({
  hasSession,
  isPickerOpen,
  shouldEnterSession,
}: KnowledgeReviewViewStateInput) {
  const currentStep = hasSession && shouldEnterSession ? 'session' : isPickerOpen ? 'picker' : 'entry'

  return {
    currentStep,
    currentStepIndex: currentStep === 'entry' ? 0 : currentStep === 'picker' ? 1 : 2,
    shouldShowEntryStep: currentStep === 'entry',
    shouldShowPickerStep: currentStep === 'picker',
    shouldShowSessionStep: currentStep === 'session',
    shouldShowHistory: currentStep === 'entry',
  }
}
