import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { BlogNode } from '@/store/blogStore'
import { exportSeriesPdfs, syncSeriesToObsidian } from './sidebarExport'

function createSeries(overrides: Partial<BlogNode> = {}): BlogNode {
  return {
    id: 'series-1',
    title: '系列/导读',
    content: '',
    source_type: 'file',
    status: 1,
    chapter_sort: 0,
    parent_id: null,
    created_at: '2026-05-25T00:00:00Z',
    updated_at: '2026-05-25T00:00:00Z',
    children: [],
    ...overrides,
  }
}

describe('sidebarExport service', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('posts each selected series root to the Obsidian series export endpoint with auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ code: 200 }),
    })

    const count = await syncSeriesToObsidian([createSeries({ id: 'series-a' }), createSeries({ id: 'series-b' })], {
      fetchImpl,
      getToken: () => 'token-123',
    })

    expect(count).toBe(2)
    expect(fetchImpl).toHaveBeenCalledTimes(2)
    expect(fetchImpl).toHaveBeenNthCalledWith(
      1,
      '/api/v1/blogs/series-a/export/obsidian/series',
      {
        method: 'POST',
        headers: {
          Authorization: 'Bearer token-123',
        },
      },
    )
  })

  it('downloads each exported pdf with a sanitized filename and records failures', async () => {
    const createExportTask = vi
      .fn()
      .mockResolvedValueOnce({ task_id: 'task-a', stream_url: '/api/v1/tasks/task-a/stream' })
      .mockResolvedValueOnce({ task_id: 'task-b', stream_url: '/api/v1/tasks/task-b/stream' })
    const waitForTaskCompletion = vi
      .fn()
      .mockResolvedValueOnce({ id: 'task-a', status: 'succeeded' })
      .mockResolvedValueOnce({ id: 'task-b', status: 'failed', error_message: '导出失败' })
    const downloadTaskArtifact = vi.fn().mockResolvedValue(undefined)

    const result = await exportSeriesPdfs(
      [createSeries({ id: 'series-a', title: '系列/导读' }), createSeries({ id: 'series-b', title: '系列B' })],
      {
        createExportTask,
        waitForTaskCompletion,
        downloadTaskArtifact,
      },
    )

    expect(downloadTaskArtifact).toHaveBeenCalledWith('task-a', '系列-导读.pdf')
    expect(result).toEqual({
      succeededCount: 1,
      failed: [{ id: 'series-b', title: '系列B', message: '导出失败' }],
    })
  })
})
