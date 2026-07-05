// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ReviewSessionResponse } from '@/services/review'

const {
  completeReadingMock,
  createSessionMock,
  getSessionMock,
  loadHistoryMock,
  loadRecommendationMock,
  requestHintMock,
  setCurrentSessionMock,
  setShouldResumeSessionOnOpenMock,
  storeState,
} = vi.hoisted(() => {
  const state = {
    currentSession: null as ReviewSessionResponse | null,
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
    completeReadingMock: vi.fn(),
    createSessionMock: vi.fn(),
    getSessionMock: vi.fn(),
    loadHistoryMock: state.loadHistory,
    loadRecommendationMock: state.loadRecommendation,
    requestHintMock: vi.fn(),
    setCurrentSessionMock: state.setCurrentSession,
    setShouldResumeSessionOnOpenMock: state.setShouldResumeSessionOnOpen,
    storeState: state,
  }
})

vi.mock('@/services/review', () => ({
  reviewService: {
    completeReading: completeReadingMock,
    createSession: createSessionMock,
    getSession: getSessionMock,
    requestHint: requestHintMock,
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

describe('useKnowledgeReview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    storeState.selectedMode = 'light_recall'
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

    const { result } = renderHook(() => useKnowledgeReview())

    await act(async () => {
      await (result.current.startSession as unknown as (
        card: { note_path: string },
        entryType: 'manual_random',
        modeOverride?: 'light_recall' | 'detailed_qa'
      ) => Promise<void>)(
        { note_path: 'wiki/concepts/random.md' },
        'manual_random',
        'detailed_qa',
      )
    })

    expect(createSessionMock).toHaveBeenCalledWith({
      note_path: 'wiki/concepts/random.md',
      mode: 'detailed_qa',
      entry_type: 'manual_random',
    })
  })

  it('keeps the explicit resume intent when restoring an active session on page open', async () => {
    vi.spyOn(window.localStorage, 'getItem').mockReturnValue('session-1')
    vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {})
    vi.spyOn(window.localStorage, 'removeItem').mockImplementation(() => {})

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

    const { result } = renderHook(() => useKnowledgeReview())

    await act(async () => {
      await result.current.initialize()
    })

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

  it('preserves the session when completing the reading phase', async () => {
    const session: ReviewSessionResponse = {
      session_id: 'session-1', status: 'created', phase: 'reading', mode: 'light_recall', title: '并发控制',
      reading_content: '完整原文', opening_prompt: '请先阅读', initial_hints: [],
      session_outline: { summary: '摘要', main_question: '主旨是什么？', core_concepts: [], process_steps: [], application_cases: [], checkpoints: [] },
      turn_index: 1,
    }
    storeState.currentSession = session
    completeReadingMock.mockResolvedValue({ session_id: 'session-1', status: 'created', phase: 'recalling' })
    const { result } = renderHook(() => useKnowledgeReview())

    await act(async () => { await result.current.completeReading() })

    expect(setCurrentSessionMock).toHaveBeenCalledWith({ ...session, phase: 'recalling' })
  })

  it('sends the current draft answer when requesting an AI hint', async () => {
    storeState.currentSession = {
      session_id: 'session-1', status: 'created', phase: 'recalling', mode: 'light_recall', title: '并发控制',
      reading_content: '完整原文', opening_prompt: '请复述', initial_hints: [],
      session_outline: { summary: '摘要', main_question: '主旨是什么？', core_concepts: [], process_steps: [], application_cases: [], checkpoints: [] },
      turn_index: 1, turns: [],
    }
    requestHintMock.mockResolvedValue({ session_id: 'session-1', hint_text: '想想它限制的是数量还是频率。', remaining_hint_count: 2 })
    const { result } = renderHook(() => useKnowledgeReview())

    await act(async () => { await result.current.requestHint('  我记得它用于保护系统  ') })

    expect(requestHintMock).toHaveBeenCalledWith('session-1', '我记得它用于保护系统')
  })
})
