import { useRef, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import type { DelegatedPublicationReviewInput, EditorialWorkspace, PublicationReviewStage } from '@/services/textbook'

const stages: Array<[PublicationReviewStage, string]> = [
  ['developmental', '发展性审校'], ['technical', '技术审校'], ['self_study', '自学性审校'], ['consistency', '全书一致性审校'],
  ['copy_editing', '文字审校'], ['layout', '版式审校'], ['rights', '权利与合规审校'], ['reader_trial', '读者试学'],
]
const verdictLabel = { pass: '委托 AI 通过', needs_revision: '需要修改', not_assessed: '尚未评估' }
const control = 'w-full rounded-md border border-border bg-background px-3 py-2'
const lines = (value: string) => value.split('\n').map((line) => line.trim()).filter(Boolean)

export function DelegatedReviewPanel({ workspace, disabled, onRecord }: {
  workspace: EditorialWorkspace
  disabled: boolean
  onRecord: (input: DelegatedPublicationReviewInput) => Promise<void>
}) {
  const [stage, setStage] = useState<PublicationReviewStage>('developmental')
  const [verdict, setVerdict] = useState<DelegatedPublicationReviewInput['verdict']>('not_assessed')
  const [score, setScore] = useState(3)
  const [reviewer, setReviewer] = useState('')
  const [delegation, setDelegation] = useState('')
  const [scope, setScope] = useState('')
  const [notes, setNotes] = useState('')
  const [evidence, setEvidence] = useState('')
  const [failures, setFailures] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const request = useRef<{ fingerprint: string; id: string } | null>(null)
  const reviews = workspace.delegated_reviews ?? []
  const locked = workspace.build.status !== 'ready_for_review'
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (saving || disabled || locked) return
    const input = {
      manifest_hash: workspace.build.manifest_hash,
      expected_revision: Math.max(0, ...reviews.filter((item) => item.stage === stage).map((item) => item.revision)),
      stage, verdict, score: verdict === 'not_assessed' ? 0 : score,
      reviewer: reviewer.trim(), delegation_note: delegation.trim(), scope: scope.trim(), notes: notes.trim(),
      evidence_refs: lines(evidence), hard_failures: verdict === 'needs_revision' ? lines(failures) : [],
    }
    const fingerprint = JSON.stringify([workspace.build.id, input])
    if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, id: crypto.randomUUID() }
    setSaving(true)
    setError(null)
    try {
      await onRecord({ ...input, id: request.current.id })
      request.current = null
      setNotes('')
      setFailures('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '保存失败；原记录保留，可重试或刷新后核对。')
    } finally { setSaving(false) }
  }
  return <section className="mt-4 rounded-md border border-border p-4" aria-labelledby="delegated-review-title">
    <h3 id="delegated-review-title" className="font-medium">用户委托 AI 整书审阅</h3>
    <p className="mt-2 text-sm text-muted-foreground">明确记录授权、范围、证据与评分。AI 审阅不等于真人同行审校、读者掌握或出版社认证；需要修改的结论继续阻断晋级。记录绑定当前冻结构建，复审追加版本。</p>
    <ul className="mt-3 grid gap-2 sm:grid-cols-2">{stages.map(([value, label]) => {
      const latest = reviews.filter((review) => review.stage === value).sort((a, b) => b.revision - a.revision)[0]
      return <li key={value} className="rounded border border-border p-2 text-sm">{label} · {latest ? `${verdictLabel[latest.verdict]} · v${latest.revision} · ${latest.verdict === 'not_assessed' ? '未评分' : `${latest.score}/4`}` : '未记录委托审阅'}</li>
    })}</ul>
    {reviews.length > 0 ? <details className="mt-3 text-sm"><summary className="cursor-pointer">审阅历史（{reviews.length} 条）</summary><ol className="mt-2 space-y-3">{reviews.map((review) => <li key={review.id} className="rounded border border-border p-3"><p>{stages.find(([value]) => value === review.stage)?.[1]} · v{review.revision} · {verdictLabel[review.verdict]} · 用户委托 AI / {review.reviewer}</p><p>授权：{review.delegation_note}</p><p>范围：{review.scope}</p><p className="whitespace-pre-wrap">{review.notes}</p>{review.hard_failures.length > 0 ? <p>阻断：{review.hard_failures.join('；')}</p> : null}<p className="break-words">证据：{review.evidence_refs.join('；') || '尚无证据'}</p></li>)}</ol></details> : null}
    {!locked ? <form className="mt-4 grid gap-3" onSubmit={(event) => void submit(event)}>
      <div className="grid gap-3 sm:grid-cols-3">
        <label>委托审阅阶段<select className={control} value={stage} onChange={(event) => setStage(event.target.value as PublicationReviewStage)}>{stages.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label>委托审阅结论<select className={control} value={verdict} onChange={(event) => setVerdict(event.target.value as typeof verdict)}>{Object.entries(verdictLabel).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label>评分（0–4）<input className={control} type="number" min={verdict === 'pass' ? 3 : 0} max={4} required disabled={verdict === 'not_assessed'} placeholder="未评分" value={verdict === 'not_assessed' ? '' : score} onChange={(event) => setScore(Number(event.target.value))} /></label>
      </div>
      <label>AI 审阅者<input className={control} required maxLength={128} value={reviewer} onChange={(event) => setReviewer(event.target.value)} /></label>
      <label>用户授权说明<textarea className={control} required minLength={8} maxLength={1000} value={delegation} onChange={(event) => setDelegation(event.target.value)} /></label>
      <label>实际审阅范围<textarea className={control} required minLength={8} maxLength={1000} value={scope} onChange={(event) => setScope(event.target.value)} /></label>
      <label>审阅结论与限制<textarea className={`${control} min-h-28`} required minLength={8} maxLength={4000} value={notes} onChange={(event) => setNotes(event.target.value)} /></label>
      <label>证据引用（每行一项）<textarea className={control} required={verdict === 'pass'} value={evidence} onChange={(event) => setEvidence(event.target.value)} /></label>
      {verdict === 'needs_revision' ? <label>阻断发现（每行一项）<textarea className={control} required value={failures} onChange={(event) => setFailures(event.target.value)} /></label> : null}
      {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      <Button type="submit" disabled={disabled || saving}>{saving ? '正在保存委托审阅…' : '保存委托 AI 审阅'}</Button>
    </form> : null}
  </section>
}
