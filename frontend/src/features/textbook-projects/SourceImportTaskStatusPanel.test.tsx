// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SourceImportTaskStatusPanel } from './SourceImportTaskStatusPanel'
import { textbookService } from '@/services/textbook'

vi.mock('@/services/textbook', () => ({
  textbookService: { getTask: vi.fn(), retryTask: vi.fn() },
}))

const getTask = vi.mocked(textbookService.getTask)
const retryTask = vi.mocked(textbookService.retryTask)

describe('SourceImportTaskStatusPanel', () => {
  beforeEach(() => vi.resetAllMocks())
  afterEach(cleanup)

  it('explains that retrying official sources fetches live bytes again', async () => {
    getTask.mockResolvedValue({ id: 'official-task', task_type: 'parse', task_subtype: 'textbook_official_web_import', status: 'failed', error_message: '预算耗尽', retry_count: 0 })
    render(<SourceImportTaskStatusPanel taskID="official-task" onSucceeded={vi.fn()} />)
    expect(await screen.findByRole('button', { name: '按原范围重新抓取' })).toBeTruthy()
    expect(screen.getByText(/官网内容可能已发生变化/)).toBeTruthy()
    expect(screen.queryByText(/可重试同一冻结文件/)).toBeNull()
  })

  it('refreshes the source library once after a successful immutable import', async () => {
    const onSucceeded = vi.fn()
    getTask.mockResolvedValue({ id: 'task-12345678', task_type: 'parse', task_subtype: 'textbook_source_import', status: 'succeeded', retry_count: 0 })

    render(<SourceImportTaskStatusPanel taskID="task-12345678" onSucceeded={onSucceeded} />)

    expect(await screen.findByText('资料已入库')).toBeTruthy()
    await waitFor(() => expect(onSucceeded).toHaveBeenCalledOnce())
    expect(screen.getByText('资料导入任务')).toBeTruthy()
  })

  it('requeues a failed import without replacing its frozen source input', async () => {
    getTask.mockResolvedValue({ id: 'task-12345678', task_type: 'parse', task_subtype: 'textbook_source_import', status: 'failed', error_message: '发布任务消息失败', retry_count: 0 })
    retryTask.mockResolvedValue({ id: 'task-12345678', task_type: 'parse', task_subtype: 'textbook_source_import', status: 'queued', retry_count: 1 })

    render(<SourceImportTaskStatusPanel taskID="task-12345678" onSucceeded={vi.fn()} />)

    fireEvent.click(await screen.findByRole('button', { name: '重试冻结导入' }))
    await waitFor(() => expect(retryTask).toHaveBeenCalledWith('task-12345678'))
    expect(await screen.findByText('等待解析')).toBeTruthy()
  })

	it.each([
		['pending', '准备解析'],
		['streaming', '正在接收解析结果'],
	] as const)('renders the backend %s status explicitly', async (status, label) => {
		getTask.mockResolvedValue({ id: 'task-12345678', task_type: 'parse', task_subtype: 'textbook_source_import', status, retry_count: 0 })
		render(<SourceImportTaskStatusPanel taskID="task-12345678" onSucceeded={vi.fn()} />)
		expect(await screen.findByText(label)).toBeTruthy()
	})
})
