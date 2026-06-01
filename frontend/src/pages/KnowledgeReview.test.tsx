import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { KnowledgeReview } from './KnowledgeReview'

const {
  buttonClickHandlers,
  capturedEntryCardsProps,
  clearSessionMock,
  finishMock,
  initializeMock,
  requestHintMock,
  respondMock,
  setShouldResumeSessionOnOpenMock,
  setSelectedModeMock,
  startSessionMock,
  storeState,
} = vi.hoisted(() => {
  const recommendationCard = {
    note_path: 'wiki/concepts/random.md',
    title: '随机文章',
    source_title: '知识库',
    review_reason: '从随机文章开始复习',
    estimated_minutes: 6,
    available_modes: ['light_recall', 'detailed_qa'] as const,
  }

  const state = {
    recommendationCard,
    isLoadingRecommendation: false,
    currentSession: null,
    shouldResumeSessionOnOpen: false,
    latestStageFeedback: null,
    latestHint: null,
    finalFeedback: null,
    historyItems: [],
    isLoadingHistory: false,
    noteOptions: [],
    isLoadingNotes: false,
    selectedMode: 'light_recall' as const,
    loadRecommendation: vi.fn().mockResolvedValue(undefined),
    refreshRecommendation: vi.fn().mockResolvedValue(undefined),
    loadNotes: vi.fn().mockResolvedValue(undefined),
    loadHistory: vi.fn().mockResolvedValue(undefined),
    setShouldResumeSessionOnOpen: vi.fn(),
    setSelectedMode: vi.fn(),
  }

  return {
    buttonClickHandlers: new Map<string, (() => Promise<void> | void) | undefined>(),
    capturedEntryCardsProps: {
      current: null as null | Record<string, unknown>,
    },
    clearSessionMock: vi.fn(),
    finishMock: vi.fn(),
    initializeMock: vi.fn().mockResolvedValue(undefined),
    requestHintMock: vi.fn(),
    respondMock: vi.fn(),
    setShouldResumeSessionOnOpenMock: state.setShouldResumeSessionOnOpen,
    setSelectedModeMock: state.setSelectedMode,
    startSessionMock: vi.fn(),
    storeState: state,
  }
})

vi.mock('@/components/ui/button', () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: string
    onClick?: () => Promise<void> | void
  }) => {
    buttonClickHandlers.set(children, onClick)
    return <button>{children}</button>
  },
}))

vi.mock('@/components/review/ReviewEntryCards', () => ({
  ReviewEntryCards: (props: Record<string, unknown>) => {
    capturedEntryCardsProps.current = props
    return <div>ReviewEntryCardsStub</div>
  },
}))

vi.mock('@/components/review/ReviewHistoryList', () => ({
  ReviewHistoryList: () => <div>ReviewHistoryListStub</div>,
}))

vi.mock('@/components/review/ReviewNotePicker', () => ({
  ReviewNotePicker: () => <div>ReviewNotePickerStub</div>,
}))

vi.mock('@/components/review/ReviewSessionCard', () => ({
  ReviewSessionCard: () => <div>ReviewSessionCardStub</div>,
}))

vi.mock('@/components/shared/StepStrip', () => ({
  StepStrip: () => <div>StepStripStub</div>,
}))

vi.mock('@/hooks/useKnowledgeReview', () => ({
  useKnowledgeReview: () => ({
    initialize: initializeMock,
    startSession: startSessionMock,
    respond: respondMock,
    requestHint: requestHintMock,
    finish: finishMock,
    clearSession: clearSessionMock,
  }),
}))

vi.mock('@/store/reviewStore', () => {
  const useReviewStore = () => storeState
  useReviewStore.getState = () => storeState

  return {
    useReviewStore,
  }
})

describe('KnowledgeReview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    buttonClickHandlers.clear()
    capturedEntryCardsProps.current = null
    storeState.currentSession = null
    storeState.shouldResumeSessionOnOpen = false
    storeState.selectedMode = 'light_recall'
  })

  it('点击提问开始时先切到细致提问模式，再用推荐卡片开启会话', async () => {
    renderToStaticMarkup(<KnowledgeReview />)

    const props = capturedEntryCardsProps.current as null | {
      onStartQuestionRecommendation?: () => Promise<void>
    }

    expect(props?.onStartQuestionRecommendation).toBeTypeOf('function')

    await props?.onStartQuestionRecommendation?.()

    expect(setSelectedModeMock).toHaveBeenCalledWith('detailed_qa')
    expect(startSessionMock).toHaveBeenCalledWith(
      expect.objectContaining({ note_path: 'wiki/concepts/random.md' }),
      'manual_random',
      'detailed_qa',
    )
    expect(setSelectedModeMock.mock.invocationCallOrder[0]).toBeLessThan(
      startSessionMock.mock.invocationCallOrder[0],
    )
  })

  it('在入口态保留已恢复会话的继续入口，而不是把会话静默藏起来', async () => {
    storeState.currentSession = {
      session_id: 'session-1',
      status: 'in_progress',
      mode: 'detailed_qa',
      title: '恢复中的会话',
      opening_prompt: '继续作答',
      initial_hints: [],
      turn_index: 2,
    }
    storeState.shouldResumeSessionOnOpen = false

    const html = renderToStaticMarkup(<KnowledgeReview />)

    expect(html).toContain('继续当前会话')
    await buttonClickHandlers.get('继续当前会话')?.()
    expect(setShouldResumeSessionOnOpenMock).toHaveBeenCalledWith(true)
  })

  it('当存在当前会话时右侧摘要优先显示 session.mode', () => {
    storeState.currentSession = {
      session_id: 'session-1',
      status: 'in_progress',
      mode: 'detailed_qa',
      title: '恢复中的会话',
      opening_prompt: '继续作答',
      initial_hints: [],
      turn_index: 2,
    }
    storeState.shouldResumeSessionOnOpen = false
    storeState.selectedMode = 'light_recall'

    const html = renderToStaticMarkup(<KnowledgeReview />)

    expect(html).toContain('当前模式')
    expect(html).toContain('细致提问')
  })
})
