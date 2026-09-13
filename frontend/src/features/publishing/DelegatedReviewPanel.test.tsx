// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DelegatedPublicationReview, EditorialWorkspace } from '@/services/textbook'
import { DelegatedReviewPanel } from './DelegatedReviewPanel'

const review: DelegatedPublicationReview = { id: 'review-1', build_id: 'build-1', manifest_hash: 'sha256:frozen', contract_version: 'inkwords.delegated-publication-review.v1', reviewer_kind: 'delegated_ai', stage: 'layout', revision: 1, reviewer: 'Codex', delegation_note: '用户明确委托 AI 进行审校。', verdict: 'needs_revision', score: 2, scope: '已渲染的固定校样逐页检查。', notes: '页码和目录尚未达到送审要求。', evidence_refs: ['proof:docx-sha'], hard_failures: ['缺少页码'], completed_at: '2026-09-10T00:00:00Z' }
const workspace = { build: { id: 'build-1', manifest_hash: 'sha256:frozen', status: 'ready_for_review' }, human_reviews: [], delegated_reviews: [review] } as unknown as EditorialWorkspace

describe('DelegatedReviewPanel', () => {
  afterEach(cleanup)
  it('shows failed AI history without claiming a human approval and retries the same request identity', async () => {
    const onRecord = vi.fn().mockRejectedValueOnce(new Error('网络中断')).mockResolvedValue(undefined)
    render(<DelegatedReviewPanel workspace={workspace} disabled={false} onRecord={onRecord} />)
    expect(screen.getByText('版式审校 · 需要修改 · v1 · 2/4')).toBeTruthy()
    expect(screen.getByText(/用户委托 AI \/ Codex/)).toBeTruthy()
    fireEvent.change(screen.getByLabelText('委托审阅阶段'), { target: { value: 'layout' } })
    fireEvent.change(screen.getByLabelText('委托审阅结论'), { target: { value: 'needs_revision' } })
    for (const [label, value] of [['AI 审阅者', 'Codex'], ['用户授权说明', review.delegation_note], ['实际审阅范围', review.scope], ['审阅结论与限制', review.notes], ['证据引用（每行一项）', 'proof:docx-sha'], ['阻断发现（每行一项）', '缺少页码']]) {
      fireEvent.change(screen.getByLabelText(label), { target: { value } })
    }
    fireEvent.click(screen.getByRole('button', { name: '保存委托 AI 审阅' }))
    await screen.findByRole('alert')
    fireEvent.click(screen.getByRole('button', { name: '保存委托 AI 审阅' }))
    await waitFor(() => expect(onRecord).toHaveBeenCalledTimes(2))
    expect(onRecord.mock.calls[0][0]).toEqual(onRecord.mock.calls[1][0])
    expect(onRecord.mock.calls[0][0]).toMatchObject({ expected_revision: 1, manifest_hash: 'sha256:frozen', verdict: 'needs_revision', hard_failures: ['缺少页码'] })
    expect(onRecord.mock.calls[0][0]).not.toHaveProperty('reviewer_kind')
  })
  it('locks the append form after publication promotion and preserves review history', () => {
    render(<DelegatedReviewPanel workspace={{ ...workspace, build: { ...workspace.build, status: 'publication_candidate' } }} disabled={false} onRecord={vi.fn()} />)
    expect(screen.queryByRole('button', { name: '保存委托 AI 审阅' })).toBeNull()
    expect(screen.getByText('审阅历史（1 条）')).toBeTruthy()
  })
  it('does not display the unassessed wire placeholder as an actual zero score', () => {
    render(<DelegatedReviewPanel workspace={{ ...workspace, delegated_reviews: [{ ...review, stage: 'reader_trial', verdict: 'not_assessed', score: 0, hard_failures: [] }] }} disabled={false} onRecord={vi.fn()} />)
    expect(screen.getByText('读者试学 · 尚未评估 · v1 · 未评分')).toBeTruthy()
    expect((screen.getByRole('spinbutton', { name: '评分（0–4）' }) as HTMLInputElement).value).toBe('')
  })
})
