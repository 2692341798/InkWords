import { beforeEach, describe, expect, it, vi } from 'vitest'
import { userService } from './user'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('userService', () => {
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
    globalThis.localStorage.setItem('token', 'user-token')
  })

  it('loads dashboard stats and profile using auth headers', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 200,
          data: {
            tokens_used: 1000,
            estimated_cost: 1.2,
            total_articles: 6,
            total_words: 5000,
            tech_stack_stats: [],
          },
        }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          code: 200,
          data: {
            username: 'InkWords',
            email: 'test@example.com',
            avatar_url: '',
            subscription_tier: 1,
            token_limit: 100000,
          },
        }),
      } as Response)

    const result = await userService.getDashboardData()

    expect(result.stats.tokens_used).toBe(1000)
    expect(result.profile.username).toBe('InkWords')
    expect(mockFetch).toHaveBeenCalledTimes(2)
    const [, statsInit] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(new Headers(statsInit.headers).get('Authorization')).toBe('Bearer user-token')
  })

  it('updates username with json payload', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: null,
      }),
    } as Response)

    await userService.updateUsername('new-name')

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/user/profile')
    expect(init.method).toBe('PUT')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
    expect(init.body).toBe(JSON.stringify({ username: 'new-name' }))
  })

  it('uploads avatar and returns avatar url', async () => {
    const formData = new FormData()
    formData.append('avatar', new Blob(['avatar']))
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          avatar_url: 'https://example.com/avatar.png',
        },
      }),
    } as Response)

    await expect(userService.uploadAvatar(formData)).resolves.toBe('https://example.com/avatar.png')

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/user/avatar')
    expect(init.method).toBe('POST')
  })
})
