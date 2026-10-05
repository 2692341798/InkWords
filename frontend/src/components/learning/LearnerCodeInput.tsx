import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import type { LearnerCodeFile } from '@/services/mastery'
import { learnerCodeError, readLearnerCodeFiles } from '@/lib/learnerCode'

export function LearnerCodeInput({ files, disabled, onChange, onBusy }: { files: LearnerCodeFile[]; disabled: boolean; onChange: (files: LearnerCodeFile[]) => void; onBusy: (busy: boolean) => void }) {
  const [error, setError] = useState<string | null>(null)
  const [reading, setReading] = useState(false)
  const request = useRef(0)
  const select = async (selected: File[]) => {
    const token = ++request.current
    setReading(true); onBusy(true); setError(null)
    try { const loaded = await readLearnerCodeFiles(selected); if (token === request.current) onChange(loaded) }
    catch (cause) { if (token === request.current) setError(cause instanceof TypeError ? '文件不是有效的 UTF-8 文本，请检查编码。' : cause instanceof Error ? cause.message : '读取文件失败。') }
    finally { if (token === request.current) { setReading(false); onBusy(false) } }
  }
  return <details className="mt-4 rounded-md border border-border p-3 text-sm"><summary className="cursor-pointer">随作答保存代码（可选）{files.length ? ` · ${files.length} 个文件` : ''}</summary>
    <p className="my-3 text-xs text-muted-foreground">选择自己编写的 .go 文件及可选 go.mod，最多 32 个文件、总计 256 KiB。与本次作答一同保存后不可替换；后续修改需开始新的练习。保存后可请求模型静态评阅；隔离运行必须另外明确发起，未运行时相关条目保持未知。</p>
    <label className="grid gap-2">选择本次代码文件<input type="file" multiple accept=".go,.mod" disabled={disabled || reading} onChange={(event) => { const selected = Array.from(event.target.files || []); event.target.value = ''; if (selected.length) void select(selected) }} /></label>
    {reading ? <p role="status">正在读取代码文件…</p> : null}
    {error || learnerCodeError(files) ? <p role="alert" className="mt-2 text-destructive">{error || learnerCodeError(files)}</p> : null}
    {files.map((file, index) => <details key={index} className="mt-3 border-t border-border pt-2"><summary className="cursor-pointer break-all">{file.path}</summary><label className="mt-2 grid gap-1">文件路径 {index + 1}<input value={file.path} disabled={disabled || reading} onChange={(event) => onChange(files.map((item, i) => i === index ? { ...item, path: event.target.value } : item))} className="rounded border border-border bg-background p-2" /></label><label className="mt-2 grid gap-1">代码内容 {index + 1}<textarea rows={8} value={file.content} disabled={disabled || reading} onChange={(event) => onChange(files.map((item, i) => i === index ? { ...item, content: event.target.value } : item))} className="rounded border border-border bg-background p-2 font-mono text-xs" /></label><Button type="button" size="sm" variant="ghost" disabled={disabled || reading} onClick={() => onChange(files.filter((_, i) => i !== index))}>移除 {file.path}</Button></details>)}
  </details>
}
