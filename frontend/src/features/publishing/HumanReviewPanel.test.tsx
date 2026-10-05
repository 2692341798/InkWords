// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { EditorialWorkspace, HumanPublicationReview, HumanPublicationReviewInput } from '@/services/textbook'
import { HumanReviewPanel } from './HumanReviewPanel'

const legacy: HumanPublicationReview = { id: 'legacy-1', build_id: 'build-1', stage: 'rights', reviewer: '历史测试夹具', notes: '保留原始说明，不推断为通过。', automated: false, completed_at: '2026-09-09T00:00:00Z', created_at: '2026-09-09T00:00:00Z' }
const workspace = { build: { id: 'build-1', manifest_hash: 'sha256:frozen', status: 'ready_for_review' }, human_reviews: [legacy] } as EditorialWorkspace
function fillDecision(verdict = 'pass') {
  for (const [label, value] of [
    ['真人审校阶段', 'rights'], ['真人审校结论', verdict], ['人工验收者', '隔离组件测试夹具'],
    ['真人审校范围', '测试当前冻结构建的权利材料。'], ['真人审校说明与限制', '这是组件测试夹具，不构成真实审校证据。'],
    ['真人审校证据（每行一项）', 'test:isolated-evidence'],
  ]) fireEvent.change(screen.getByLabelText(label), { target: { value } })
}
function submit() {
  fireEvent.submit(screen.getByRole('button', { name: '追加真人审校记录' }).closest('form')!)
}

describe('HumanReviewPanel', () => {
  afterEach(cleanup)
  it('preserves legacy notes and appends a decision; uncertain retries never silently rebase after refresh', async () => {
    const onRecord = vi.fn().mockRejectedValueOnce(new Error('响应中断')).mockResolvedValue(undefined)
    const view = render(<HumanReviewPanel workspace={workspace} disabled={false} onRecord={onRecord} />)
    expect(screen.getByText('权利与合规审校 · 旧版说明，未记录结论 · 未评分')).toBeTruthy()
    expect(screen.getByText(legacy.notes)).toBeTruthy()
    expect(screen.getByRole('combobox', { name: '真人审校阶段' }).querySelectorAll('option')).toHaveLength(8)
    fillDecision()
    submit()
    await screen.findByRole('alert')
    const newer: HumanPublicationReview = { ...legacy, id: 'review-2', revision: 2, contract_version: 'inkwords.human-publication-review.v2', verdict: 'pass', score: 3 }
    view.rerender(<HumanReviewPanel workspace={{ ...workspace, human_reviews: [legacy, newer] }} disabled={false} onRecord={onRecord} />)
    submit()
    await waitFor(() => expect(onRecord).toHaveBeenCalledTimes(2))
    expect(onRecord.mock.calls[0][0]).toEqual(onRecord.mock.calls[1][0])
    expect(onRecord.mock.calls[0][0]).toMatchObject({ expected_revision: 1, manifest_hash: 'sha256:frozen', verdict: 'pass', score: 3, hard_failures: [] })
    expect(onRecord.mock.calls[0][0]).not.toHaveProperty('reviewer_kind')
    expect(onRecord.mock.calls[0][0]).not.toHaveProperty('completed_at')
  })
  it('rejects low scores, blank evidence and unsupported revision findings before calling the API', async () => {
    const onRecord = vi.fn().mockResolvedValue(undefined)
    render(<HumanReviewPanel workspace={workspace} disabled={false} onRecord={onRecord} />)
    fillDecision()
    fireEvent.change(screen.getByLabelText('真人审校评分（0–4）'), { target: { value: '2' } })
    submit()
    expect(onRecord).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('真人审校评分（0–4）'), { target: { value: '3' } })
    fireEvent.change(screen.getByLabelText('真人审校证据（每行一项）'), { target: { value: '  \n  ' } })
    submit()
    expect(onRecord).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('真人审校结论'), { target: { value: 'needs_revision' } })
    submit()
    expect(onRecord).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText('真人审校阻断发现（每行一项）'), { target: { value: '测试尚缺署名依据' } })
    submit()
    await waitFor(() => expect(onRecord).toHaveBeenCalledTimes(1))
    expect(onRecord.mock.calls[0][0]).toMatchObject({ verdict: 'needs_revision', hard_failures: ['测试尚缺署名依据'] })
  })
  it('disables in-flight edits and does not submit twice', async () => {
    let resolve!: () => void
    const onRecord = vi.fn<(input: HumanPublicationReviewInput) => Promise<void>>(() => new Promise<void>((done) => { resolve = done }))
    render(<HumanReviewPanel workspace={workspace} disabled={false} onRecord={onRecord} />)
    fillDecision('not_assessed')
    submit()
    const button = screen.getByRole('button', { name: '正在保存真人审校…' })
    expect((button.closest('fieldset') as HTMLFieldSetElement).disabled).toBe(true)
    fireEvent.submit(button.closest('form')!)
    expect(onRecord).toHaveBeenCalledTimes(1)
    expect(onRecord.mock.calls[0][0]).toMatchObject({ verdict: 'not_assessed', score: 0 })
    resolve()
    await screen.findByRole('button', { name: '追加真人审校记录' })
  })
  it('keeps evidence readable and the form locked after promotion', () => {
    render(<HumanReviewPanel workspace={{ ...workspace, build: { ...workspace.build, status: 'publication_candidate' } }} disabled={false} onRecord={vi.fn()} />)
    expect(screen.queryByRole('button', { name: '追加真人审校记录' })).toBeNull()
    expect(screen.getByText(legacy.notes)).toBeTruthy()
  })
})
