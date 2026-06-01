import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  capturedHook,
  createSessionMock,
  getSessionMock,
  loadHistoryMock,
  loadRecommendationMock,
  setCurrentSessionMock,
  setShouldResumeSessionOnOpenMock,
  storeState,
} = vi.hoisted(() => {
  const state = {
    currentSession: null,
    selectedMode: 'light_recall' as const,
    clearSessionState: vi.fn(),
    setCurrentSession: vi.fn(),
    setShouldResumeSessionOnOpen: vi.fn(),
    setLatestHint: vi.fn(),
    setLatestStageFeedback: vi.fn(),
    setFinalFeedback: vi.fn(),
    loadHistory: vi.fn().mockResolvedValue(undefined),
    loadRecommendation: vi.fn().mockResolvedValue(undefined),
  }

  return {
    capturedHook: {
      current: null as null | ReturnType<typeof import('./useKnowledgeReview').useKnowledgeReview>,
    },
    createSessionMock: vi.fn(),
    getSessionMock: vi.fn(),
    loadHistoryMock: state.loadHistory,
    loadRecommendationMock: state.loadRecommendation,
    setCurrentSessionMock: state.setCurrentSession,
    setShouldResumeSessionOnOpenMock: state.setShouldResumeSessionOnOpen,
    storeState: state,
  }
})

vi.mock('@/services/review', () => ({
  reviewService: {
    createSession: createSessionMock,
    getSession: getSessionMock,
  },
}))

vi.mock('@/store/reviewStore', () => {
  const useReviewStore = () => storeState
  useReviewStore.getState = () => storeState

  return {
    useReviewStore,
  }
})

import { useKnowledgeReview } from './useKnowledgeReview'

function HookHarness() {
  capturedHook.current = useKnowledgeReview()
  return null
}

describe('useKnowledgeReview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    capturedHook.current = null
    storeState.selectedMode = 'light_recall'
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: undefined,
    })
  })

  it('uses the explicit mode override when starting a session from a dedicated entry action', async () => {
    createSessionMock.mockResolvedValue({
      session_id: 'session-1',
      status: 'in_progress',
      mode: 'detailed_qa',
      title: '随机文章',
      opening_prompt: '先说主线',
      initial_hints: [],
      session_outline: {
        summary: '随机文章摘要',
        main_question: '随机文章主要在解决什么问题？',
        core_concepts: ['主线'],
        process_steps: [],
        application_cases: [],
        checkpoints: ['先说主线'],
      },
      turn_index: 1,
    })

    renderToStaticMarkup(<HookHarness />)

    await (
      capturedHook.current?.startSession as unknown as (
        card: { note_path: string },
        entryType: 'manual_random',
        modeOverride?: 'light_recall' | 'detailed_qa'
      ) => Promise<void>
    )(
      {
        note_path: 'wiki/concepts/random.md',
      },
      'manual_random',
      'detailed_qa',
    )

    expect(createSessionMock).toHaveBeenCalledWith({
      note_path: 'wiki/concepts/random.md',
      mode: 'detailed_qa',
      entry_type: 'manual_random',
    })
  })

  it('keeps the explicit resume intent when restoring an active session on page open', async () => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: {
        localStorage: {
          getItem: vi.fn().mockReturnValue('session-1'),
          setItem: vi.fn(),
          removeItem: vi.fn(),
        },
      },
    })

    getSessionMock.mockResolvedValue({
      session_id: 'session-1',
      status: 'in_progress',
      mode: 'light_recall',
      title: '恢复的会话',
      opening_prompt: '继续回答',
      initial_hints: [],
      session_outline: {
        summary: '恢复会话摘要',
        main_question: '恢复的会话主要在讲什么？',
        core_concepts: ['主线'],
        process_steps: [],
        application_cases: [],
        checkpoints: ['继续回答主线'],
      },
      turn_index: 2,
    })

    renderToStaticMarkup(<HookHarness />)

    await capturedHook.current?.initialize()

    expect(loadRecommendationMock).toHaveBeenCalledTimes(1)
    expect(loadHistoryMock).toHaveBeenCalledWith(5)
    expect(setCurrentSessionMock).toHaveBeenCalledWith(
      expect.objectContaining({
        session_id: 'session-1',
        title: '恢复的会话',
      }),
    )
    expect(setShouldResumeSessionOnOpenMock).not.toHaveBeenCalledWith(false)
  })
})
