// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { masteryAssessmentService, type AssessmentJob } from '@/services/masteryAssessment'
import { AssessmentFeedbackPanel } from './AssessmentFeedbackPanel'

vi.mock('@/services/masteryAssessment', () => ({ masteryAssessmentService: { correct: vi.fn(), start: vi.fn(), apply: vi.fn() } }))
const finding = { text: '原始提示', evidence_ids: ['source-1'] }
const feedback = { criteria: [{ id: 'accuracy', score: 3, reason: '流程正确', answer_quote: '先选方法', evidence_ids: ['source-1'] }], correct_points: [], missing_points: [{ ...finding, text: '题目未要求的遗漏' }], misconceptions: [], next_hint: finding, remediation: [finding] }
const job: AssessmentJob = { id: 'job-1', objective_id: 'objective-1', attempt_id: 'attempt-1', status: 'succeeded', preview: { input_hash: 'sha256:input', request_hash: 'sha256:request', provider: 'fixture', model: 'fixture', request_bytes: 2000, input_byte_limit: 32000, max_output_tokens: 3000 }, input: { answer: '先选方法', rubric: [{ id: 'accuracy', description: '说明方法与路径查找', requires_runtime: false }], evidence: [{ id: 'source-1', kind: 'source', excerpt: '先按方法再按路径查找。' }] }, result: { origin: 'automated', provider_calls: 1, latency_millis: 1, usage: { known: false }, feedback }, corrections: [], effective_feedback: feedback, effective_hash: 'sha256:original' }
beforeEach(() => vi.clearAllMocks())
afterEach(cleanup)

it('preserves the frozen edit and retry ID across a failed save and a refreshed view', async () => {
  vi.mocked(masteryAssessmentService.correct).mockRejectedValue(new Error('评分版本已变化'))
  const onSaved = vi.fn()
  const view = render(<AssessmentFeedbackPanel job={job} onSaved={onSaved} />)
  fireEvent.click(screen.getByRole('button', { name: '纠正反馈与提示' }))
  expect((screen.getByRole('button', { name: '纠正此项' }) as HTMLButtonElement).disabled).toBe(true)
  fireEvent.click(screen.getByRole('button', { name: '移除遗漏部分 1' }))
  fireEvent.change(screen.getByLabelText('更正下一步提示'), { target: { value: '方法改变后会选哪棵树？' } })
  fireEvent.change(screen.getByLabelText('反馈更正说明'), { target: { value: '删除题目未要求的遗漏' } })
  fireEvent.click(screen.getByRole('button', { name: '保存反馈纠正' }))
  await screen.findByText('评分版本已变化')
  view.rerender(<AssessmentFeedbackPanel job={{ ...job, effective_hash: 'sha256:newer' }} onSaved={onSaved} />)
  fireEvent.click(screen.getByRole('button', { name: '保存反馈纠正' }))
  await screen.findByText('评分版本已变化')
  const calls = vi.mocked(masteryAssessmentService.correct).mock.calls
  expect(calls).toHaveLength(2)
  expect(calls[0][2]).toEqual(calls[1][2])
  expect(calls[0][2]).toMatchObject({ previous_hash: 'sha256:original', changes: [], findings: { missing_points: [], next_hint: { text: '方法改变后会选哪棵树？', evidence_ids: ['source-1'] } } })
  expect((screen.getByLabelText('反馈更正说明') as HTMLTextAreaElement).value).toBe('删除题目未要求的遗漏')
  expect(feedback.missing_points[0].text).toBe('题目未要求的遗漏')
  expect(feedback.next_hint.text).toBe('原始提示')
  expect(onSaved).not.toHaveBeenCalled()
  expect(masteryAssessmentService.start).not.toHaveBeenCalled()
  expect(masteryAssessmentService.apply).not.toHaveBeenCalled()
})

it('requires each new finding to cite frozen evidence and retains the original advice in history', async () => {
  const onSaved = vi.fn()
  vi.mocked(masteryAssessmentService.correct).mockResolvedValue(job)
  render(<AssessmentFeedbackPanel job={job} onSaved={onSaved} />)
  expect(screen.getByText('原始模型反馈与提示')).toBeTruthy()
  fireEvent.click(screen.getByRole('button', { name: '纠正反馈与提示' }))
  fireEvent.click(screen.getByRole('button', { name: '添加答对部分' }))
  fireEvent.change(screen.getByLabelText('答对部分 1'), { target: { value: '已说明方法树' } })
  fireEvent.change(screen.getByLabelText('反馈更正说明'), { target: { value: '按冻结题目重新核对反馈' } })
  fireEvent.click(screen.getByRole('button', { name: '保存反馈纠正' }))
  expect(screen.getByRole('alert').textContent).toContain('选择冻结来源')
  expect(masteryAssessmentService.correct).not.toHaveBeenCalled()
  fireEvent.click(within(screen.getByRole('region', { name: '编辑答对部分' })).getByLabelText('引用 source-1'))
  fireEvent.click(screen.getByRole('button', { name: '保存反馈纠正' }))
  await screen.findByRole('button', { name: '纠正反馈与提示' })
  expect(onSaved).toHaveBeenCalledWith(job)
  expect(masteryAssessmentService.correct).toHaveBeenCalledWith('objective-1', 'job-1', expect.objectContaining({ changes: [], findings: expect.objectContaining({ correct_points: [{ text: '已说明方法树', evidence_ids: ['source-1'] }] }) }))
})
