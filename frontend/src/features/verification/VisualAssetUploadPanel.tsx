import { useState } from 'react'
import { ImagePlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ChapterRevision, TextbookRuntimeEvidence } from '@/services/textbook'

type VisualPurpose = 'layout' | 'memory_map' | 'call_stack' | 'network_flow' | 'rendered_ui'
interface Props {
  revisions: ChapterRevision[]
  evidence: TextbookRuntimeEvidence[]
  disabled: boolean
  onUpload: (input: {
    revisionId: string; evidenceId: string; file: File; altText: string; source: string
    generationMethod: string; visualPurpose: VisualPurpose; rightsStatus: 'pending'
  }) => Promise<void>
}

export function VisualAssetUploadPanel({ revisions, evidence, disabled, onUpload }: Props) {
  const [file, setFile] = useState<File | null>(null)
  const [altText, setAltText] = useState('')
  const [source, setSource] = useState('本地 IDE 人工截图')
  const [visualPurpose, setVisualPurpose] = useState<VisualPurpose>('call_stack')
  const [evidenceId, setEvidenceId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const revisionsById = new Map(revisions.map(item => [item.id, item]))
  const availableEvidence = evidence.filter(item => revisionsById.has(item.revision_id))
  // Both IDs must come from the same receipt, including after a workspace refresh.
  const evidenceItem = availableEvidence.find(item => item.id === evidenceId)
  const revision = evidenceItem ? revisionsById.get(evidenceItem.revision_id) : undefined
  const busy = disabled || uploading

  async function upload() {
    if (busy || !file || !revision || !evidenceItem || !altText.trim() || !source.trim()) return
    setError(null)
    setUploading(true)
    try {
      await onUpload({ revisionId: revision.id, evidenceId: evidenceItem.id, file, altText, source, generationMethod: 'manual_capture', visualPurpose, rightsStatus: 'pending' })
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '登记截图失败')
    } finally {
      setUploading(false)
    }
  }

  if (availableEvidence.length === 0) return null

  return <section className="mt-4 rounded-md border border-dashed border-border p-3 text-sm">
    <p className="flex items-center gap-2 font-medium"><ImagePlus className="h-4 w-4" />登记人工截图</p>
    <p className="mt-1 text-xs text-muted-foreground">运行输出默认应保留为结构化文本。只有布局、内存图、调用栈、网络流或实际界面状态本身构成证据时才上传截图；上传不会改变运行验证状态。</p>
    <div className="mt-3 grid gap-2 sm:grid-cols-2">
      <label className="sm:col-span-2">截图关联运行证据
        <select aria-label="截图关联运行证据" className="mt-1 w-full rounded border border-border bg-background px-2 py-1" disabled={busy} value={evidenceItem?.id ?? ''} onChange={event => { setEvidenceId(event.target.value); setError(null) }}>
          <option value="">请选择与截图对应的修订和运行记录</option>
          {availableEvidence.map(item => <option key={item.id} value={item.id}>r{revisionsById.get(item.revision_id)!.revision_number} · {item.tool_name || item.kind} · {item.captured_at || '未记录采集时间'} · {item.id}</option>)}
        </select>
      </label>
      <input aria-label="截图文件" type="file" accept="image/png,image/jpeg,image/webp" disabled={busy} onChange={event => setFile(event.target.files?.[0] ?? null)} />
      <input aria-label="截图替代文本" className="rounded border border-border bg-background px-2 py-1" placeholder="替代文本" value={altText} disabled={busy} onChange={event => setAltText(event.target.value)} />
      <select aria-label="截图观察目的" className="rounded border border-border bg-background px-2 py-1" value={visualPurpose} disabled={busy} onChange={event => setVisualPurpose(event.target.value as VisualPurpose)}>
        <option value="layout">页面布局</option><option value="memory_map">内存图</option><option value="call_stack">调用栈</option><option value="network_flow">网络流</option><option value="rendered_ui">实际界面状态</option>
      </select>
      <input aria-label="截图来源" className="rounded border border-border bg-background px-2 py-1 sm:col-span-2" value={source} disabled={busy} onChange={event => setSource(event.target.value)} />
    </div>
    {revision && evidenceItem ? <div className="mt-3 space-y-1 break-all text-xs text-muted-foreground">
      <p>关联修订：r{revision.revision_number}；代码工件：{evidenceItem.code_artifact_id}</p>
      <p>代码指纹：{evidenceItem.code_artifact_hash}</p>
      <p>请核对截图中的代码与所选运行记录一致。历史记录的截图只作归档，不能证明当前批准稿已验证。</p>
      {evidenceItem.stale_reason ? <p className="text-[var(--warning)]">{evidenceItem.stale_reason}</p> : null}
      {evidenceItem.expires_at ? <p>运行证据有效期至：{new Date(evidenceItem.expires_at).toLocaleString()}</p> : <p>未记录运行证据有效期。</p>}
    </div> : null}
    {error ? <p role="alert" className="mt-2 text-xs text-destructive">{error}</p> : null}
    <Button className="mt-3" size="sm" variant="outline" disabled={busy || !file || !revision || !evidenceItem || !altText.trim() || !source.trim()} onClick={() => void upload()}>{uploading ? '正在登记截图…' : '登记截图资产'}</Button>
  </section>
}
