import { useCallback, useEffect, useState } from 'react'
import { RefreshCw, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { StatusPill } from '@/components/ui/workspace'
import { textbookService, type TextbookGenerationTaskSnapshot } from '@/services/textbook'
import { GenerationFailureDetails } from './GenerationFailureDetails'
import { SampleCorrectionPanel } from './SampleCorrectionPanel'

interface GenerationTaskStatusPanelProps {
  taskID: string
}

const statusLabel: Record<TextbookGenerationTaskSnapshot['status'], string> = {
	pending: '准备排队',
  queued: '等待执行',
  running: '正在生成',
	streaming: '正在接收模型输出',
  succeeded: '已完成',
  failed: '执行失败',
  cancelled: '已取消',
}

export function GenerationTaskStatusPanel({ taskID }: GenerationTaskStatusPanelProps) {
  const [submitted, setSubmitted] = useState<{ source: string; id: string } | null>(null)
  const activeTaskID = submitted?.source === taskID ? submitted.id : taskID
  const [task, setTask] = useState<TextbookGenerationTaskSnapshot | null>(null)
  const [isWorking, setIsWorking] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showRetryConfirmation, setShowRetryConfirmation] = useState(false)

  const refresh = useCallback(async () => {
    setIsWorking(true)
    setError(null)
    try {
      setTask(await textbookService.getGenerationTask(activeTaskID))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '查询候选教材稿任务失败')
    } finally {
      setIsWorking(false)
    }
  }, [activeTaskID])

  useEffect(() => {
    setShowRetryConfirmation(false)
    void refresh()
  }, [refresh])

  const retry = async () => {
    const confirmation = task?.retry_confirmation
    if (!confirmation) {
      setError('该任务缺少可验证的冻结重试摘要，不能发起模型调用')
      return
    }
    setIsWorking(true)
    setError(null)
    try {
      setTask(await textbookService.retryGenerationTask(activeTaskID, confirmation.input_hash))
      setShowRetryConfirmation(false)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '重试候选教材稿任务失败')
    } finally {
      setIsWorking(false)
    }
  }

  const tone = task?.status === 'succeeded' ? 'success' : task?.status === 'failed' || task?.status === 'cancelled' ? 'warning' : 'brand'

  return <section className="mt-4 rounded-md border border-[var(--brand)]/30 bg-[var(--brand)]/5 p-3" aria-label="候选教材稿任务状态">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div>
        <p className="text-sm font-medium">候选教材稿任务</p>
        <p className="mt-1 text-xs text-muted-foreground">任务 {activeTaskID.slice(0, 8)}… 使用创建时冻结的资料、合同和章节输入；重试不会改用当前页面的新内容。</p>
      </div>
      {task ? <StatusPill tone={tone}>{task.execution_mode && task.status === 'running' ? '正在复核局部修订' : statusLabel[task.status]}</StatusPill> : <StatusPill tone="brand">正在查询</StatusPill>}
    </div>
    {task?.error_message ? <p role="alert" className="mt-3 text-sm text-destructive">失败原因：{task.error_message}</p> : null}
    {task ? <GenerationFailureDetails task={task} /> : null}
    {task?.execution_mode ? <p className="mt-3 text-xs">本任务仅复核局部修订，不调用模型。</p> : null}
    {task ? <SampleCorrectionPanel key={task.id} task={task} onCreated={id => { setTask(null); setSubmitted({ source: taskID, id }) }} /> : null}
    {task?.status === 'succeeded' ? <p className="mt-3 text-xs text-muted-foreground">任务完成后重新打开本章，即可审阅带证据谱系的候选稿。</p> : null}
    {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
    {task?.status === 'failed' && !task.retry_confirmation ? <p className="mt-3 text-xs text-muted-foreground">该失败任务没有可验证的冻结重试摘要，不会提供模型重试入口。</p> : null}
    {task?.status === 'failed' && task.retry_confirmation && showRetryConfirmation ? <div className="mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm" role="region" aria-label="模型重试确认">
      <p className="font-medium">{task.retry_confirmation.execution_mode ? '确认重新复核同一局部修订' : `确认再次调用 ${task.retry_confirmation.generation_target.provider_name} / ${task.retry_confirmation.generation_target.model_name}`}</p>
      {task.retry_confirmation.execution_mode ? <p className="mt-1 text-xs text-muted-foreground">不调用模型。重新核对冻结原稿、修订内容和当前批准输入；成功后保存待审候选。</p> : <p className="mt-1 text-xs text-muted-foreground">将重用创建时冻结的资料、合同和章节输入；估算输入 {task.retry_confirmation.estimated_input_tokens.toLocaleString('zh-CN')} / {task.retry_confirmation.allowed_input_tokens.toLocaleString('zh-CN')} Token，输出预留 {task.retry_confirmation.reserved_output_tokens.toLocaleString('zh-CN')} Token。再次调用可能产生额外费用，成功后也只生成待审候选。</p>}
      <p className="mt-1 break-all text-xs text-muted-foreground">冻结输入：{task.retry_confirmation.input_hash}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button type="button" size="sm" disabled={isWorking} onClick={() => void retry()}>确认重试一次</Button>
        <Button type="button" size="sm" variant="outline" disabled={isWorking} onClick={() => setShowRetryConfirmation(false)}>取消</Button>
      </div>
    </div> : null}
    <div className="mt-3 flex flex-wrap gap-2">
      <Button type="button" size="sm" variant="outline" disabled={isWorking} onClick={() => void refresh()} className="gap-2"><RefreshCw className="h-3.5 w-3.5" />刷新状态</Button>
      {task?.status === 'failed' && task.retry_confirmation && !showRetryConfirmation ? <Button type="button" size="sm" disabled={isWorking} onClick={() => setShowRetryConfirmation(true)} className="gap-2"><RotateCcw className="h-3.5 w-3.5" />查看重试影响</Button> : null}
    </div>
  </section>
}
