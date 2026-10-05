import { useCallback, useEffect, useRef, useState } from 'react'
import { masteryService, type LearnerVerificationJob, type LearnerVerificationPreview } from '@/services/mastery'

type StartRequest = Parameters<typeof masteryService.startLearnerVerification>[2]
const isRunning = (job: LearnerVerificationJob | null) => job?.status === 'queued' || job?.status === 'running'

// Each read has its own deadline; a fast code-file read must not remove the
// timeout for the independent capability or persisted-job request.
function boundedRead<T>(load: (signal: AbortSignal) => Promise<T>, parent: AbortSignal, message: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const controller = new AbortController()
    const abort = () => { controller.abort(); reject(new DOMException('aborted', 'AbortError')) }
    parent.addEventListener('abort', abort, { once: true })
    const timer = setTimeout(() => { reject(new Error(message)); controller.abort() }, 10000)
    const cleanup = () => { clearTimeout(timer); parent.removeEventListener('abort', abort) }
    controller.signal.addEventListener('abort', cleanup, { once: true })
    if (parent.aborted) abort()
    else void load(controller.signal).then(resolve, reject).finally(cleanup)
  })
}

export function useLearnerVerification(objectiveID: string, attemptID: string, hash: string) {
  const [preview, setPreview] = useState<LearnerVerificationPreview | null>(null)
  const [job, setJob] = useState<LearnerVerificationJob | null>(null)
  const [ready, setReady] = useState(false)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [reload, setReload] = useState(0)
  const [pending, setPending] = useState<StartRequest | null>(null)
  const action = useRef(false)
  const epoch = useRef(0)
  const mounted = useRef(false)
  const actionController = useRef<AbortController | null>(null)
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; actionController.current?.abort() } }, [])

  const validate = useCallback((saved: LearnerVerificationJob) => {
    if (saved.objective_id !== objectiveID || saved.attempt_id !== attemptID || saved.preview.snapshot_hash !== hash) throw new Error('验证记录与本次代码快照不匹配。')
    return saved
  }, [objectiveID, attemptID, hash])

  useEffect(() => {
    const controller = new AbortController()
    const generation = epoch.current
    const current = () => !controller.signal.aborted && generation === epoch.current && !action.current
    setReady(false)
    setPreview(null)
    setPreviewError(null)
    setStatusError(null)
    void boundedRead((signal) => masteryService.learnerVerificationPreview(objectiveID, attemptID, signal), controller.signal, '隔离预检读取超时，请重新读取。').then((value) => {
      if (!value.capability?.accepted || value.capability.format !== 'inkwords.learner-verification-capability.v1' || value.capability.profile !== 'inkwords.learner-go-test-offline.v2' || value.requires_explicit_start !== true) throw new Error('代码隔离预检合同不匹配。')
      if (value.capability.available && (value.snapshot_hash !== hash || !value.input_hash)) throw new Error('隔离预检与本次代码快照不匹配。')
      if (!controller.signal.aborted) setPreview(value)
    }).catch((cause) => { if (!controller.signal.aborted) setPreviewError(cause instanceof Error ? cause.message : '读取隔离预检失败。') })
    void boundedRead((signal) => masteryService.learnerVerificationLatest(objectiveID, attemptID, signal), controller.signal, '验证状态读取超时。').then((saved) => {
      if (saved) validate(saved)
      if (current()) { setJob(saved); setReady(true); setStatusError(null); if (saved) setPending((request) => saved.request_id === request?.request_id ? null : request) }
    }).catch(() => { if (current()) setStatusError('无法确认最新验证状态，请重新读取；不会自动启动验证。') })
    return () => controller.abort()
  }, [objectiveID, attemptID, hash, reload, validate])

  useEffect(() => {
    if (!job || !isRunning(job)) return
    const runID = job.id
    const controller = new AbortController()
    let timer: ReturnType<typeof setTimeout>
    const poll = async () => {
      const generation = epoch.current
      try {
        const saved = validate(await boundedRead((signal) => masteryService.readLearnerVerification(objectiveID, runID, signal), controller.signal, '验证状态读取超时。'))
        if (!controller.signal.aborted && generation === epoch.current && !action.current) { setJob(saved); setReady(true); setStatusError(null) }
      } catch {
        if (!controller.signal.aborted && generation === epoch.current && !action.current) { setReady(false); setStatusError('验证状态暂时无法更新，以下保留最后已知状态；不会自动重新执行。') }
      } finally {
        if (!controller.signal.aborted) timer = setTimeout(() => void poll(), 2000)
      }
    }
    timer = setTimeout(() => void poll(), 1000)
    return () => { controller.abort(); clearTimeout(timer) }
  }, [objectiveID, job, reload, validate])

  const start = async () => {
    if (action.current || (!pending && (!ready || !preview?.capability.available || !preview.input_hash))) return
    const request = pending ?? { request_id: crypto.randomUUID(), expected_input_hash: preview!.input_hash!, ...(job ? { retry_of: job.id } : {}) }
    action.current = true; epoch.current++; setBusy(true); setPending(request); setActionError(null)
    actionController.current = new AbortController()
    try {
      const saved = validate(await boundedRead((signal) => masteryService.startLearnerVerification(objectiveID, attemptID, request, signal), actionController.current.signal, '启动响应超时，任务状态未确认。请重新读取或重试同一启动请求。'))
      if (mounted.current) { setJob(saved); setReady(true); setPending(null); setStatusError(null) }
    } catch (cause) {
      if (mounted.current) { setReady(false); setActionError(cause instanceof Error ? cause.message : '启动验证响应未确认。') }
    } finally { action.current = false; if (mounted.current) setBusy(false) }
  }
  const cancel = async () => {
    if (!job || action.current) return
    action.current = true; epoch.current++; setBusy(true); setActionError(null)
    actionController.current = new AbortController()
    try {
      const saved = validate(await boundedRead((signal) => masteryService.cancelLearnerVerification(objectiveID, job.id, signal), actionController.current.signal, '取消响应超时，请读取实际状态；尚未确认取消。'))
      if (mounted.current) { setJob(saved); setReady(true); setStatusError(null) }
    } catch (cause) { if (mounted.current) setActionError(cause instanceof Error ? cause.message : '取消验证响应未确认，请读取实际状态。') }
    finally { action.current = false; if (mounted.current) setBusy(false) }
  }
  return { preview, job, ready, previewError, statusError, actionError, busy, pending, start, cancel, refresh: () => setReload((value) => value + 1) }
}
