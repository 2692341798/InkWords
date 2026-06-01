import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reviewService } from './review'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('reviewService', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    vi.stubGlobal('fetch', mockFetch)
    storage.clear()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => {
        storage.set(key, value)
      }),
      removeItem: vi.fn((key: string) => {
        storage.delete(key)
      }),
      clear: vi.fn(() => {
        storage.clear()
      }),
    })
    globalThis.localStorage.setItem('token', 'review-token')
  })

  it('calls GET /api/v1/review/notes with query params', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: { items: [], total: 0, page: 1, page_size: 20 },
      }),
    } as Response)

    await reviewService.listNotes({ query: '并发', page: 1, pageSize: 20 })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]

    expect(url).toBe('/api/v1/review/notes?query=%E5%B9%B6%E5%8F%91&page=1&page_size=20')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer review-token')
  })

  it('calls GET /api/v1/review/today with auth header', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          note_path: 'wiki/concepts/review.md',
          title: '复习主题',
          source_title: '知识库',
          review_reason: '今日推荐',
          estimated_minutes: 5,
          available_modes: ['light_recall', 'detailed_qa'],
        },
      }),
    } as Response)

    await reviewService.getToday()

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]

    expect(url).toBe('/api/v1/review/today')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer review-token')
  })

  it('calls GET /api/v1/review/history with limit query', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          items: [],
          limit: 5,
        },
      }),
    } as Response)

    await reviewService.getHistory(5)

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]

    expect(url).toBe('/api/v1/review/history?limit=5')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer review-token')
  })

  it('preserves structured review feedback fields returned by respond', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          session_id: 'session-1',
          session_status: 'in_progress',
          turn_index: 3,
          stage_feedback: '你已经答到主线。',
          current_round_goal: '补充关键概念',
          review_feedback: {
            judgement: '部分答对',
            hit_points: ['答到主线'],
            missed_points: ['没提速率限制'],
            suggestion: '补充速率限制和适用场景',
          },
          completed: false,
          final_feedback: {
            summary: '',
            strengths: [],
            gaps: [],
            next_focus: [],
          },
        },
      }),
    } as Response)

    const response = await reviewService.respond('session-1', { answer: '并发控制是控制任务数' })

    expect(response.review_feedback.judgement).toBe('部分答对')
    expect(response.current_round_goal).toBe('补充关键概念')
  })
})
