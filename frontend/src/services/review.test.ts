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
})
