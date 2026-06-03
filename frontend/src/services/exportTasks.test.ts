import { describe, expect, it, vi } from 'vitest'

import { createExportTask } from './exportTasks'

describe('exportTasks', () => {
  it('creates an export_pdf task and returns its stream url', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      status: 202,
      json: async () => ({
        task_id: 'task-export-1',
        status: 'queued',
        stream_url: '/api/v1/tasks/task-export-1/stream',
      }),
    })

    await expect(
      createExportTask('series-1', { fetchImpl, getToken: () => 'token-123' }),
    ).resolves.toEqual({
      task_id: 'task-export-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-export-1/stream',
    })
  })
})
