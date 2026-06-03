import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  buildContinueTaskPayload,
  buildGenerationTaskRequest,
  buildPolishTaskPayload,
  cancelGenerationTask,
  createGenerationTask,
  extractTaskChunkContent,
} from './generationTasks'

const mockFetch = vi.fn()
const storage = new Map<string, string>()

describe('generationTasks', () => {
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
    globalThis.localStorage.setItem('token', 'task-token')
  })

  it('maps single generation payload to task request', () => {
    expect(
      buildGenerationTaskRequest('generate_single', {
        source_type: 'file',
        source_content: 'hello',
        outline: [],
        scenario_mode: 'ebook_interpretation',
      }),
    ).toEqual({
      kind: 'generate_single',
      payload: {
        source_type: 'file',
        source_content: 'hello',
        outline: [],
        scenario_mode: 'ebook_interpretation',
      },
    })
  })

  it('builds continue and polish payloads with one task contract', () => {
    expect(buildContinueTaskPayload('blog-1')).toEqual({
      blog_id: 'blog-1',
    })

    expect(buildPolishTaskPayload('blog-2', '标题', '正文')).toEqual({
      blog_id: 'blog-2',
      title: '标题',
      content: '正文',
    })
  })

  it('extracts text content from task chunk payloads', () => {
    expect(extractTaskChunkContent('plain text chunk')).toBe('plain text chunk')
    expect(extractTaskChunkContent('{"content":"wrapped chunk"}')).toBe('wrapped chunk')
    expect(extractTaskChunkContent('{"status":"streaming","content":"chapter chunk"}')).toBe(
      'chapter chunk',
    )
  })

  it('creates generation task with auth header and returns task metadata', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        task_id: 'task-123',
        status: 'queued',
        stream_url: '/api/v1/tasks/task-123/stream',
      }),
    } as Response)

    await expect(
      createGenerationTask(
        buildGenerationTaskRequest('generate_series', {
          source_type: 'git',
          git_url: 'https://github.com/inkwords/demo',
        }),
      ),
    ).resolves.toEqual({
      task_id: 'task-123',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-123/stream',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/tasks/generation')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer task-token')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
  })

  it('cancels generation task with auth header', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        task_id: 'task-123',
        status: 'cancelled',
      }),
    } as Response)

    await expect(cancelGenerationTask('task-123')).resolves.toEqual({
      task_id: 'task-123',
      status: 'cancelled',
    })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/tasks/task-123/cancel')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer task-token')
  })
})
