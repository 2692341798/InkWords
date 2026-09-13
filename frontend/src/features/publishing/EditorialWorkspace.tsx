import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Panel, StatusPill } from '@/components/ui/workspace'
import type { DelegatedPublicationReviewInput, HumanPublicationReviewInput, EditorialWorkspace as EditorialWorkspaceData, RightsStatus, RightsWorkType } from '@/services/textbook'
import { RightsPanel } from './RightsPanel'
import { DelegatedReviewPanel } from './DelegatedReviewPanel'
import { HumanReviewPanel } from './HumanReviewPanel'
import type { RightsAmendmentInput } from '@/services/rightsAmendments'

export function EditorialWorkspace({ workspace, disabled, onAddRight, onAmendRight, onCompleteReview, onRecordDelegatedReview, onPromote }: {
  workspace: EditorialWorkspaceData
  disabled: boolean
  onAddRight: (input: { subjectRef: string; workType: RightsWorkType; rightsBasis: string; allowedUse: string; attribution: string; publicationStatus: RightsStatus }) => Promise<void>
  onCompleteReview: (input: HumanPublicationReviewInput) => Promise<void>
  onPromote: () => Promise<void>
  onRecordDelegatedReview?: (input: DelegatedPublicationReviewInput) => Promise<void>
  onAmendRight?: (input: RightsAmendmentInput) => Promise<void>
}) {
  const locked = workspace.build.status !== 'ready_for_review'
  return <Panel className="mt-4 p-5">
    <div className="flex flex-wrap items-start justify-between gap-3"><div><p className="text-xs text-muted-foreground">出版编辑工作台</p><h2 className="mt-1 text-lg font-semibold">以冻结构建为单位记录审阅证据</h2><p className="mt-2 max-w-3xl text-sm text-muted-foreground">分别记录真人审校、用户委托 AI 审阅与自动检查；“出版候选”只是 InkWords 内部预检状态，不代表出版社、ISBN 或 CIP 批准。</p></div><StatusPill tone={workspace.preflight.passed ? 'success' : 'warning'}>{workspace.preflight.passed ? '预检证据齐全' : '预检仍阻断'}</StatusPill></div>
    <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(300px,0.8fr)]">
      <RightsPanel key={`rights:${workspace.build.id}`} workspace={workspace} disabled={disabled} onAdd={onAddRight} onAmend={onAmendRight} />
      <HumanReviewPanel key={`human-review:${workspace.build.id}`} workspace={workspace} disabled={disabled} onRecord={onCompleteReview} />
    </div>
    {workspace.delegated_review_contract === 'inkwords.delegated-publication-review.v1' && onRecordDelegatedReview ? <DelegatedReviewPanel key={workspace.build.id} workspace={workspace} disabled={disabled} onRecord={onRecordDelegatedReview} /> : null}
    <section className="mt-4 rounded-md border border-border p-4"><h3 className="font-medium">自动完整性检查</h3><ul className="mt-2 space-y-1 text-sm">{workspace.automated_checks.map((check) => <li key={check.id}>{check.id} · {check.detector} · {check.status}</li>)}</ul></section>
    {workspace.preflight.blockers.length > 0 ? <div className="mt-4 rounded border border-amber-500/30 bg-amber-500/5 p-3 text-sm"><p className="flex items-center gap-1 font-medium"><ShieldAlert className="h-4 w-4" />仍不能晋级出版候选</p><ul className="mt-2 list-disc space-y-1 pl-5">{workspace.preflight.blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul></div> : null}
    <div className="mt-4 flex flex-wrap items-center gap-3"><Button disabled={disabled || locked || !workspace.preflight.passed} onClick={() => void onPromote().catch(() => undefined)}>显式标记为出版候选</Button><span className="text-xs text-muted-foreground">只有证据全部齐全时可点击；不会自动触发，也不会写入虚构的出版信息。</span></div>
  </Panel>
}
