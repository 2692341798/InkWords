import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { textbookService, type TextbookGenerationTaskSnapshot } from '@/services/textbook'

const record = (value: unknown): Record<string, unknown> | null => value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null

export function SampleCorrectionPanel({ task, onCreated }: { task: TextbookGenerationTaskSnapshot; onCreated: (taskID: string) => void }) {
  const [edit, setEdit] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const result = record(task.result)
  if (task.status !== 'failed' || task.execution_mode || result?.rejected_draft_storage !== 'saved'
    || typeof result.project_id !== 'string' || typeof result.chapter_id !== 'string') return null

  const readFile = async (file?: File) => {
    setEdit(null)
    setError('')
    if (!file) return
    try {
      if (file.size > 1024 * 1024) throw new Error('修订文件不能超过 1 MiB')
      const parsed = record(JSON.parse(await file.text()))
      if (parsed?.format !== 'inkwords.sample-correction.v1' || parsed.original_receipt_hash !== result.rejected_draft_hash
        || typeof parsed.reason !== 'string' || typeof parsed.markdown_body !== 'string') throw new Error('修订文件格式不正确，或不属于当前失败任务')
      setEdit(parsed)
    } catch (reason) { setError(reason instanceof Error ? reason.message : '读取修订文件失败') }
  }
  const submit = async () => {
    if (!edit) return
    setBusy(true)
    setError('')
    try {
      const response = await textbookService.correctSample(String(result.project_id), String(result.chapter_id), task.id, edit)
      onCreated(response.data.task_id)
    } catch (reason) { setError(reason instanceof Error ? reason.message : '提交修订失败') }
    finally { setBusy(false) }
  }
  return <section aria-label="局部修订候选" className="mt-3 space-y-2 rounded-md border border-border bg-background p-3 text-xs">
    <h3 className="text-sm font-medium">提交结构化局部修订</h3>
    <p className="text-muted-foreground">选择基于本次留存原稿的修订 JSON。系统会重新核对原稿、批准输入和质量门禁，再保存待审候选；不调用模型，代码仍需独立验证。</p>
    <label className="block">修订文件<input className="mt-1 block w-full" type="file" accept=".json,application/json" disabled={busy} onChange={event => void readFile(event.target.files?.[0])} /></label>
    {edit && <p>修订理由：{String(edit.reason)} · 正文 {String(edit.markdown_body).length.toLocaleString('zh-CN')} 字符</p>}
    {error && <p role="alert" className="text-destructive">{error}</p>}
    <Button size="sm" disabled={!edit || busy} onClick={() => void submit()}>提交局部修订候选（不调用模型）</Button>
  </section>
}
