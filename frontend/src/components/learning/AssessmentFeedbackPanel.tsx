import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { masteryAssessmentService, type AssessmentJob, type CriterionAssessment } from '@/services/masteryAssessment'
import { AssessmentEvidence, CorrectionEvidence } from './AssessmentEvidence'
import { AssessmentFindings, AssessmentFindingsEditor } from './AssessmentFindingsEditor'
import { AssessmentDecisionDetails } from './AssessmentDecisionDetails'

interface Draft { id: string; previousHash: string; criterion: CriterionAssessment; reason: string; score: string; quote: string; path: string; evidenceIDs: string[] }

export function AssessmentFeedbackPanel({ job, onSaved }: { job: AssessmentJob; onSaved: (job: AssessmentJob) => void }) {
  const [draft, setDraft] = useState<Draft | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [findingsEditing, setFindingsEditing] = useState(false)
  const feedback = job.effective_feedback
  if (!feedback) return null
  const lacksRuntime = !!draft && !!job.input.rubric.find((item) => item.id === draft.criterion.id)?.requires_runtime && !job.input.evidence.some((item) => item.kind === 'runtime' && draft.evidenceIDs.includes(item.id))
  const mayOmitEvidence = !!draft && draft.score === '' && job.input.decision_policy === 'inkwords.criterion-coverage.v2' && !!job.input.learner_artifact && !!job.input.rubric.find((item) => item.id === draft.criterion.id)?.requires_runtime && !job.input.evidence.some((item) => item.kind === 'runtime')
  const save = async () => {
    if (!draft || !draft.reason.trim()) { setError('请说明纠正依据。'); return }
    if (!draft.evidenceIDs.length && !mayOmitEvidence) { setError('至少选择一项本次冻结的依据。'); return }
    if (lacksRuntime && draft.score !== '') { setError('请选择本次运行证据，或将分数改为未知。'); return }
    setBusy(true); setError(null)
    try {
      const saved = await masteryAssessmentService.correct(job.objective_id, job.id, { id: draft.id, previous_hash: draft.previousHash, reason: draft.reason, changes: [{ ...draft.criterion, score: draft.score === '' ? null : Number(draft.score), reason: draft.reason, answer_quote: draft.quote, answer_path: draft.path || undefined, evidence_ids: draft.evidenceIDs }] })
      onSaved(saved); setDraft(null)
    } catch (cause) { setError(cause instanceof Error ? cause.message : '保存纠正失败，草稿已保留。') }
    finally { setBusy(false) }
  }
  return <div className="space-y-3">
    <p>自动评分建议{job.corrections.length ? ` · 已追加 ${job.corrections.length} 次用户纠正` : ''}</p>
    {job.input.decision_policy === 'inkwords.criterion-coverage.v2' ? <p className="text-xs text-muted-foreground">本次评分说明及答对、遗漏、误解列表由同一组模型判断生成；请结合逐项作答原文和来源核对。原始判断保留在下方，用户纠正另行记录。</p> : null}
    {job.input.learner_artifact ? job.input.evidence.some((item) => item.kind === 'runtime') ? <p>本次包含 {job.input.learner_artifact.files.length} 个原代码文件，并绑定同一快照的受控运行记录；运行条目可引用该事实，但学习者自写测试不等于独立验收。</p> : <p>本次包含 {job.input.learner_artifact.files.length} 个原代码文件，模型仅作静态评阅；运行、测试和修复验证保持未知。</p> : null}
    {feedback.criteria.map((criterion) => <div key={criterion.id} className="rounded-md border border-border p-3">
      <p className="font-medium">{job.input.rubric.find((item) => item.id === criterion.id)?.description || criterion.id}：{criterion.score === null ? '未知' : `${criterion.score}/4`}</p><p className="mt-1">{criterion.reason}</p>
      {criterion.answer_quote ? <blockquote className="mt-2 whitespace-pre-wrap border-l-2 border-border pl-3 text-xs">{criterion.answer_path ? `代码原文（${criterion.answer_path}）：` : '作答原文：'}{criterion.answer_quote}</blockquote> : null}
      <p className="mt-1 break-all text-xs text-muted-foreground">依据：{criterion.evidence_ids.length ? criterion.evidence_ids.join('、') : '本次未提供运行记录，此项保持未知。'}</p>
      <AssessmentEvidence evidence={job.input.evidence} ids={criterion.evidence_ids} />
      <Button type="button" variant="ghost" size="sm" disabled={busy || findingsEditing} onClick={() => { setDraft({ id: crypto.randomUUID(), previousHash: job.effective_hash!, criterion, score: criterion.score === null ? '' : String(criterion.score), reason: '', quote: criterion.answer_quote, path: criterion.answer_path || '', evidenceIDs: [...criterion.evidence_ids] }); setError(null) }}>纠正此项</Button>
    </div>)}
    {draft ? <fieldset disabled={busy} className="space-y-2 rounded-md border border-border p-3"><legend>纠正 {draft.criterion.id}</legend>
      <p>本项要求：{job.input.rubric.find((item) => item.id === draft.criterion.id)?.description || draft.criterion.id}</p>
      <label className="grid gap-1">更正分数<select aria-label="更正分数" value={draft.score} onChange={(event) => setDraft({ ...draft, score: event.target.value })} className="rounded-md border border-border bg-background p-2"><option value="">未知</option>{[0, 1, 2, 3, 4].map((score) => <option key={score} value={score} disabled={lacksRuntime}>{score}/4</option>)}</select></label>
      {mayOmitEvidence ? <p className="text-xs text-muted-foreground">本次没有运行记录，可保持未知并更正说明，无需选择无关来源。</p> : lacksRuntime ? <p className="text-xs text-muted-foreground">尚未引用本次运行证据；未选择前，此项只能保持未知。</p> : null}
      <label className="grid gap-1">更正说明<textarea aria-label="更正说明" rows={3} value={draft.reason} onChange={(event) => setDraft({ ...draft, reason: event.target.value })} className="rounded-md border border-border bg-background p-2" /></label>
      {job.input.learner_artifact ? <label className="grid gap-1">引文位置<select aria-label="引文位置" value={draft.path} onChange={(event) => setDraft({ ...draft, path: event.target.value, quote: '' })} className="rounded-md border border-border bg-background p-2"><option value="">文字作答</option>{job.input.learner_artifact.files.map((file) => <option key={file.path} value={file.path}>{file.path}</option>)}</select></label> : null}
      <label className="grid gap-1">作答原文依据<textarea aria-label="作答原文依据" rows={3} value={draft.quote} onChange={(event) => setDraft({ ...draft, quote: event.target.value })} className="rounded-md border border-border bg-background p-2" /></label>
      <CorrectionEvidence evidence={job.input.evidence} selected={draft.evidenceIDs} onChange={(evidenceIDs) => setDraft({ ...draft, evidenceIDs })} />
      <p className="text-xs text-muted-foreground">正分须引用已保存文字或所选文件中的原文；纠正会追加保存。若评分版本已更新，请读取最新评分后重新选择条目。</p>
      {error ? <p role="alert" className="text-destructive">{error}</p> : null}<div className="flex gap-2"><Button type="button" size="sm" onClick={() => void save()}>保存纠正</Button><Button type="button" size="sm" variant="ghost" onClick={() => setDraft(null)}>取消编辑</Button></div>
    </fieldset> : null}
    <p className="text-xs text-muted-foreground">以下显示当前反馈{job.corrections.some((item) => item.findings) ? '，包含用户纠正；原文与修改内容可在历史中对照' : '，尚未纠正的内容来自模型'}。</p>
    <AssessmentFindings findings={feedback} />
    <AssessmentFindingsEditor job={job} disabled={busy || !!draft} onSaved={onSaved} onEditingChange={setFindingsEditing} />
    {job.result?.feedback ? <details><summary>原始模型反馈与提示</summary><AssessmentFindings findings={job.result.feedback} /></details> : null}
    {job.corrections.some((item) => item.findings) ? <details><summary>反馈与提示的纠正历史</summary>{job.corrections.filter((item) => item.findings).map((item) => <div key={item.id} className="mt-3 space-y-2 border-t border-border pt-2"><p>{new Date(item.corrected_at).toLocaleString()} · 用户纠正：{item.reason}</p><AssessmentFindings findings={item.findings!} /></div>)}</details> : null}
    <details><summary>原始自动评分与纠正记录</summary><ul className="mt-2 list-disc pl-5">{job.result?.feedback?.criteria.map((item) => <li key={item.id}>{job.input.rubric.find((criterion) => criterion.id === item.id)?.description || item.id}：{item.score === null ? '未知' : `${item.score}/4`} · {item.reason}</li>)}</ul><ol className="mt-3 space-y-2">{job.corrections.map((correction) => <li key={correction.id}>{new Date(correction.corrected_at).toLocaleString()} · 用户纠正：{correction.reason}<ul className="list-disc pl-5">{correction.changes.map((item) => <li key={item.id}>{job.input.rubric.find((criterion) => criterion.id === item.id)?.description || item.id}：{item.score === null ? '未知' : `${item.score}/4`} · {item.reason}</li>)}</ul></li>)}</ol></details>
    <AssessmentDecisionDetails decisions={job.result?.decisions} reference={job.input.task_reference} />
    <details><summary>本次引用原文</summary>{job.input.evidence.map((item) => <div key={item.id} className="mt-2"><p className="break-all text-xs">{item.id}</p><pre className="mt-1 whitespace-pre-wrap break-words text-xs">{item.excerpt}</pre></div>)}</details>
    {job.input.learner_artifact ? <details><summary>本次待评文件</summary>{job.input.learner_artifact.files.map((file) => <div key={file.path} className="mt-2"><p className="break-all text-xs">{file.path}</p><pre className="mt-1 whitespace-pre-wrap break-words text-xs">{file.content}</pre></div>)}</details> : null}
  </div>
}
