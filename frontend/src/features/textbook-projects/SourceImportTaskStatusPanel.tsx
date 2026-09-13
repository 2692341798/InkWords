import { useCallback, useEffect, useRef, useState } from 'react'
import { RefreshCw, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { StatusPill } from '@/components/ui/workspace'
import { textbookService, type TextbookGenerationTaskSnapshot } from '@/services/textbook'

interface SourceImportTaskStatusPanelProps {
  taskID: string
  onSucceeded: () => void
}

const statusLabel: Record<TextbookGenerationTaskSnapshot['status'], string> = {
	pending: '准备解析',
  queued: '等待解析',
  running: '正在解析',
	streaming: '正在接收解析结果',
  succeeded: '资料已入库',
  failed: '导入失败',
  cancelled: '已取消',
}

export function SourceImportTaskStatusPanel({ taskID, onSucceeded }: SourceImportTaskStatusPanelProps) {
  const [task, setTask] = useState<TextbookGenerationTaskSnapshot | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const refreshedTaskID = useRef<string | null>(null)

  const refresh = useCallback(async () => {
    setIsRefreshing(true)
    setError(null)
    try {
      const next = await textbookService.getTask(taskID)
      setTask(next)
      if (next.status === 'succeeded' && refreshedTaskID.current !== taskID) {
        refreshedTaskID.current = taskID
        onSucceeded()
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '查询资料导入任务失败')
    } finally {
      setIsRefreshing(false)
    }
  }, [onSucceeded, taskID])

  const retry = async () => {
    setIsRefreshing(true)
    setError(null)
    try {
      setTask(await textbookService.retryTask(taskID))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '重试资料导入任务失败')
    } finally {
      setIsRefreshing(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [refresh])

  const tone = task?.status === 'succeeded' ? 'success' : task?.status === 'failed' || task?.status === 'cancelled' ? 'warning' : 'brand'
  const isOfficialWeb = task?.task_subtype === 'textbook_official_web_import'
  return <section className="mt-4 rounded-md border border-[var(--brand)]/30 bg-[var(--brand)]/5 p-3" aria-label="资料导入任务状态">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div>
        <p className="text-sm font-medium">资料导入任务</p>
        <p className="mt-1 text-xs text-muted-foreground">任务 {taskID.slice(0, 8)}… {isOfficialWeb ? '按创建时确认的官网范围抓取，成功后保存不可变快照；重试会重新访问官网。' : '只解析创建时冻结的文件和固定版本；完成后资料库会自动刷新。'}</p>
      </div>
      {task ? <StatusPill tone={tone}>{statusLabel[task.status]}</StatusPill> : <StatusPill tone="brand">正在查询</StatusPill>}
    </div>
    {task?.error_message ? <p role="alert" className="mt-3 text-sm text-destructive">失败原因：{task.error_message}。{isOfficialWeb ? '可按同一范围重新抓取；官网内容可能已发生变化。' : '可重试同一冻结文件；若文件或固定版本有误，请修正后重新提交。'}</p> : null}
    {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
    <div className="mt-3 flex flex-wrap gap-2">
      <Button type="button" size="sm" variant="outline" disabled={isRefreshing} onClick={() => void refresh()} className="gap-2"><RefreshCw className="h-3.5 w-3.5" />刷新状态</Button>
      {task?.status === 'failed' ? <Button type="button" size="sm" disabled={isRefreshing} onClick={() => void retry()} className="gap-2"><RotateCcw className="h-3.5 w-3.5" />{isOfficialWeb ? '按原范围重新抓取' : '重试冻结导入'}</Button> : null}
    </div>
  </section>
}
