// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { GenerationTaskStatusPanel } from './GenerationTaskStatusPanel'
import { textbookService } from '@/services/textbook'

vi.mock('@/services/textbook', () => ({
  textbookService: {
    getGenerationTask: vi.fn(),
    retryGenerationTask: vi.fn(),
    correctSample: vi.fn(),
  },
}))

const getGenerationTask = vi.mocked(textbookService.getGenerationTask)
const retryGenerationTask = vi.mocked(textbookService.retryGenerationTask)

const rejectedTask = {
  id: 'task-12345678', task_type: 'generation', task_subtype: 'textbook_sample_generate', status: 'failed' as const, retry_count: 0,
  result: {
    result_version: 1, task_subtype: 'textbook_sample_generate', final_status: 'failed', candidate_persisted: false,
    provider_name: 'deepseek', model_name: 'deepseek-v4-flash',
    provider_usage_json: { known: true, input_tokens: 4064, output_tokens: 5191, provider_call_count: 1, provider_latency_ms: 31698, local_cache_hit: false },
    quality_report_json: { contract_version: 'inkwords.sample-quality.v8', passed: false, failures: ['teaching_go_files_invalid'] },
    markdown: 'rejected prose must not be rendered',
  },
}

const retryConfirmation = {
  generation_target: { provider_name: 'deepseek', model_name: 'deepseek-v4-flash' },
  input_hash: 'sha256:confirmed',
  prompt_schema_version: 'inkwords.textbook.sample.v12',
  quality_contract_version: 'inkwords.sample-quality.v9',
  estimated_input_tokens: 10191,
  allowed_input_tokens: 20000,
  reserved_output_tokens: 12000,
  estimated_cost_known: false as const,
  requires_confirmation: true as const,
}

describe('GenerationTaskStatusPanel', () => {
  it('explains correction retries without presenting another model call', async () => {
    getGenerationTask.mockResolvedValue({ ...rejectedTask, result: {}, execution_mode: 'automated_local_correction', retry_confirmation: { ...retryConfirmation, execution_mode: 'automated_local_correction' } })
    render(<GenerationTaskStatusPanel taskID="correction-task" />)
    fireEvent.click(await screen.findByRole('button', { name: '查看重试影响' }))
    expect(screen.getByText('确认重新复核同一局部修订')).toBeTruthy()
    expect(screen.queryByText(/确认再次调用/)).toBeNull()
  })
  afterEach(cleanup)
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('shows safe rejected-generation diagnostics and usage without rendering rejected prose or triggering another call', async () => {
    getGenerationTask.mockResolvedValue(rejectedTask)
    render(<GenerationTaskStatusPanel taskID="task-12345678" />)
    const diagnostic = await screen.findByRole('region', { name: '失败生成诊断' })
    expect(diagnostic.textContent).toContain('deepseek / deepseek-v4-flash')
    expect(diagnostic.textContent).toContain('inkwords.sample-quality.v8')
    expect(diagnostic.textContent).toContain('teaching_go_files_invalid')
    expect(diagnostic.textContent).toContain('教学实现未按要求提供两个完整的 Go 代码块')
    expect(diagnostic.textContent).toContain('未生成可审阅候选稿')
    expect(within(diagnostic).getByLabelText('生成使用量').textContent).toContain('输入 Token4064')
    expect(diagnostic.textContent).toContain('供应商调用：1 次 · 调用耗时：31698 ms')
    expect(screen.queryByText('rejected prose must not be rendered')).toBeNull()
    expect(retryGenerationTask).not.toHaveBeenCalled()
  })

  it('does not show retained failed-result telemetry as a new running task result', async () => {
    getGenerationTask.mockResolvedValue({ ...rejectedTask, status: 'running' })
    render(<GenerationTaskStatusPanel taskID="task-12345678" />)
    expect(await screen.findByText('正在生成')).toBeTruthy()
    expect(screen.queryByRole('region', { name: '失败生成诊断' })).toBeNull()
  })

  it.each([null, { ...rejectedTask.result, result_version: 2 }, { ...rejectedTask.result, candidate_persisted: true }])('does not label an unsupported result as a rejected candidate: %j', async (result) => {
    getGenerationTask.mockResolvedValue({ ...rejectedTask, result })
    render(<GenerationTaskStatusPanel taskID="task-12345678" />)
    expect(await screen.findByText('执行失败')).toBeTruthy()
    expect(screen.queryByRole('region', { name: '失败生成诊断' })).toBeNull()
  })

  it('shows a retry action only for a failed task and follows the same task after retrying', async () => {
    getGenerationTask.mockResolvedValue({ id: 'task-12345678', task_type: 'generation', task_subtype: 'textbook_sample_generate', status: 'failed', error_message: 'provider timed out', retry_count: 0, retry_confirmation: retryConfirmation })
    retryGenerationTask.mockResolvedValue({ id: 'task-12345678', task_type: 'generation', task_subtype: 'sample_generation', status: 'queued', retry_count: 1 })

    render(<GenerationTaskStatusPanel taskID="task-12345678" />)

    expect(await screen.findByText('执行失败')).toBeTruthy()
    expect(screen.getByText('失败原因：provider timed out')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '查看重试影响' }))

    const confirmation = screen.getByRole('region', { name: '模型重试确认' })
    expect(confirmation.textContent).toContain('deepseek / deepseek-v4-flash')
    expect(confirmation.textContent).toContain('10,191 / 20,000 Token')
    expect(confirmation.textContent).toContain('可能产生额外费用')
    expect(retryGenerationTask).not.toHaveBeenCalled()
    fireEvent.click(within(confirmation).getByRole('button', { name: '确认重试一次' }))

    await waitFor(() => expect(retryGenerationTask).toHaveBeenCalledWith('task-12345678', 'sha256:confirmed'))
    expect(await screen.findByText('等待执行')).toBeTruthy()
    expect(screen.queryByRole('button', { name: '查看重试影响' })).toBeNull()
  })

  it('fails closed when a failed model task has no frozen retry summary', async () => {
    getGenerationTask.mockResolvedValue({ id: 'task-12345678', task_type: 'generation', task_subtype: 'textbook_sample_generate', status: 'failed', retry_count: 0 })
    render(<GenerationTaskStatusPanel taskID="task-12345678" />)
    expect(await screen.findByText(/没有可验证的冻结重试摘要/)).toBeTruthy()
    expect(screen.queryByRole('button', { name: '查看重试影响' })).toBeNull()
    expect(retryGenerationTask).not.toHaveBeenCalled()
  })

	it.each([
		['pending', '准备排队'],
		['streaming', '正在接收模型输出'],
	] as const)('renders the backend %s status explicitly', async (status, label) => {
		getGenerationTask.mockResolvedValue({ id: 'task-12345678', task_type: 'generation', task_subtype: 'textbook_sample_generate', status, retry_count: 0 })
		render(<GenerationTaskStatusPanel taskID="task-12345678" />)
		expect(await screen.findByText(label)).toBeTruthy()
	})
})
