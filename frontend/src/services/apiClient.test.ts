import { describe, expect, it, vi } from 'vitest'
import {
  GATEWAY_UNAVAILABLE_MESSAGE,
  LOCAL_ACCESS_DENIED_MESSAGE,
  requestBlob,
  requestEnvelope,
  requestJson,
} from './apiClient'

const jsonResponse = (payload: unknown, init: { ok?: boolean; status?: number } = {}) => ({
  ok: init.ok ?? true,
  status: init.status ?? 200,
  json: vi.fn().mockResolvedValue(payload),
}) as unknown as Response

describe('apiClient', () => {
  it('serializes JSON, preserves ordinary headers, and strips request identity', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))

    await requestJson('/api/v1/example', {
      method: 'POST',
      headers: { 'X-Test': '1', Authorization: 'Bearer caller-token' },
      json: { title: 'demo' },
      fetchImpl,
    })

    const [, init] = fetchImpl.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(headers.has('Authorization')).toBe(false)
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('X-Test')).toBe('1')
    expect(init.body).toBe(JSON.stringify({ title: 'demo' }))
  })

  it('does not send a legacy browser token in local-workspace mode', async () => {
    const getItem = vi.fn().mockReturnValue('stale-browser-token')
    vi.stubGlobal('localStorage', { getItem })
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))

    await requestJson('/api/v1/example', { fetchImpl })

    const [, init] = fetchImpl.mock.calls[0] as [string, RequestInit]
    expect(new Headers(init.headers).has('Authorization')).toBe(false)
    expect(getItem).not.toHaveBeenCalled()
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

  it('reports a local configuration error on unexpected 401 responses', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse({}, { ok: false, status: 401 }))

    await expect(requestJson('/api/v1/private', { fetchImpl })).rejects.toThrow(LOCAL_ACCESS_DENIED_MESSAGE)
  })

  it.each([502, 503, 504])('normalizes gateway status %s', async (status) => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(
      { message: 'upstream detail' },
      { ok: false, status },
    ))

    await expect(requestJson('/api/v1/example', { fetchImpl })).rejects.toThrow(GATEWAY_UNAVAILABLE_MESSAGE)
  })

  it('keeps actionable local-workspace initialization errors', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(
      { code: 'LOCAL_WORKSPACE_UNAVAILABLE', message: '本地工作区尚未初始化完成。请检查数据库连接后重试。' },
      { ok: false, status: 503 },
    ))

    await expect(requestJson('/api/v1/textbook-projects', { fetchImpl })).rejects.toThrow('本地工作区尚未初始化完成')
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
