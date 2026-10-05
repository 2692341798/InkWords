import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { masteryAssessmentService, type AssessmentApplyInput, type AssessmentFeedback, type AssessmentJob } from '@/services/masteryAssessment'
import type { NextMasteryTask } from '@/services/mastery'

function outcome(feedback: AssessmentFeedback) {
  if (feedback.criteria.some((item) => item.score !== null && item.score < 3)) return '尚有未达标项，将按未通过重新安排练习。'
  if (feedback.criteria.some((item) => item.score === null)) return '尚有未知项，将安排补证与巩固；不计为通过或答错。'
  return '各项至少达到 3/4，将结合原提示次数、独立性、耗时和信心安排复习。'
}

const skillLabels = { explain: '复述', complete: '补全', reproduce: '复现', transfer: '迁移', diagnose: '诊断', retain: '延迟回忆' }

export function AssessmentApplicationPanel({ job, onSaved, onApplied }: { job: AssessmentJob; onSaved: (job: AssessmentJob) => void; onApplied?: (schedule: NextMasteryTask) => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const pending = useRef<AssessmentApplyInput | undefined>(undefined)
  if (!job.effective_feedback || !job.effective_hash) return null
  const applied = job.applied_assessment
  const currentApplied = applied?.job_id === job.id && applied.feedback_hash === job.effective_hash
  const apply = async () => {
    if (!job.effective_hash) return
    setBusy(true); setError(null)
    try {
      if (!pending.current || pending.current.expected_feedback_hash !== job.effective_hash || pending.current.previous_id !== applied?.id) pending.current = { id: crypto.randomUUID(), expected_feedback_hash: job.effective_hash, previous_id: applied?.id }
      const saved = await masteryAssessmentService.apply(job.objective_id, job.id, pending.current)
      onSaved(saved)
      if (saved.schedule) onApplied?.(saved.schedule)
    } catch (cause) { setError(cause instanceof Error ? cause.message : '应用评分失败；可读取最新状态确认。') }
    finally { setBusy(false) }
  }
  return <section aria-label="评分与复习安排" className="space-y-2 rounded-md border border-border p-3">
    <h4 className="font-medium">评分与复习安排</h4>
    <p>{outcome(job.effective_feedback)}</p>
    <p className="text-xs text-muted-foreground">明确应用后才更新复习安排；不会增加作答次数或改变原作答时间，也不会再次调用模型。</p>
    {applied ? <><p>{currentApplied ? '当前评分已应用' : '当前评分尚未应用；复习安排仍使用上次应用的版本'} · {new Date(applied.applied_at).toLocaleString()}</p><details><summary className="cursor-pointer">查看已应用的评分</summary><ul className="mt-2 space-y-1">{applied.feedback.criteria.map((item) => <li key={item.id}>{job.input.rubric.find((criterion) => criterion.id === item.id)?.description || item.id}：{item.score === null ? '未知' : `${item.score}/4`}</li>)}</ul></details></> : <p>此作答尚未应用模型评分，复习安排仍使用原自评。</p>}
    {job.schedule ? <p>当前安排：{skillLabels[job.schedule.skill]} · {new Date(job.schedule.due_at).toLocaleString()}。{job.schedule.reason}</p> : null}
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    {!currentApplied ? <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => void apply()}>应用当前评分并更新复习安排</Button> : null}
  </section>
}
