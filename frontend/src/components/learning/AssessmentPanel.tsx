import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { masteryAssessmentService, type AssessmentJob, type AssessmentPreview } from '@/services/masteryAssessment'
import { AssessmentFeedbackPanel } from './AssessmentFeedbackPanel'
import { AssessmentApplicationPanel } from './AssessmentApplicationPanel'
import type { NextMasteryTask } from '@/services/mastery'

const statusLabels = { running: '正在评分', succeeded: '评分已完成', failed: '评分失败', cancelled: '已取消', interrupted: '执行结果未知' }
const failureLabels: Record<string, string> = { invalid_feedback: '返回内容未通过证据检查，未保存为有效评分。', timeout: '评分超时。', execution_unknown: '上次执行结果未知，重试可能再次消耗 Token。', provider_failed: '模型调用失败。', unavailable: '评分模型尚未配置。', budget_exceeded: '评分请求超出预算。', busy: '已有评分正在执行。' }

function mergeSaved(current: AssessmentJob | null, saved: AssessmentJob | null) {
  if (!saved) return current
  if (!current) return saved
  if (current.created_at && saved.created_at && current.id !== saved.id && new Date(current.created_at) > new Date(saved.created_at)) return current
  let next = saved
  if (current.id === saved.id && (current.corrections.length > saved.corrections.length || (current.status !== 'running' && saved.status === 'running'))) next = { ...current }
  if ((current.applied_assessment?.sequence_no ?? 0) > (saved.applied_assessment?.sequence_no ?? 0)) next = { ...next, applied_assessment: current.applied_assessment, schedule: current.schedule }
  else if (next !== saved) next = { ...next, applied_assessment: saved.applied_assessment, schedule: saved.schedule }
  return next
}

export function AssessmentPanel({ objectiveID, attemptID, onApplied }: { objectiveID: string; attemptID: string; onApplied?: (schedule: NextMasteryTask) => void }) {
  const [job, setJob] = useState<AssessmentJob | null>(null)
  const [preview, setPreview] = useState<AssessmentPreview | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [refresh, setRefresh] = useState(0)
  const [prior, setPrior] = useState<AssessmentJob | null>(null)
  const requestID = useRef<string | undefined>(undefined)
  const running = job?.status === 'running'

  useEffect(() => {
    let active = true
    let timer: ReturnType<typeof setTimeout> | undefined
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 10000)
    const read = async () => {
      try {
        const saved = await masteryAssessmentService.latest(objectiveID, attemptID, controller.signal)
        if (active) {
          setJob((current) => mergeSaved(current, saved))
          setLoading(false); setError(null)
        }
        if (active && saved?.status === 'running') timer = setTimeout(() => setRefresh((value) => value + 1), 1000)
      } catch (cause) {
        if (active) { setLoading(false); setError(cause instanceof Error ? cause.message : '读取评分状态失败，请手动刷新。') }
      } finally { clearTimeout(timeout) }
    }
    void read()
    return () => { active = false; controller.abort(); clearTimeout(timeout); clearTimeout(timer) }
  }, [objectiveID, attemptID, running, refresh])

  const prepare = async () => {
    setBusy(true); setError(null)
    try { setPreview(await masteryAssessmentService.preview(objectiveID, attemptID)); requestID.current = undefined }
    catch (cause) { setError(cause instanceof Error ? cause.message : '准备评分失败') }
    finally { setBusy(false) }
  }
  const start = async () => {
    if (!preview) return
    setBusy(true); setError(null)
    try {
      requestID.current ??= crypto.randomUUID()
      const saved = await masteryAssessmentService.start(objectiveID, attemptID, { request_id: requestID.current, expected_input_hash: preview.input_hash, expected_request_hash: preview.request_hash, retry_of: job && job.status !== 'succeeded' && job.status !== 'running' ? job.id : undefined })
      setJob(saved); setPreview(null)
    } catch (cause) { setError(cause instanceof Error ? cause.message : '提交评分失败；可读取状态确认。') }
    finally { setBusy(false) }
  }
  const cancel = async () => {
    if (!job) return
    setBusy(true); setError(null)
    try { setJob(await masteryAssessmentService.cancel(objectiveID, job.id)) }
    catch (cause) { setError(cause instanceof Error ? cause.message : '取消失败') }
    finally { setBusy(false) }
  }
  const readPrior = async (id: string) => {
    setBusy(true); setError(null)
    try { setPrior(await masteryAssessmentService.read(objectiveID, id)) }
    catch (cause) { setError(cause instanceof Error ? cause.message : '读取前次评分失败') }
    finally { setBusy(false) }
  }

  return <section aria-label="模型逐项评分" className="mt-4 space-y-3 border-t border-border pt-3 text-sm">
    <p className="font-medium">模型逐项评分</p>
    <p className="text-xs text-muted-foreground">模型结果可逐项核对和纠正，明确应用后才影响复习安排。运行相关条目需要本次工件的执行证据。</p>
    {loading ? <p>正在读取评分状态…</p> : null}
    {job ? <p role="status">{statusLabels[job.status]} · {job.preview.model}</p> : null}
    {job?.error_code && failureLabels[job.error_code] ? <p>{failureLabels[job.error_code]}</p> : null}
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    <div className="flex flex-wrap gap-2">
      <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => setRefresh((value) => value + 1)}>读取最新评分</Button>
      {job?.retry_of ? <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => void readPrior(job.retry_of!)}>查看前次评分</Button> : null}
      {running ? <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => void cancel()}>取消评分</Button> : !loading && job?.status !== 'succeeded' ? <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => void prepare()}>{job ? '准备重新评分' : '准备模型评分'}</Button> : null}
    </div>
    {prior ? <aside className="space-y-2 rounded-md border border-border p-3"><p>前次评分：{statusLabels[prior.status]} · {prior.preview.model}{prior.created_at ? ` · ${new Date(prior.created_at).toLocaleString()}` : ''}</p><p>{prior.result ? `模型调用 ${prior.result.provider_calls} 次；${prior.result.usage.known ? `输入 ${prior.result.usage.input_tokens ?? 0} / 输出 ${prior.result.usage.output_tokens ?? 0} Token` : 'Token 用量未知'}` : '执行用量未知'}</p>{prior.error_code && failureLabels[prior.error_code] ? <p>{failureLabels[prior.error_code]}</p> : null}{prior.retry_of ? <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => void readPrior(prior.retry_of!)}>查看更早一次</Button> : null}</aside> : null}
    {preview && !running && job?.status !== 'succeeded' ? <div className="space-y-2 rounded-md border border-border p-3"><p>模型：{preview.provider} / {preview.model}。输入预算检查通过，输出最多 {preview.max_output_tokens.toLocaleString()} Token。</p>{preview.verification_run_id ? <p className="break-all text-xs text-muted-foreground">本次请求绑定代码运行 {preview.verification_run_id}（{preview.runtime_status}）；运行条目只能引用这条记录。</p> : <p className="text-xs text-muted-foreground">本次请求没有匹配的代码运行记录；运行相关条目必须保持未知。</p>}<Button type="button" size="sm" disabled={busy} onClick={() => void start()}>调用模型评分一次</Button></div> : null}
    {job?.result ? <p className="text-xs text-muted-foreground">本任务模型调用 {job.result.provider_calls} 次 · {job.result.usage.known ? `输入 ${job.result.usage.input_tokens ?? 0} / 输出 ${job.result.usage.output_tokens ?? 0} Token` : 'Token 用量未知'} · {job.result.latency_millis} ms</p> : job ? <p className="text-xs text-muted-foreground">调用用量尚未知。</p> : null}
    {job?.effective_feedback ? <AssessmentFeedbackPanel key={job.id} job={job} onSaved={(saved) => setJob((current) => mergeSaved(current, saved))} /> : null}
    {job?.effective_feedback ? <AssessmentApplicationPanel job={job} onSaved={(saved) => setJob((current) => mergeSaved(current, saved))} onApplied={onApplied} /> : null}
  </section>
}
