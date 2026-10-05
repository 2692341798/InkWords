import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { textbookService, type ChapterWorkspace, type TextbookGenerationTaskSnapshot, type TextbookVerificationAttempt, type VerificationAttemptInput } from '@/services/textbook'

const active = (status: string) => ['pending', 'queued', 'running', 'streaming'].includes(status)
const labels: Record<TextbookGenerationTaskSnapshot['status'], string> = {
  pending: '准备验证', queued: '等待隔离执行', running: '正在隔离验证', streaming: '正在接收结果',
  succeeded: '验证任务已完成', failed: '验证任务失败', cancelled: '验证任务已取消',
}

interface Props {
  chapterId: string
  artifactId: string
  onEvidenceLoaded: (workspace: ChapterWorkspace) => void
}

export function ArtifactVerificationControls({ chapterId, artifactId, onEvidenceLoaded }: Props) {
  const [task, setTask] = useState<TextbookGenerationTaskSnapshot | null>(null)
  const [attempt, setAttempt] = useState<TextbookVerificationAttempt | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [refresh, setRefresh] = useState(0)
  const mutation = useRef(false)
  const completed = useRef('')
  const pendingRequest = useRef<VerificationAttemptInput | null>(null)

  useEffect(() => {
    let disposed = false
    let timer: ReturnType<typeof setTimeout> | undefined
    const read = async () => {
      try {
        // The server resolves the exact artifact/manifest; refresh never starts work.
        const existing = await textbookService.getArtifactVerification(chapterId, artifactId)
        const next = existing.data ? await textbookService.getTask(existing.data.id) : null
        if (next && next.task_subtype !== 'textbook_teaching_artifact_verify') throw new Error('任务类型不属于教学代码验证')
        if (disposed) return
        setTask(next)
        setAttempt(existing.data)
        setLoaded(true)
        if (next && !active(next.status)) {
          const identity = `${next.id}:${next.retry_count}:${next.status}`
          if (completed.current !== identity) {
            const workspace = await textbookService.getChapterWorkspace(chapterId)
            if (disposed) return
            onEvidenceLoaded(workspace.data)
            completed.current = identity
          }
        }
        if (next && (active(next.status) || (existing.data?.attempt_contract && !existing.data.execution_stopped))) timer = setTimeout(() => void read(), 2000)
      } catch (reason) {
        if (!disposed) setError(reason instanceof Error ? reason.message : '无法查询验证状态')
        // An observation error is not a terminal task and does not requeue it.
      }
    }
    void read()
    return () => { disposed = true; if (timer) clearTimeout(timer) }
  }, [chapterId, artifactId, refresh, onEvidenceLoaded])

  const act = async (operation: 'start' | 'retry' | 'cancel') => {
    if (mutation.current) return
    mutation.current = true
    setBusy(true)
    setError('')
    try {
      if (operation === 'start') {
        // Pin both values across an uncertain response. A new GET must not
        // silently rebase a retry onto a task another tab just created.
        pendingRequest.current ??= { request_id: crypto.randomUUID(), ...(task ? { expected_previous_task_id: task.id } : {}) }
        await textbookService.startArtifactVerification(chapterId, artifactId, pendingRequest.current)
        pendingRequest.current = null
      }
      else if (task && operation === 'retry') await textbookService.retryTask(task.id)
      else if (task) await textbookService.cancelVerificationTask(task.id)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '代码验证操作失败')
    } finally {
      mutation.current = false
      setBusy(false)
      // Resolve uncertain POST outcomes by reading, never by blindly resending.
      setRefresh(value => value + 1)
    }
  }

  return <section className="mt-3 rounded border border-border bg-muted/20 p-3" aria-label="运行教学代码验证">
    <p className="text-xs text-muted-foreground">只运行这份教学代码的冻结清单，不调用模型。关闭页面不会停止任务。</p>
    <p className="mt-2 text-sm" role="status">{task ? labels[task.status] : loaded ? '尚未创建验证任务' : '正在恢复验证状态'}</p>
    {task ? <p className="mt-1 text-xs text-muted-foreground">任务 {task.id} · {attempt?.attempt ? `第 ${attempt.attempt} 次独立验证` : `重试 ${task.retry_count} 次`}</p> : null}
    {task?.error_message ? <p role="alert" className="mt-2 text-xs text-destructive">{task.error_message}</p> : null}
    {error ? <p role="alert" className="mt-2 text-xs text-destructive">{error}。请刷新状态确认任务是否已受理。</p> : null}
    {task?.status === 'succeeded' ? <p className="mt-2 text-xs text-muted-foreground">请以下方刷新后的运行证据为准；任务结束不等于所有检查通过。</p> : null}
    {task && !active(task.status) && attempt?.attempt_contract && !attempt.execution_stopped ? <p className="mt-2 text-xs text-muted-foreground">正在等待执行器退出确认，确认前不能创建下一次验证。</p> : null}
    {task?.status === 'cancelled' ? <p className="mt-2 text-xs text-muted-foreground">取消记录已保留。重新运行将创建新任务，并保留此前的任务和证据。</p> : null}
    {attempt?.history && attempt.history.length > 1 ? <details className="mt-2 text-xs"><summary>验证历史（{attempt.history.length} 次）</summary><ol className="mt-2 space-y-1">{attempt.history.map(item => <li key={item.id}>第 {item.attempt} 次 · {labels[item.status]} · {item.id}</li>)}</ol></details> : null}
    <div className="mt-3 flex flex-wrap gap-2">
      {loaded && !task ? <Button size="sm" disabled={busy || Boolean(error)} onClick={() => void act('start')}>运行隔离验证</Button> : null}
      {task && attempt?.can_start_new ? <Button size="sm" disabled={busy || Boolean(error) || Boolean(pendingRequest.current)} onClick={() => void act('start')}>{task.status === 'succeeded' ? '重新验证' : '重新运行隔离验证'}</Button> : null}
      {task?.status === 'failed' && !attempt?.attempt_contract ? <Button size="sm" disabled={busy || Boolean(error)} onClick={() => void act('retry')}>重试同一验证</Button> : null}
      {pendingRequest.current ? <Button size="sm" disabled={busy} onClick={() => void act('start')}>重试提交本次验证</Button> : null}
      {task && active(task.status) ? <Button size="sm" variant="outline" disabled={busy} onClick={() => void act('cancel')}>取消验证</Button> : null}
      <Button size="sm" variant="outline" disabled={busy} onClick={() => { setError(''); setRefresh(value => value + 1) }}>刷新验证状态</Button>
    </div>
  </section>
}
