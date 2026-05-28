import { beforeEach, describe, expect, it, vi } from 'vitest'
import { projectService } from './project'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('projectService', () => {
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
    globalThis.localStorage.setItem('token', 'project-token')
  })

  it('uploads project parse form data with auth headers and returns parsed data', async () => {
    const formData = new FormData()
    formData.append('file', new Blob(['demo']), 'demo.md')

    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          source_content: 'parsed content',
        },
      }),
    } as Response)

    await expect(
      projectService.parseProjectFile(formData, new AbortController().signal),
    ).resolves.toEqual({
      data: {
        source_content: 'parsed content',
      },
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/project/parse')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer project-token')
    expect(init.body).toBe(formData)
  })

  it('clears token and throws normalized error on unauthorized parse', async () => {
    const formData = new FormData()
    formData.append('file', new Blob(['demo']), 'demo.md')

    mockFetch.mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        error: '登录已过期，请重新登录',
      }),
    } as Response)

    await expect(
      projectService.parseProjectFile(formData, new AbortController().signal),
    ).rejects.toThrow('登录已过期，请重新登录')

    expect(globalThis.localStorage.getItem('token')).toBeNull()
  })
})
