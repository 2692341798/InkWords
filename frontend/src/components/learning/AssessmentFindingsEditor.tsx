import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { masteryAssessmentService, type AssessmentFinding, type AssessmentFindingCorrection, type AssessmentJob } from '@/services/masteryAssessment'
import { CorrectionEvidence } from './AssessmentEvidence'

const findingGroups = [['答对部分', 'correct_points'], ['遗漏部分', 'missing_points'], ['需要纠正的理解', 'misconceptions'], ['补救材料', 'remediation']] as const
type Group = typeof findingGroups[number][1]
interface Draft { id: string; previousHash: string; reason: string; findings: AssessmentFindingCorrection }

export function AssessmentFindingsEditor({ job, disabled, onSaved, onEditingChange }: { job: AssessmentJob; disabled: boolean; onSaved: (job: AssessmentJob) => void; onEditingChange: (editing: boolean) => void }) {
  const [draft, setDraft] = useState<Draft | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const start = () => {
    if (!job.effective_feedback || !job.effective_hash) return
    const { correct_points, missing_points, misconceptions, next_hint, remediation } = job.effective_feedback
    setDraft({ id: crypto.randomUUID(), previousHash: job.effective_hash, reason: '', findings: structuredClone({ correct_points, missing_points, misconceptions, next_hint, remediation }) })
    setError(null); onEditingChange(true)
  }
  const update = (group: Group, index: number, value: AssessmentFinding) => {
    if (!draft) return
    setDraft({ ...draft, findings: { ...draft.findings, [group]: draft.findings[group].map((item, i) => i === index ? value : item) } })
  }
  const close = () => { setDraft(null); setError(null); onEditingChange(false) }
  const save = async () => {
    if (!draft) return
    if (!draft.reason.trim()) { setError('请说明修改这些反馈的依据。'); return }
    const items = [...draft.findings.correct_points, ...draft.findings.missing_points, ...draft.findings.misconceptions, ...draft.findings.remediation, draft.findings.next_hint]
    if (!draft.findings.remediation.length || items.some((item) => !item.text.trim() || item.text.length > 2000 || !item.evidence_ids.length)) { setError('每条反馈须填写内容并选择冻结来源；补救材料至少保留一条。'); return }
    setBusy(true); setError(null)
    try {
      const saved = await masteryAssessmentService.correct(job.objective_id, job.id, { id: draft.id, previous_hash: draft.previousHash, reason: draft.reason, changes: [], findings: draft.findings })
      onSaved(saved); close()
    } catch (cause) { setError(cause instanceof Error ? cause.message : '保存失败，反馈草稿已保留。') }
    finally { setBusy(false) }
  }
  if (!draft) return <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={start}>纠正反馈与提示</Button>
  return <fieldset disabled={busy || disabled} className="space-y-3 rounded-md border border-border p-3">
    <legend>纠正反馈与提示</legend>
    <p className="text-xs text-muted-foreground">对照本题要求核对遗漏与误解，可修改或移除不成立的反馈。保存会追加修改记录，原始模型反馈仍保留；复习安排需另行明确应用。</p>
    <details><summary>核对本题评分要求</summary><ul className="list-disc pl-5">{job.input.rubric.map((item) => <li key={item.id}>{item.description}</li>)}</ul></details>
    {findingGroups.map(([label, group]) => <section key={group} aria-label={`编辑${label}`} className="space-y-2">
      <h4 className="font-medium">{label}</h4>
      {draft.findings[group].map((item, index) => <div key={index} className="space-y-2 border-l-2 border-border pl-3">
        <label className="grid gap-1">{label} {index + 1}<textarea aria-label={`${label} ${index + 1}`} maxLength={2000} rows={2} value={item.text} onChange={(event) => update(group, index, { ...item, text: event.target.value })} className="rounded-md border border-border bg-background p-2" /></label>
        <CorrectionEvidence evidence={job.input.evidence} selected={item.evidence_ids} onChange={(evidence_ids) => update(group, index, { ...item, evidence_ids })} />
        <Button type="button" variant="ghost" size="sm" aria-label={`移除${label} ${index + 1}`} onClick={() => setDraft({ ...draft, findings: { ...draft.findings, [group]: draft.findings[group].filter((_, i) => i !== index) } })}>移除此条</Button>
      </div>)}
      <Button type="button" variant="ghost" size="sm" disabled={draft.findings[group].length >= 20} onClick={() => setDraft({ ...draft, findings: { ...draft.findings, [group]: [...draft.findings[group], { text: '', evidence_ids: [] }] } })}>添加{label}</Button>
    </section>)}
    <label className="grid gap-1">下一步提示<textarea aria-label="更正下一步提示" maxLength={2000} rows={2} value={draft.findings.next_hint.text} onChange={(event) => setDraft({ ...draft, findings: { ...draft.findings, next_hint: { ...draft.findings.next_hint, text: event.target.value } } })} className="rounded-md border border-border bg-background p-2" /></label>
    <CorrectionEvidence evidence={job.input.evidence} selected={draft.findings.next_hint.evidence_ids} onChange={(evidence_ids) => setDraft({ ...draft, findings: { ...draft.findings, next_hint: { ...draft.findings.next_hint, evidence_ids } } })} />
    <label className="grid gap-1">反馈更正说明<textarea aria-label="反馈更正说明" maxLength={2000} rows={2} value={draft.reason} onChange={(event) => setDraft({ ...draft, reason: event.target.value })} className="rounded-md border border-border bg-background p-2" /></label>
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    <div className="flex gap-2"><Button type="button" size="sm" onClick={() => void save()}>保存反馈纠正</Button><Button type="button" variant="ghost" size="sm" onClick={close}>取消反馈编辑</Button></div>
  </fieldset>
}

export function AssessmentFindings({ findings }: { findings: AssessmentFindingCorrection }) {
  return <div className="space-y-2">
    {findingGroups.map(([label, group]) => <div key={group}><h4 className="font-medium">{label}</h4>{findings[group].length ? <ul className="list-disc pl-5">{findings[group].map((item, index) => <li key={index}>{item.text}<span className="ml-1 break-all text-xs text-muted-foreground">〔{item.evidence_ids.join('、')}〕</span></li>)}</ul> : <p className="text-muted-foreground">本次未列出。</p>}</div>)}
    <p>下一步提示：{findings.next_hint.text}<span className="ml-1 break-all text-xs text-muted-foreground">〔{findings.next_hint.evidence_ids.join('、')}〕</span></p>
  </div>
}
