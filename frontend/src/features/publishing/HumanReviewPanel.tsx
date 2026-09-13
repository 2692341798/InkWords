import { useRef, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import type { EditorialWorkspace, HumanPublicationReview, HumanPublicationReviewInput, PublicationReviewStage } from '@/services/textbook'

const stages: Array<[PublicationReviewStage, string]> = [
  ['developmental', '发展性审校'], ['technical', '技术审校'], ['self_study', '自学性审校'], ['consistency', '全书一致性审校'],
  ['copy_editing', '文字审校'], ['layout', '版式审校'], ['rights', '权利与合规审校'], ['reader_trial', '读者试学'],
]
const labels = { pass: '真人审校通过', needs_revision: '需要修改', not_assessed: '尚未评估' }
const control = 'w-full rounded-md border border-border bg-background px-3 py-2'
const lines = (value: string) => value.split('\n').map((item) => item.trim()).filter(Boolean)
const revision = (review: HumanPublicationReview) => review.revision ?? 1
const decision = (review: HumanPublicationReview) => review.contract_version && review.verdict
  ? `${labels[review.verdict]} · v${revision(review)} · ${review.verdict === 'not_assessed' ? '未评分' : `${review.score}/4`}`
  : '旧版说明，未记录结论 · 未评分'

export function HumanReviewPanel({ workspace, disabled, onRecord }: {
  workspace: EditorialWorkspace
  disabled: boolean
  onRecord: (input: HumanPublicationReviewInput) => Promise<void>
}) {
  const [stage, setStage] = useState<PublicationReviewStage>('developmental')
  const [verdict, setVerdict] = useState<HumanPublicationReviewInput['verdict']>('not_assessed')
  const [score, setScore] = useState(3)
  const [reviewer, setReviewer] = useState('')
  const [scope, setScope] = useState('')
  const [notes, setNotes] = useState('')
  const [evidence, setEvidence] = useState('')
  const [failures, setFailures] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const request = useRef<{ fingerprint: string; input: HumanPublicationReviewInput } | null>(null)
  const reviews = workspace.human_reviews
  const locked = workspace.build.status !== 'ready_for_review'
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (saving || disabled || locked) return
    const input = {
      manifest_hash: workspace.build.manifest_hash,
      expected_revision: Math.max(0, ...reviews.filter((item) => item.stage === stage).map(revision)),
      stage, verdict, score: verdict === 'not_assessed' ? 0 : score,
      reviewer: reviewer.trim(), scope: scope.trim(), notes: notes.trim(),
      evidence_refs: lines(evidence), hard_failures: verdict === 'needs_revision' ? lines(failures) : [],
    }
    const bounded = (value: string, min: number, max: number) => [...value].length >= min && [...value].length <= max
    if (!bounded(input.reviewer, 1, 128) || !bounded(input.scope, 8, 1000) || !bounded(input.notes, 8, 4000)
      || !Number.isInteger(input.score) || input.score < 0 || input.score > 4
      || [input.evidence_refs, input.hard_failures].some((items) => items.length > 32 || items.some((item) => !bounded(item, 1, 500)))
      || (verdict === 'pass' && (score < 3 || input.evidence_refs.length === 0))
      || (verdict === 'needs_revision' && input.hard_failures.length === 0)) {
      setError('请填写完整范围和说明；通过需至少 3 分及证据，需要修改需填写阻断发现。每类引用最多 32 项，每项最多 500 字。')
      return
    }
    // Refreshing history must not silently rebase an uncertain submission.
    const fingerprint = JSON.stringify([workspace.build.id, { ...input, expected_revision: undefined }])
    if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, input: { ...input, id: crypto.randomUUID() } }
    setSaving(true)
    setError(null)
    try {
      await onRecord(request.current.input)
      request.current = null
      setNotes('')
      setFailures('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '保存失败，草稿已保留；可重试，版本冲突时请刷新核对历史后再提交。')
    } finally { setSaving(false) }
  }
  return <section className="rounded-md border border-border p-4" aria-labelledby="human-review-title">
    <h3 id="human-review-title" className="font-medium">八阶段人工审校</h3>
    <p className="mt-2 text-sm text-muted-foreground">仅登记实际真人审校。每次复审追加结论、评分和证据，保留旧版本；旧版说明不自动算通过。用户委托 AI 审阅请使用下方独立入口。</p>
    <ul className="mt-3 grid gap-2">{stages.map(([value, label]) => {
      const latest = reviews.filter((review) => review.stage === value).sort((a, b) => revision(b) - revision(a))[0]
      return <li key={value} className="rounded border border-border px-3 py-2 text-sm">{label} · {latest ? decision(latest) : '待人工完成'}</li>
    })}</ul>
    {reviews.length > 0 ? <details className="mt-3 text-sm"><summary className="cursor-pointer">真人审校历史（{reviews.length} 条）</summary><ol className="mt-2 space-y-3">{reviews.map((review) => <li key={review.id} className="rounded border border-border p-3">
      <p>{stages.find(([value]) => value === review.stage)?.[1]} · {decision(review)} · {review.reviewer}</p>
      <p>记录时间：{review.completed_at}</p>{review.scope ? <p>范围：{review.scope}</p> : null}
      <p className="whitespace-pre-wrap">{review.notes}</p>
      {review.hard_failures?.length ? <p>阻断：{review.hard_failures.join('；')}</p> : null}
      <p className="break-words">证据：{review.evidence_refs?.join('；') || '未记录证据引用'}</p>
    </li>)}</ol></details> : null}
    {!locked ? <form className="mt-4" onSubmit={(event) => void submit(event)}><fieldset disabled={disabled || saving} className="grid gap-3">
      <label>真人审校阶段<select className={control} value={stage} onChange={(event) => setStage(event.target.value as PublicationReviewStage)}>{stages.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      <label>真人审校结论<select className={control} value={verdict} onChange={(event) => setVerdict(event.target.value as typeof verdict)}>{Object.entries(labels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      <label>真人审校评分（0–4）<input className={control} type="number" required min={verdict === 'pass' ? 3 : 0} max={4} step={1} disabled={verdict === 'not_assessed'} value={verdict === 'not_assessed' ? '' : score} placeholder="未评分" onChange={(event) => setScore(Number(event.target.value))} /></label>
      <label>人工验收者<input className={control} required maxLength={128} value={reviewer} onChange={(event) => setReviewer(event.target.value)} /></label>
      <label>真人审校范围<textarea className={control} required minLength={8} maxLength={1000} value={scope} onChange={(event) => setScope(event.target.value)} /></label>
      <label>真人审校说明与限制<textarea className={`${control} min-h-28`} required minLength={8} maxLength={4000} value={notes} onChange={(event) => setNotes(event.target.value)} /></label>
      <label>真人审校证据（每行一项）<textarea className={control} required={verdict === 'pass'} value={evidence} onChange={(event) => setEvidence(event.target.value)} /></label>
      {verdict === 'needs_revision' ? <label>真人审校阻断发现（每行一项）<textarea className={control} required value={failures} onChange={(event) => setFailures(event.target.value)} /></label> : null}
      <Button type="submit">{saving ? '正在保存真人审校…' : '追加真人审校记录'}</Button>
    </fieldset>{error ? <p role="alert" className="mt-2 text-sm text-destructive">{error}</p> : null}</form> : null}
  </section>
}
