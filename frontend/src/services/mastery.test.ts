import { beforeEach, describe, expect, it, vi } from 'vitest'
import { masteryService } from './mastery'

const mockFetch = vi.fn()

describe('masteryService', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    vi.stubGlobal('fetch', mockFetch)
  })

  it('reads due work from the review-service mastery route', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ code: 200, data: { tasks: [] } }),
    } as Response)

    await masteryService.getDue()

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/mastery/due')
    expect(new Headers(init.headers).has('Authorization')).toBe(false)
  })

  it('normalizes a legacy null due queue so an empty queue cannot crash a screen', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ code: 200, data: { tasks: null } }),
    } as Response)

    await expect(masteryService.getDue()).resolves.toEqual({ tasks: [] })
  })

  it('records an append-only attempt against its objective', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ code: 200, data: { skill: 'diagnose', due_at: '2026-09-04T00:00:00Z', reason: '排错表现仍需巩固。' } }),
    } as Response)

    await masteryService.recordAttempt('objective 1', { skill: 'diagnose', correct: false, independent: false, hint_count: 1, took_millis: 2000, confidence: 2, error_kinds: ['missing_cause'], attempted_at: '2026-09-03T00:00:00Z' })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/mastery/objectives/objective%201/attempts')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toMatchObject({ skill: 'diagnose', error_kinds: ['missing_cause'] })
  })

  it('creates or reuses a frozen approved-revision objective', async () => {
    mockFetch.mockResolvedValue({ ok: true, status: 200, json: async () => ({ code: 200, data: { id: 'objective-1' } }) } as Response)

    await masteryService.createObjective({ chapter_id: 'approved-revision:chapter-1:revision-1', title: '解释路由登记', behavior: '独立解释登记过程', skills: ['explain'], rubric: ['说明因果链'], key_points: ['路由树'], prerequisites: [], evidence_refs: ['approved-revision:revision-1'] })

    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/mastery/objectives')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toMatchObject({ chapter_id: 'approved-revision:chapter-1:revision-1', skills: ['explain'] })
  })

  it('reads the frozen server basis without replacing it from the current manuscript', async () => {
    const practice = { id: 'task-transfer', mode: 'transfer', prompt: '为同一路径增加另一方法并验证隔离。' }
    const learning = { format: 'inkwords.learning-projection.v2', chapter_id: 'chapter-1', revision_id: 'revision-1', content_hash: 'sha256:source', practice_set: { version: 'inkwords.practice-set.v1', tasks: [practice] } }
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ code: 200, data: { objective: { chapter_id: 'approved-revision:chapter-1:revision-1', practice_revision_id: 'revision-1', practice_content_hash: learning.content_hash, practice_projection: learning }, attempts: [] } }) }).mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ code: 200, data: { id: 'session-1', objective_id: 'objective-1', skill: 'transfer', practice_task_id: practice.id, practice_content_hash: learning.content_hash } }) })
    await expect(masteryService.getWorkspace('objective-1', 'transfer')).resolves.toMatchObject({ practice })
    expect(mockFetch).toHaveBeenCalledTimes(2)
    expect(mockFetch.mock.calls[1][0]).toContain('/objective-1/practice-sessions')
  })

  it('keeps saved answers readable when the server practice identity is missing or mismatched', async () => {
    const attempts = [{ id: 'old-attempt', answer: '原作答' }]
    mockFetch.mockResolvedValue({ ok: true, status: 200, json: async () => ({ code: 200, data: { objective: { chapter_id: 'approved-revision:chapter-1:revision-1', practice_projection: { chapter_id: 'chapter-1', revision_id: 'revision-2' } }, attempts } }) })
    await expect(masteryService.getWorkspace('objective-1', 'explain')).resolves.toMatchObject({ attempts, practice_error: expect.stringContaining('服务端题目依据') })
    expect(mockFetch).toHaveBeenCalledOnce()
  })
})
