import { beforeEach, describe, expect, it, vi } from 'vitest'

import { authTokenStore } from '@/lib/authTokenStore'
import {
  AUTH_EXPIRED_MESSAGE,
  GATEWAY_UNAVAILABLE_MESSAGE,
  requestBlob,
  requestEnvelope,
  requestJson,
} from './apiClient'

const storage = new Map<string, string>()

const jsonResponse = (payload: unknown, init: { ok?: boolean; status?: number } = {}) => ({
  ok: init.ok ?? true,
  status: init.status ?? 200,
  json: vi.fn().mockResolvedValue(payload),
}) as unknown as Response

describe('apiClient', () => {
  beforeEach(() => {
    storage.clear()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
      removeItem: vi.fn((key: string) => storage.delete(key)),
      clear: vi.fn(() => storage.clear()),
    })
  })

  it('adds authentication and serializes JSON without losing caller headers', async () => {
    authTokenStore.setToken('api-token')
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))

    await requestJson('/api/v1/example', {
      method: 'POST',
      headers: { 'X-Test': '1' },
      json: { title: 'demo' },
      fetchImpl,
    })

    const [, init] = fetchImpl.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(headers.get('Authorization')).toBe('Bearer api-token')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('X-Test')).toBe('1')
    expect(init.body).toBe(JSON.stringify({ title: 'demo' }))
  })

  it('passes FormData and AbortSignal without forcing a content type', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({ uploaded: true }))
    const formData = new FormData()
    formData.append('file', new Blob(['demo']), 'demo.md')
    const controller = new AbortController()

    await requestJson('/api/v1/upload', {
      method: 'POST',
      body: formData,
      signal: controller.signal,
      fetchImpl,
    })

    const [, init] = fetchImpl.mock.calls[0] as [string, RequestInit]
    expect(new Headers(init.headers).has('Content-Type')).toBe(false)
    expect(init.body).toBe(formData)
    expect(init.signal).toBe(controller.signal)
  })

  it('unwraps API envelopes and preserves backend error messages', async () => {
    const successFetch = vi.fn().mockResolvedValue(jsonResponse({ code: 200, data: { id: '1' } }))
    await expect(requestEnvelope('/api/v1/example', { fetchImpl: successFetch })).resolves.toEqual({ id: '1' })

    const errorFetch = vi.fn().mockResolvedValue(jsonResponse(
      { code: 422, message: '参数不合法' },
      { ok: false, status: 422 },
    ))
    await expect(requestEnvelope('/api/v1/example', { fetchImpl: errorFetch })).rejects.toThrow('参数不合法')
  })

  it('clears authentication on 401 responses', async () => {
    authTokenStore.setToken('expired-token')
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({}, { ok: false, status: 401 }))

    await expect(requestJson('/api/v1/private', { fetchImpl })).rejects.toThrow(AUTH_EXPIRED_MESSAGE)
    expect(authTokenStore.getSnapshot()).toBeNull()
  })

  it.each([502, 503, 504])('normalizes gateway status %s', async (status) => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(
      { message: 'upstream detail' },
      { ok: false, status },
    ))

    await expect(requestJson('/api/v1/example', { fetchImpl })).rejects.toThrow(GATEWAY_UNAVAILABLE_MESSAGE)
  })

  it('returns blobs without attempting JSON decoding', async () => {
    const blob = new Blob(['pdf'], { type: 'application/pdf' })
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      blob: vi.fn().mockResolvedValue(blob),
    } as unknown as Response)

    await expect(requestBlob('/api/v1/download', { fetchImpl })).resolves.toBe(blob)
  })

  it('preserves AbortError instead of converting cancellation into an API error', async () => {
    const abortError = new DOMException('aborted', 'AbortError')
    const fetchImpl = vi.fn().mockRejectedValue(abortError)

    await expect(requestJson('/api/v1/example', { fetchImpl })).rejects.toBe(abortError)
  })
})
