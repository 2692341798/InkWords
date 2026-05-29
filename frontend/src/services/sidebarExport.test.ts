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
    const pdfBlob = new Blob(['pdf'])
    const fetchImpl = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        blob: vi.fn().mockResolvedValue(pdfBlob),
      })
      .mockResolvedValueOnce({
        ok: false,
        json: vi.fn().mockResolvedValue({ message: '导出失败' }),
      })
    const downloadBlob = vi.fn()

    const result = await exportSeriesPdfs(
      [createSeries({ id: 'series-a', title: '系列/导读' }), createSeries({ id: 'series-b', title: '系列B' })],
      {
        fetchImpl,
        getToken: () => 'token-123',
        downloadBlob,
      },
    )

    expect(downloadBlob).toHaveBeenCalledWith(pdfBlob, '系列-导读.pdf')
    expect(result).toEqual({
      succeededCount: 1,
      failed: [{ id: 'series-b', title: '系列B', message: '导出失败' }],
    })
  })
})
