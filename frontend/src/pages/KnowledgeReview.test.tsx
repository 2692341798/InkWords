import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { KnowledgeReview } from './KnowledgeReview'
import type { ReviewSessionResponse } from '@/services/review'

const {
  buttonClickHandlers,
  capturedEntryCardsProps,
  clearSessionMock,
  finishMock,
  initializeMock,
  requestHintMock,
  respondMock,
  setShouldResumeSessionOnOpenMock,
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

  const state: {
    recommendationCard: typeof recommendationCard
    isLoadingRecommendation: boolean
    currentSession: ReviewSessionResponse | null
    shouldResumeSessionOnOpen: boolean
    latestStageFeedback: string | null
    latestHint: string | null
    finalFeedback: null
    historyItems: never[]
    isLoadingHistory: boolean
    noteOptions: never[]
    isLoadingNotes: boolean
    selectedMode: 'light_recall' | 'detailed_qa'
    loadRecommendation: ReturnType<typeof vi.fn>
    refreshRecommendation: ReturnType<typeof vi.fn>
    loadNotes: ReturnType<typeof vi.fn>
    loadHistory: ReturnType<typeof vi.fn>
    setShouldResumeSessionOnOpen: ReturnType<typeof vi.fn>
    setSelectedMode: ReturnType<typeof vi.fn>
  } = {
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

  it('从推荐文章直接开始沉浸阅读，不再要求预选训练模式', async () => {
    renderToStaticMarkup(<KnowledgeReview />)

    const props = capturedEntryCardsProps.current as null | {
      onStartRecommendation?: () => Promise<void>
    }

    expect(props?.onStartRecommendation).toBeTypeOf('function')
    await props?.onStartRecommendation?.()
    expect(startSessionMock).toHaveBeenCalledWith(
      expect.objectContaining({ note_path: 'wiki/concepts/random.md' }),
      'manual_random',
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
      session_outline: {
        summary: '恢复中的会话摘要',
        main_question: '继续回答这篇文章的主问题',
        core_concepts: ['当前主线'],
        process_steps: [],
        application_cases: [],
        checkpoints: ['继续补充主线'],
      },
      turn_index: 2,
    }
    storeState.shouldResumeSessionOnOpen = false

    const html = renderToStaticMarkup(<KnowledgeReview />)

    expect(html).toContain('继续上次的复习')
    await buttonClickHandlers.get('继续')?.()
    expect(setShouldResumeSessionOnOpenMock).toHaveBeenCalledWith(true)
  })

  it('入口态不再展示会话模式与任务摘要', () => {
    storeState.currentSession = {
      session_id: 'session-1',
      status: 'in_progress',
      mode: 'detailed_qa',
      title: '恢复中的会话',
      opening_prompt: '继续作答',
      initial_hints: [],
      session_outline: {
        summary: '恢复中的会话摘要',
        main_question: '继续回答这篇文章的主问题',
        core_concepts: ['当前主线'],
        process_steps: [],
        application_cases: [],
        checkpoints: ['继续补充主线'],
      },
      turn_index: 2,
    }
    storeState.shouldResumeSessionOnOpen = false
    storeState.selectedMode = 'light_recall'

    const html = renderToStaticMarkup(<KnowledgeReview />)

    expect(html).not.toContain('当前模式')
    expect(html).not.toContain('任务摘要')
  })
})
