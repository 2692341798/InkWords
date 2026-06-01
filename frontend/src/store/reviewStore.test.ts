import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useReviewStore } from './reviewStore'
import { type ReviewCardResponse, reviewService } from '@/services/review'

vi.mock('@/services/review', () => ({
  reviewService: {
    getToday: vi.fn(),
    pickRandom: vi.fn(),
    listNotes: vi.fn(),
    getHistory: vi.fn(),
  },
}))

const mockedReviewService = vi.mocked(reviewService)
const randomCard: ReviewCardResponse = {
  note_path: 'wiki/concepts/random.md',
  title: '随机复习',
  source_title: '知识库',
  review_reason: '这是一次随机抽取的复习内容',
  estimated_minutes: 5,
  available_modes: ['light_recall', 'detailed_qa'],
}
const refreshedCard: ReviewCardResponse = {
  note_path: 'wiki/concepts/refreshed.md',
  title: '刷新后的推荐',
  source_title: '知识库',
  review_reason: '换一篇继续复习',
  estimated_minutes: 8,
  available_modes: ['light_recall'],
}

describe('useReviewStore', () => {
  beforeEach(() => {
    useReviewStore.getState().reset()
    vi.clearAllMocks()
  })

  it('loads the automatic entry card from the random-pick endpoint', async () => {
    mockedReviewService.pickRandom.mockResolvedValue(randomCard)

    await useReviewStore.getState().loadRecommendation()

    expect(mockedReviewService.getToday).not.toHaveBeenCalled()
    expect(mockedReviewService.pickRandom).toHaveBeenCalledTimes(1)
    expect(useReviewStore.getState().recommendationCard?.note_path).toBe(randomCard.note_path)
  })

  it('refreshes the recommendation card with a different random article when available', async () => {
    mockedReviewService.pickRandom.mockResolvedValue(refreshedCard)

    useReviewStore.setState({
      recommendationCard: randomCard,
    })

    await useReviewStore.getState().refreshRecommendation()

    expect(useReviewStore.getState().recommendationCard?.note_path).toBe(refreshedCard.note_path)
  })

  it('keeps the current random card when refresh returns the same article', async () => {
    mockedReviewService.pickRandom.mockResolvedValue(randomCard)

    useReviewStore.setState({
      recommendationCard: randomCard,
    })

    await useReviewStore.getState().refreshRecommendation()

    expect(useReviewStore.getState().recommendationCard?.note_path).toBe(randomCard.note_path)
  })

  it('retries refresh when the first random result matches the current article', async () => {
    mockedReviewService.pickRandom
      .mockResolvedValueOnce(randomCard)
      .mockResolvedValueOnce(refreshedCard)

    useReviewStore.setState({
      recommendationCard: randomCard,
    })

    await useReviewStore.getState().refreshRecommendation()

    expect(mockedReviewService.pickRandom).toHaveBeenCalledTimes(2)
    expect(useReviewStore.getState().recommendationCard?.note_path).toBe(refreshedCard.note_path)
  })

  it('loads note options into store', async () => {
    mockedReviewService.listNotes.mockResolvedValue({
      items: [
        {
          note_path: 'wiki/concepts/concurrency.md',
          title: '并发控制',
          source_title: '并发专题',
          preferred_mode: 'detailed_qa',
          last_reviewed_at: null,
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })

    await useReviewStore.getState().loadNotes('并发')

    expect(useReviewStore.getState().noteOptions).toHaveLength(1)
    expect(mockedReviewService.listNotes).toHaveBeenCalledWith({ query: '并发' })
  })

  it('updates selected mode, tracks resume intent, and reset clears review state', () => {
    useReviewStore.getState().setSelectedMode('detailed_qa')
    useReviewStore.getState().setShouldResumeSessionOnOpen(true)
    useReviewStore.setState({
      recommendationCard: randomCard,
    })

    expect(useReviewStore.getState().selectedMode).toBe('detailed_qa')
    expect(useReviewStore.getState().shouldResumeSessionOnOpen).toBe(true)

    useReviewStore.getState().reset()

    expect(useReviewStore.getState().selectedMode).toBe('light_recall')
    expect(useReviewStore.getState().shouldResumeSessionOnOpen).toBe(false)
    expect(useReviewStore.getState().recommendationCard).toBeNull()
    expect(useReviewStore.getState().noteOptions).toEqual([])
  })

  it('loads recent review history into store', async () => {
    mockedReviewService.getHistory.mockResolvedValue({
      items: [
        {
          session_id: 'session-1',
          note_path: 'wiki/concepts/history.md',
          title: '并发控制',
          source_title: '后端系列',
          mode: 'light_recall',
          status: 'completed',
          summary: '已经讲清楚主线',
          reviewed_at: '2026-05-27T09:00:00Z',
        },
      ],
      limit: 5,
    })

    await useReviewStore.getState().loadHistory(5)

    expect(mockedReviewService.getHistory).toHaveBeenCalledWith(5)
    expect(useReviewStore.getState().historyItems).toHaveLength(1)
    expect(useReviewStore.getState().historyItems[0]?.title).toBe('并发控制')
  })

  it('clears the active session state before the user chooses a new article', () => {
    useReviewStore.setState({
      currentSession: {
        session_id: 'session-1',
        status: 'in_progress',
        mode: 'light_recall',
        title: '旧的复习会话',
        opening_prompt: '先讲一讲',
        initial_hints: [],
        turn_index: 2,
      },
      shouldResumeSessionOnOpen: true,
      latestStageFeedback: '上一轮反馈',
      latestHint: '上一轮提示',
      finalFeedback: {
        summary: '旧总结',
        strengths: [],
        gaps: [],
        next_focus: [],
      },
    })

    useReviewStore.getState().clearSessionState()

    expect(useReviewStore.getState().currentSession).toBeNull()
    expect(useReviewStore.getState().shouldResumeSessionOnOpen).toBe(false)
    expect(useReviewStore.getState().latestStageFeedback).toBeNull()
    expect(useReviewStore.getState().latestHint).toBeNull()
    expect(useReviewStore.getState().finalFeedback).toBeNull()
  })
})
