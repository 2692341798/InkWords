import { beforeEach, describe, expect, it, vi } from 'vitest'

const fetchEventSourceMock = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))

vi.mock('@microsoft/fetch-event-source', async (importOriginal) => {
  const original = await importOriginal<typeof import('@microsoft/fetch-event-source')>()
  return {
    ...original,
    fetchEventSource: fetchEventSourceMock,
  }
})

import { GATEWAY_UNAVAILABLE_MESSAGE, LOCAL_ACCESS_DENIED_MESSAGE } from './apiClient'
import { fetchEventSourceLocal } from './sse'

describe('fetchEventSourceLocal', () => {
  beforeEach(() => {
    fetchEventSourceMock.mockClear()
  })

  it('strips request identity and reports unexpected unauthorized stream opens', async () => {
    void fetchEventSourceLocal('/api/v1/tasks/task-1/stream', {
      headers: { Authorization: 'Bearer caller-token', 'X-Test': '1' },
      onmessage: vi.fn(),
    })

    const [, options] = fetchEventSourceMock.mock.calls[0] as [string, {
      headers: Record<string, string>
      onopen: (response: Response) => Promise<void>
    }]
    expect(options.headers).toEqual({ 'X-Test': '1' })

    const response = {
      ok: false,
      status: 401,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: vi.fn().mockResolvedValue({ message: 'unauthorized' }),
    } as unknown as Response
    await expect(options.onopen(response)).rejects.toThrow(LOCAL_ACCESS_DENIED_MESSAGE)
  })

  it('does not send a legacy browser token on local-workspace streams', () => {
    const getItem = vi.fn().mockReturnValue('stale-stream-token')
    vi.stubGlobal('localStorage', { getItem })
    void fetchEventSourceLocal('/api/v1/tasks/task-1/stream', { onmessage: vi.fn() })

    const [, options] = fetchEventSourceMock.mock.calls[0] as [string, {
      headers: Record<string, string>
    }]
    expect(options.headers).not.toHaveProperty('Authorization')
    expect(getItem).not.toHaveBeenCalled()
  })

  it('normalizes unavailable gateways and rejects implicit stream retries', async () => {
    const callerOnError = vi.fn()
    void fetchEventSourceLocal('/api/v1/tasks/task-1/stream', {
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
