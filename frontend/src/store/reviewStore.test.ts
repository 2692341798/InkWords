import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useReviewStore } from './reviewStore'
import { reviewService } from '@/services/review'

vi.mock('@/services/review', () => ({
  reviewService: {
    getToday: vi.fn(),
    pickRandom: vi.fn(),
    listNotes: vi.fn(),
    getHistory: vi.fn(),
  },
}))

const mockedReviewService = vi.mocked(reviewService)

describe('useReviewStore', () => {
  beforeEach(() => {
    useReviewStore.getState().reset()
    vi.clearAllMocks()
  })

  it('loads today card into store', async () => {
    mockedReviewService.getToday.mockResolvedValue({
      note_path: 'wiki/concepts/today.md',
      title: '今日复习',
      source_title: '知识库',
      review_reason: '今天适合回顾这篇内容',
      estimated_minutes: 5,
      available_modes: ['light_recall', 'detailed_qa'],
    })

    await useReviewStore.getState().loadToday()

    expect(useReviewStore.getState().todayCard?.note_path).toBe('wiki/concepts/today.md')
  })

  it('loads random card and note options into store', async () => {
    mockedReviewService.pickRandom.mockResolvedValue({
      note_path: 'wiki/concepts/random.md',
      title: '随机复习',
      source_title: '知识库',
      review_reason: '随机抽题',
      estimated_minutes: 8,
      available_modes: ['light_recall'],
    })
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

    await useReviewStore.getState().loadRandom()
    await useReviewStore.getState().loadNotes('并发')

    expect(useReviewStore.getState().randomCard?.note_path).toBe('wiki/concepts/random.md')
    expect(useReviewStore.getState().noteOptions).toHaveLength(1)
    expect(mockedReviewService.listNotes).toHaveBeenCalledWith({ query: '并发' })
  })

  it('updates selected mode and reset clears review state', () => {
    useReviewStore.getState().setSelectedMode('detailed_qa')
    useReviewStore.setState({
      todayCard: {
        note_path: 'wiki/concepts/today.md',
        title: '今日复习',
        source_title: '知识库',
        review_reason: '今天适合回顾这篇内容',
        estimated_minutes: 5,
        available_modes: ['light_recall', 'detailed_qa'],
      },
    })

    expect(useReviewStore.getState().selectedMode).toBe('detailed_qa')

    useReviewStore.getState().reset()

    expect(useReviewStore.getState().selectedMode).toBe('light_recall')
    expect(useReviewStore.getState().todayCard).toBeNull()
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
})
