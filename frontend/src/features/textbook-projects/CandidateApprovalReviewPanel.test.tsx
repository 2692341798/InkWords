// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { CandidateApprovalReviewPanel } from './CandidateApprovalReviewPanel';
import { CandidateDecisionPanel } from './CandidateDecisionPanel';
import { fallbackSampleHumanReviewContract as legacy } from './sampleHumanReview';

afterEach(cleanup);

it('requires explicit delegated authority and keeps every score gate', () => {
  const onApprove = vi.fn().mockResolvedValue(undefined);
  render(<CandidateApprovalReviewPanel disabled={false} contract={{...legacy, reviewer_kinds: ['human', 'delegated_ai']}} onApprove={onApprove} />);
  fireEvent.change(screen.getByLabelText('审阅来源'), {target: {value: 'delegated_ai'}});
  for (const dimension of legacy.dimensions) fireEvent.change(screen.getByLabelText(`${dimension}委托 AI评分`), {target: {value: '3'}});
  fireEvent.change(screen.getByLabelText('委托 AI 审阅说明'), {target: {value: '逐项检查正文、来源、代码和练习后批准。'}});
  const button = screen.getByRole('button', {name: '按用户授权批准（AI 审阅）'}) as HTMLButtonElement;
  expect(button.disabled).toBe(true);
  fireEvent.change(screen.getByLabelText('授权说明'), {target: {value: '用户明确委托 Codex 审阅当前章节并决定批准。'}});
  expect(button.disabled).toBe(false);
  fireEvent.change(screen.getByLabelText(`${legacy.dimensions[0]}委托 AI评分`), {target: {value: '2'}});
  expect(button.disabled).toBe(true);
  fireEvent.change(screen.getByLabelText(`${legacy.dimensions[0]}委托 AI评分`), {target: {value: '3'}});
  fireEvent.click(button);
  expect(onApprove).toHaveBeenCalledWith(expect.objectContaining({reviewerKind: 'delegated_ai', delegationNote: '用户明确委托 Codex 审阅当前章节并决定批准。'}));
});

it('does not offer delegated approval against a legacy server', () => {
  render(<CandidateApprovalReviewPanel disabled={false} contract={legacy} onApprove={vi.fn()} />);
  expect(screen.queryByLabelText('审阅来源')).toBeNull();
});

it('shows persisted AI provenance without labeling it human review', () => {
  render(<CandidateDecisionPanel candidateReview={{id:'review',project_id:'project',chapter_id:'chapter',candidate_revision_id:'candidate',reviewer_workspace_id:'workspace',decision:'approved',reason:'逐项复核已通过。',candidate_content_hash:'hash',quality_contract_version:'quality',created_at:'2026-09-09T00:00:00Z',human_review:{contract_version:'inkwords.sample-delegated-review.v1',reviewer_kind:'delegated_ai',delegation_note:'用户明确委托。',dimension_scores:[]}}} isBusy={false} lockIsMine={false} rejectReason='' setRejectReason={vi.fn()} rejectReasonIsValid={false} rejectReasonLength={0} onReject={vi.fn()} />);
  expect(screen.getByText('该候选稿已由用户授权 AI 审阅并批准')).toBeTruthy();
  expect(screen.queryByText('该候选稿已人工审阅并批准')).toBeNull();
});
