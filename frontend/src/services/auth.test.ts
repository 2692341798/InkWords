import { beforeEach, describe, expect, it, vi } from 'vitest'
import { authService, buildAuthHeaders } from './auth'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('authService', () => {
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
  })

  it('builds auth headers from local storage and preserves existing headers', () => {
    globalThis.localStorage.setItem('token', 'auth-token')

    const headers = buildAuthHeaders({ 'X-Test': '1' })

    expect(headers.get('Authorization')).toBe('Bearer auth-token')
    expect(headers.get('X-Test')).toBe('1')
  })

  it('fetches captcha payload from the auth endpoint', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          captcha_id: 'captcha-1',
          image: 'data:image/png;base64,abc',
        },
      }),
    } as Response)

    await expect(authService.fetchCaptcha()).resolves.toEqual({
      captcha_id: 'captcha-1',
      image: 'data:image/png;base64,abc',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/auth/captcha')
    expect(init?.method).toBeUndefined()
  })

  it('posts login payload and returns the parsed auth data', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        code: 200,
        data: {
          token: 'login-token',
        },
      }),
    } as Response)

    await expect(
      authService.login({
        email: 'user@example.com',
        password: 'secret',
        captcha_id: 'captcha-1',
        captcha_value: '1234',
        remember_me: true,
      }),
    ).resolves.toEqual({
      token: 'login-token',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/auth/login')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
    expect(init.body).toBe(
      JSON.stringify({
        email: 'user@example.com',
        password: 'secret',
        captcha_id: 'captcha-1',
        captcha_value: '1234',
        remember_me: true,
      }),
    )
  })

  it('clears the token and throws a normalized error on unauthorized register', async () => {
    globalThis.localStorage.setItem('token', 'expired-token')
    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        code: 401,
        message: '登录已过期，请重新登录',
      }),
    } as Response)

    await expect(
      authService.register({
        username: 'InkWords',
        email: 'user@example.com',
        password: 'secret',
        captcha_id: 'captcha-1',
        captcha_value: '1234',
      }),
    ).rejects.toThrow('登录已过期，请重新登录')

    expect(globalThis.localStorage.getItem('token')).toBeNull()
  })
})
