// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { SampleCorrectionPanel } from './SampleCorrectionPanel'
import { textbookService, type TextbookGenerationTaskSnapshot } from '@/services/textbook'
vi.mock('@/services/textbook', () => ({ textbookService: { correctSample: vi.fn() } }))
afterEach(() => { cleanup(); vi.resetAllMocks() })
const task: TextbookGenerationTaskSnapshot = { id: 'source', task_type: 'generation', task_subtype: 'textbook_sample_generate', status: 'failed', retry_count: 0, result: { rejected_draft_storage: 'saved', rejected_draft_hash: 'receipt', project_id: 'project', chapter_id: 'chapter' } }
it('rejects an unrelated receipt before submission and submits only the selected edit', async () => {
  const created = vi.fn()
  vi.mocked(textbookService.correctSample).mockResolvedValue({ code: 0, data: { task_id: 'correction', status: 'queued' } })
  render(<SampleCorrectionPanel task={task} onCreated={created} />)
  const input = screen.getByLabelText('修订文件')
  fireEvent.change(input, { target: { files: [{ size: 100, text: async () => JSON.stringify({ format: 'inkwords.sample-correction.v1', original_receipt_hash: 'wrong' }) }] } })
  expect(await screen.findByRole('alert')).toBeTruthy()
  expect(textbookService.correctSample).not.toHaveBeenCalled()
  const edit = { format: 'inkwords.sample-correction.v1', original_receipt_hash: 'receipt', reason: '修正查询路由时的隐含副作用', markdown_body: '# corrected' }
  fireEvent.change(input, { target: { files: [{ size: 100, text: async () => JSON.stringify(edit) }] } })
  await screen.findByText(/修订理由/)
  fireEvent.click(screen.getByRole('button', { name: '提交局部修订候选（不调用模型）' }))
  await waitFor(() => expect(created).toHaveBeenCalledWith('correction'))
  expect(textbookService.correctSample).toHaveBeenCalledWith('project', 'chapter', 'source', edit)
})
