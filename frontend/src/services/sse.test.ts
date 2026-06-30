import { beforeEach, describe, expect, it, vi } from 'vitest'

const fetchEventSourceMock = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))

vi.mock('@microsoft/fetch-event-source', async (importOriginal) => {
  const original = await importOriginal<typeof import('@microsoft/fetch-event-source')>()
  return {
    ...original,
    fetchEventSource: fetchEventSourceMock,
  }
})

import { authTokenStore } from '@/lib/authTokenStore'
import { AUTH_EXPIRED_MESSAGE, GATEWAY_UNAVAILABLE_MESSAGE } from './apiClient'
import { buildAuthHeader, fetchEventSourceWithAuth } from './sse'

const storage = new Map<string, string>()

describe('buildAuthHeader', () => {
  beforeEach(() => {
    fetchEventSourceMock.mockClear()
    storage.clear()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
      removeItem: vi.fn((key: string) => storage.delete(key)),
    })
  })

  it('returns empty object when token missing', () => {
    expect(buildAuthHeader(null)).toEqual({})
  })

  it('returns Bearer token header when token present', () => {
    expect(buildAuthHeader('t')).toEqual({ Authorization: 'Bearer t' })
  })

  it('uses shared auth handling and rejects unauthorized stream opens', async () => {
    authTokenStore.setToken('stream-token')
    void fetchEventSourceWithAuth('/api/v1/tasks/task-1/stream', { onmessage: vi.fn() })

    const [, options] = fetchEventSourceMock.mock.calls[0] as [string, {
      headers: Record<string, string>
      onopen: (response: Response) => Promise<void>
    }]
    expect(options.headers.Authorization).toBe('Bearer stream-token')

    const response = {
      ok: false,
      status: 401,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: vi.fn().mockResolvedValue({ message: 'unauthorized' }),
    } as unknown as Response
    await expect(options.onopen(response)).rejects.toThrow(AUTH_EXPIRED_MESSAGE)
    expect(authTokenStore.getSnapshot()).toBeNull()
  })

  it('normalizes unavailable gateways and rejects implicit stream retries', async () => {
    const callerOnError = vi.fn()
    void fetchEventSourceWithAuth('/api/v1/tasks/task-1/stream', {
      onmessage: vi.fn(),
      onerror: callerOnError,
    })

    const [, options] = fetchEventSourceMock.mock.calls[0] as [string, {
      onopen: (response: Response) => Promise<void>
      onerror: (error: unknown) => void
    }]
    const response = {
      ok: false,
      status: 503,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: vi.fn().mockResolvedValue({ message: 'upstream unavailable' }),
    } as unknown as Response
    await expect(options.onopen(response)).rejects.toThrow(GATEWAY_UNAVAILABLE_MESSAGE)

    const streamError = new Error('stream failed')
    expect(() => options.onerror(streamError)).toThrow(streamError)
    expect(callerOnError).toHaveBeenCalledWith(streamError)
  })
})
