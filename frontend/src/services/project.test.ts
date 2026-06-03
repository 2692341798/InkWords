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

  it('creates a parse task for async archive parsing', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        task_id: 'task-parse-1',
        status: 'queued',
        stream_url: '/api/v1/tasks/task-parse-1/stream',
      }),
    } as Response)

    const file = new File(['zip'], 'courseware.zip')
    await expect(projectService.createParseTask(file)).resolves.toEqual({
      task_id: 'task-parse-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-parse-1/stream',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/tasks/parse')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer project-token')
    expect(String(init.body)).toContain('"kind":"parse_archive"')
    expect(String(init.body)).toContain('"filename":"courseware.zip"')
  })

  it('creates a parse_file task for large non-zip files', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        task_id: 'task-parse-file-1',
        status: 'queued',
        stream_url: '/api/v1/tasks/task-parse-file-1/stream',
      }),
    } as Response)

    const file = new File(['pdf'], 'course.pdf', { type: 'application/pdf' })
    await expect(projectService.createParseTask(file)).resolves.toEqual({
      task_id: 'task-parse-file-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-parse-file-1/stream',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/tasks/parse')
    expect(init.method).toBe('POST')
    expect(String(init.body)).toContain('"kind":"parse_file"')
    expect(String(init.body)).toContain('"filename":"course.pdf"')
  })

  it('fetches parse task snapshot with result payload', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        id: 'task-parse-1',
        status: 'succeeded',
        result: {
          source_content: 'parsed content',
        },
      }),
    } as Response)

    await expect(projectService.getTaskSnapshot('task-parse-1')).resolves.toEqual({
      id: 'task-parse-1',
      status: 'succeeded',
      result: {
        source_content: 'parsed content',
      },
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/tasks/task-parse-1')
    expect(init.method).toBe('GET')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer project-token')
  })
})
