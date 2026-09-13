// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { masteryAssessmentService, type AssessmentJob, type AssessmentPreview } from '@/services/masteryAssessment'
import { AssessmentPanel } from './AssessmentPanel'
import { AssessmentFeedbackPanel } from './AssessmentFeedbackPanel'

vi.mock('@/services/masteryAssessment', () => ({ masteryAssessmentService: { latest: vi.fn(), read: vi.fn(), preview: vi.fn(), start: vi.fn(), cancel: vi.fn(), correct: vi.fn(), apply: vi.fn() } }))
const preview: AssessmentPreview = { input_hash: 'sha256:input', request_hash: 'sha256:request', provider: 'test-provider', model: 'test-model', request_bytes: 2000, input_byte_limit: 32000, max_output_tokens: 3000 }
const criterion = { id: 'accuracy', score: 3, reason: '已说明主要步骤', answer_quote: '先选方法', evidence_ids: ['source-1'] }
const feedback = { criteria: [criterion], correct_points: [], missing_points: [], misconceptions: [], next_hint: { text: '再检查路径', evidence_ids: ['source-1'] }, remediation: [{ text: '回看原文', evidence_ids: ['source-1'] }] }
const job: AssessmentJob = { id: 'job-1', objective_id: 'objective-1', attempt_id: 'attempt-1', status: 'succeeded', preview, input: { answer: '先选方法', rubric: [{ id: 'accuracy', description: '准确解释', requires_runtime: false }], evidence: [{ id: 'source-1', kind: 'source', excerpt: '来源原文' }] }, result: { origin: 'automated', provider_calls: 1, latency_millis: 500, usage: { known: true, input_tokens: 120, output_tokens: 80 }, feedback }, corrections: [], effective_feedback: feedback, effective_hash: 'sha256:effective' }

beforeEach(() => { vi.clearAllMocks(); vi.mocked(masteryAssessmentService.latest).mockResolvedValue(null); vi.mocked(masteryAssessmentService.preview).mockResolvedValue(preview); vi.mocked(masteryAssessmentService.read).mockResolvedValue(job) })
afterEach(cleanup)

describe('assessment workflow', () => {
  it('corrects a missing-runtime explanation without inventing a source or a score', async () => {
    const unknown = { ...criterion, id: 'runtime', score: null, reason: '缺少运行记录', answer_quote: '', evidence_ids: [] }
    const codeJob: AssessmentJob = { ...job, input: { ...job.input, decision_policy: 'inkwords.criterion-coverage.v2', rubric: [{ id: 'runtime', description: '实际运行', requires_runtime: true }], learner_artifact: { format: 'inkwords.learner-artifact.v1', workspace_id: 'owner', objective_id: job.objective_id, attempt_id: job.attempt_id, session_id: 'session', revision_id: 'revision', task_id: 'task', practice_content_hash: 'sha256:practice', skill: 'reproduce', submitted_at: '2026-09-05T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'router.go', content: 'package routing' }] } }, effective_feedback: { ...feedback, criteria: [unknown] } }
    vi.mocked(masteryAssessmentService.correct).mockResolvedValue(codeJob)
    render(<AssessmentFeedbackPanel job={codeJob} onSaved={vi.fn()} />)
    expect(screen.getByText(/本次未提供运行记录，此项保持未知/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    expect((screen.getByRole('option', { name: '4/4' }) as HTMLOptionElement).disabled).toBe(true)
    expect(screen.getByText(/无需选择无关来源/)).toBeTruthy()
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '仅有静态代码，尚未取得这次作答的运行记录。' } })
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    await waitFor(() => expect(masteryAssessmentService.correct).toHaveBeenCalledWith('objective-1', 'job-1', expect.objectContaining({ changes: [expect.objectContaining({ score: null, evidence_ids: [] })] })))
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
  })
  it('lets a correction replace model citations using only the frozen evidence', async () => {
    const withSources = { ...job, input: { ...job.input, evidence: [...job.input.evidence, { id: 'source-2', kind: 'source', excerpt: '按方法选择对应路由树，再按路径匹配。' }] } }
    vi.mocked(masteryAssessmentService.correct).mockRejectedValue(new Error('保存响应中断'))
    render(<AssessmentFeedbackPanel job={withSources} onSaved={vi.fn()} />)
    expect(screen.getByText('本项引用原文')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '按冻结的方法树来源核对，原引用不充分' } })
    fireEvent.click(screen.getByLabelText('引用 source-1'))
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    expect(screen.getByRole('alert').textContent).toContain('至少选择一项本次冻结的依据')
    expect(masteryAssessmentService.correct).not.toHaveBeenCalled()
    fireEvent.click(screen.getByLabelText('引用 source-2'))
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    await screen.findByText('保存响应中断')
    expect(masteryAssessmentService.correct).toHaveBeenCalledWith('objective-1', 'job-1', expect.objectContaining({ previous_hash: job.effective_hash, changes: [expect.objectContaining({ evidence_ids: ['source-2'] })] }))
    expect((screen.getByLabelText('引用 source-2') as HTMLInputElement).checked).toBe(true)
    expect(criterion.evidence_ids).toEqual(['source-1'])
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
  })

  it('requires the correction to actually cite the runtime record for a numeric runtime score', async () => {
    const runtimeFeedback = { ...feedback, criteria: [{ ...criterion, id: 'runtime', evidence_ids: ['run-1'] }] }
    const runtimeJob = { ...job, input: { ...job.input, rubric: [{ id: 'runtime', description: '运行结果', requires_runtime: true }], evidence: [...job.input.evidence, { id: 'run-1', kind: 'runtime', excerpt: '固定作答运行失败' }] }, effective_feedback: runtimeFeedback }
    render(<AssessmentFeedbackPanel job={runtimeJob} onSaved={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '核对运行证据' } })
    fireEvent.click(screen.getByLabelText('引用 source-1'))
    fireEvent.click(screen.getByLabelText('引用 run-1'))
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    expect(screen.getByRole('alert').textContent).toContain('请选择本次运行证据，或将分数改为未知')
    expect(masteryAssessmentService.correct).not.toHaveBeenCalled()
  })
  it('displays exact code paths and preserves them through explicit corrections', async () => {
    const codeFeedback = { ...feedback, criteria: [{ ...criterion, answer_path: 'router.go', answer_quote: 'return true' }] }
    const codeJob: AssessmentJob = { ...job, input: { ...job.input, learner_artifact: { format: 'inkwords.learner-artifact.v1', workspace_id: 'owner', objective_id: job.objective_id, attempt_id: job.attempt_id, session_id: 'session', revision_id: 'revision', task_id: 'task', practice_content_hash: 'sha256:practice', skill: 'reproduce', submitted_at: '2026-09-05T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'router.go', content: 'func Match() bool { return true }' }, { path: 'router_test.go', content: '// 测试由学习者编写，尚未执行' }] } }, effective_feedback: codeFeedback }
    vi.mocked(masteryAssessmentService.correct).mockRejectedValue(new Error('请核对原文件'))
    render(<AssessmentFeedbackPanel job={codeJob} onSaved={vi.fn()} />)
    expect(screen.getByText('代码原文（router.go）：return true')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    expect((screen.getByLabelText('引文位置') as HTMLSelectElement).value).toBe('router.go')
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '按源文件核对' } })
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    await screen.findByText('请核对原文件')
    expect(masteryAssessmentService.correct).toHaveBeenCalledWith('objective-1', 'job-1', expect.objectContaining({ changes: [expect.objectContaining({ answer_path: 'router.go', answer_quote: 'return true' })] }))
    fireEvent.change(screen.getByLabelText('引文位置'), { target: { value: 'router_test.go' } })
    expect((screen.getByLabelText('作答原文依据') as HTMLTextAreaElement).value).toBe('')
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
  })
  it('applies only explicitly, retries the same decision, and rejects an older read view', async () => {
    let resolveRead!: (value: AssessmentJob) => void
    vi.mocked(masteryAssessmentService.latest).mockResolvedValueOnce(job).mockImplementationOnce(() => new Promise((resolve) => { resolveRead = resolve }))
    const saved: AssessmentJob = { ...job, applied_assessment: { id: 'application-1', job_id: job.id, feedback_hash: job.effective_hash!, sequence_no: 1, algorithm_version: 'fsrs-v4-assessment-v1', applied_at: '2026-09-05T12:00:00Z', feedback }, schedule: { skill: 'complete', due_at: '2026-09-06T12:00:00Z', reason: '安排下一题' } }
    vi.mocked(masteryAssessmentService.apply).mockRejectedValueOnce(new Error('应用响应中断')).mockResolvedValueOnce(saved)
    const onApplied = vi.fn()
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" onApplied={onApplied} />)
    await screen.findByRole('button', { name: '应用当前评分并更新复习安排' })
    expect(masteryAssessmentService.apply).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '读取最新评分' }))
    fireEvent.click(screen.getByRole('button', { name: '应用当前评分并更新复习安排' }))
    await screen.findByText('应用响应中断')
    fireEvent.click(screen.getByRole('button', { name: '应用当前评分并更新复习安排' }))
    await screen.findByText(/当前评分已应用/)
    const calls = vi.mocked(masteryAssessmentService.apply).mock.calls
    expect(calls[0][2]).toEqual(calls[1][2])
    expect(calls[0][2].expected_feedback_hash).toBe(job.effective_hash)
    expect(onApplied).toHaveBeenCalledWith(saved.schedule)
    await act(async () => { resolveRead(job) })
    expect(screen.getByText(/当前评分已应用/)).toBeTruthy()
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
  })

  it('shows an earlier applied snapshot after correction and keeps unknown scores explicit', async () => {
    const unknown = { ...feedback, criteria: [{ ...criterion, score: null }] }
    vi.mocked(masteryAssessmentService.latest).mockResolvedValue({ ...job, effective_feedback: unknown, applied_assessment: { id: 'application-1', job_id: job.id, sequence_no: 1, feedback_hash: 'sha256:older', applied_at: '2026-09-05T12:00:00Z', algorithm_version: 'fsrs-v4-assessment-v1', feedback } })
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" />)
    await screen.findByText(/当前评分尚未应用；复习安排仍使用上次应用的版本/)
    expect(screen.getByText('尚有未知项，将安排补证与巩固；不计为通过或答错。')).toBeTruthy()
    expect(masteryAssessmentService.apply).not.toHaveBeenCalled()
  })
  it('requires explicit model action and preserves request identity after a failed response', async () => {
    vi.mocked(masteryAssessmentService.start).mockRejectedValueOnce(new Error('响应中断')).mockResolvedValueOnce(job)
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" />)
    await screen.findByRole('button', { name: '准备模型评分' })
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '准备模型评分' }))
    await screen.findByRole('button', { name: '调用模型评分一次' })
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '调用模型评分一次' }))
    await screen.findByText('响应中断')
    fireEvent.click(screen.getByRole('button', { name: '调用模型评分一次' }))
    await screen.findByText('自动评分建议')
    const calls = vi.mocked(masteryAssessmentService.start).mock.calls
    expect(calls[0][2].request_id).toBe(calls[1][2].request_id)
    expect(calls[0][2]).toMatchObject({ expected_input_hash: preview.input_hash, expected_request_hash: preview.request_hash })
    expect(screen.getByText('准确解释：3/4')).toBeTruthy()
  })

  it('shows the exact runtime record included in a prepared model request', async () => {
    vi.mocked(masteryAssessmentService.preview).mockResolvedValue({ ...preview, verification_run_id: 'run-verified-1', runtime_status: 'failed' })
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" />)
    fireEvent.click(await screen.findByRole('button', { name: '准备模型评分' }))
    expect(await screen.findByText(/run-verified-1（failed）/)).toBeTruthy()
    expect(masteryAssessmentService.start).not.toHaveBeenCalled()

    const codeJob: AssessmentJob = { ...job, input: { ...job.input, artifact_hash: 'sha256:code', learner_artifact: { format: 'inkwords.learner-artifact.v1', workspace_id: 'owner', objective_id: job.objective_id, attempt_id: job.attempt_id, session_id: 'session', revision_id: 'revision', task_id: 'task', practice_content_hash: 'sha256:practice', skill: 'reproduce', submitted_at: '2026-09-05T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'router.go', content: 'package main' }] }, evidence: [...job.input.evidence, { id: 'learner-runtime:run-verified-1', kind: 'runtime', excerpt: '{"status":"failed"}', verification_run_id: 'run-verified-1', artifact_hash: 'sha256:code' }] } }
    cleanup()
    render(<AssessmentFeedbackPanel job={codeJob} onSaved={vi.fn()} />)
    expect(screen.getByText(/绑定同一快照的受控运行记录/)).toBeTruthy()
    expect(screen.getByText(/学习者自写测试不等于独立验收/)).toBeTruthy()
  })

  it('keeps a stale correction draft and never changes the original feedback', async () => {
    vi.mocked(masteryAssessmentService.correct).mockRejectedValue(new Error('评分版本已变化'))
    render(<AssessmentFeedbackPanel job={job} onSaved={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    fireEvent.change(screen.getByLabelText('更正分数'), { target: { value: '4' } })
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '原作答已经说明边界' } })
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    await screen.findByText('评分版本已变化')
    expect((screen.getByLabelText('更正说明') as HTMLTextAreaElement).value).toBe('原作答已经说明边界')
    expect(criterion.score).toBe(3)
    expect(masteryAssessmentService.correct).toHaveBeenCalledWith('objective-1', 'job-1', expect.objectContaining({ previous_hash: job.effective_hash, changes: [expect.objectContaining({ score: 4, answer_quote: '先选方法' })] }))
  })

  it('restores interrupted work without automatically retrying a provider', async () => {
    const interrupted: AssessmentJob = { ...job, status: 'interrupted', error_code: 'execution_unknown', result: undefined, effective_feedback: undefined }
    vi.mocked(masteryAssessmentService.latest).mockResolvedValue(interrupted)
    vi.mocked(masteryAssessmentService.read).mockResolvedValue(interrupted)
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" />)
    await screen.findByText('上次执行结果未知，重试可能再次消耗 Token。')
    await waitFor(() => expect(masteryAssessmentService.start).not.toHaveBeenCalled())
    expect(screen.getByRole('button', { name: '准备重新评分' })).toBeTruthy()
  })

  it('does not let a delayed read overwrite a newly saved correction', async () => {
    let resolveRead!: (value: AssessmentJob) => void
    vi.mocked(masteryAssessmentService.latest).mockResolvedValueOnce(job).mockImplementationOnce(() => new Promise((resolve) => { resolveRead = resolve }))
    const corrected: AssessmentJob = { ...job, effective_hash: 'sha256:new', effective_feedback: { ...feedback, criteria: [{ ...criterion, score: 4 }] }, corrections: [{ id: 'correction-1', previous_hash: job.effective_hash!, reason: '已说明边界', changes: [{ ...criterion, score: 4 }], reviewer_id: 'local', corrected_at: '2026-09-05T00:00:00Z' }] }
    vi.mocked(masteryAssessmentService.correct).mockResolvedValue(corrected)
    render(<AssessmentPanel objectiveID="objective-1" attemptID="attempt-1" />)
    await screen.findByText('准确解释：3/4')
    fireEvent.click(screen.getByRole('button', { name: '读取最新评分' }))
    fireEvent.click(screen.getByRole('button', { name: '纠正此项' }))
    fireEvent.change(screen.getByLabelText('更正说明'), { target: { value: '已说明边界' } })
    fireEvent.click(screen.getByRole('button', { name: '保存纠正' }))
    await screen.findByText('准确解释：4/4')
    await act(async () => { resolveRead(job) })
    expect(screen.getByText('准确解释：4/4')).toBeTruthy()
  })
})
