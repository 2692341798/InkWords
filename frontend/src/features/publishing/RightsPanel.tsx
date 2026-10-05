import { useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import type { EditorialWorkspace, RightsStatus, RightsWorkType } from '@/services/textbook'
import type { PublicationRightsItem } from '@/services/textbook'
import type { RightsAmendmentInput } from '@/services/rightsAmendments'
import { RightsAmendmentForm } from './RightsAmendmentForm'
import { assetFontContext } from './assetFontContext'
import type { AssetFontContext } from './AssetFontReviewFields'

const workTypes: Array<[RightsWorkType, string]> = [
  ['prose', '正文'], ['code', '代码'], ['image', '图片'], ['screenshot', '截图'],
  ['font', '字体'], ['trademark', '商标'], ['data', '数据'],
]

export function RightsPanel({ workspace, disabled, onAdd, onAmend }: {
  workspace: EditorialWorkspace
  disabled: boolean
  onAdd: (input: { subjectRef: string; workType: RightsWorkType; rightsBasis: string; allowedUse: string; attribution: string; publicationStatus: RightsStatus }) => Promise<void>
  onAmend?: (input: RightsAmendmentInput) => Promise<void>
}) {
  const [subjectRef, setSubjectRef] = useState('')
  const [workType, setWorkType] = useState<RightsWorkType>('prose')
  const [rightsBasis, setRightsBasis] = useState('')
  const [allowedUse, setAllowedUse] = useState('')
  const [attribution, setAttribution] = useState('')
  const [publicationStatus, setPublicationStatus] = useState<RightsStatus>('pending')
  const locked = workspace.build.status !== 'ready_for_review'
  const [editing, setEditing] = useState<{ item: PublicationRightsItem; previousId: string | null; fonts?: AssetFontContext } | null>(null)
  const ledger = workspace.rights_ledger?.contract_version === 'inkwords.rights-ledger.v1' ? workspace.rights_ledger : undefined
  const effective = ledger?.effective_items ?? workspace.rights_items
  const covered = new Set(effective.filter((item) => item.publication_status === 'ready').map((item) => `${item.work_type}:${item.subject_ref}`))
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    await onAdd({ subjectRef, workType, rightsBasis, allowedUse, attribution, publicationStatus })
    setSubjectRef('')
    setRightsBasis('')
    setAllowedUse('')
    setAttribution('')
    setPublicationStatus('pending')
  }
  return <section className="rounded-md border border-border p-4" aria-labelledby="rights-title">
    <h3 id="rights-title" className="font-medium">逐作品权利清单</h3>
    <p className="mt-1 text-xs text-muted-foreground">每条记录指向具体作品。原件不可改写；补充证据或更正结论时追加补证，保留完整历史。</p>
    <ul className="mt-3 space-y-2" aria-label="冻结构建要求的权利对象">{workspace.required_rights_subjects.map((subject) => {
      const isCovered = covered.has(`${subject.work_type}:${subject.subject_ref}`)
      const registered = effective.some((item) => item.subject_ref === subject.subject_ref)
      return <li key={`${subject.work_type}:${subject.subject_ref}`} className="flex flex-wrap items-center justify-between gap-2 rounded border border-border px-3 py-2 text-xs"><span><span className="font-medium">{subject.subject_ref}</span> · {subject.work_type}</span><span className={isCovered ? 'text-emerald-600' : 'text-muted-foreground'}>{isCovered ? '已覆盖' : registered ? '已登记，待补证' : '待登记'}</span>{!locked && !registered ? <Button type="button" size="sm" variant="outline" onClick={() => { setSubjectRef(subject.subject_ref); setWorkType(subject.work_type) }}>登记此对象</Button> : null}</li>
    })}</ul>
    {effective.length > 0 ? <ul className="mt-3 space-y-2" aria-label="已登记权利项">{effective.map((item) => <li key={item.id} className="rounded border border-border px-3 py-2 text-xs"><div className="flex flex-wrap justify-between gap-2"><span className="font-medium">{item.subject_ref}</span><span>{item.work_type} · {item.publication_status}</span></div><p className="mt-1 text-muted-foreground">依据：{item.rights_basis}；允许用途：{item.allowed_use}；署名：{item.attribution}</p>{!locked && ledger && onAmend ? <Button type="button" size="sm" variant="outline" disabled={disabled} onClick={() => setEditing({ item, fonts: assetFontContext(workspace, item), previousId: ledger.amendments.filter((a) => a.base_item_id === item.id).sort((a,b) => b.revision - a.revision)[0]?.id ?? null })}>为此对象追加补证</Button> : null}</li>)}</ul> : <p className="mt-3 text-sm text-muted-foreground">尚无权利项，预检会保持阻断。</p>}
    {ledger && ledger.amendments.length > 0 ? <details className="mt-3 text-sm"><summary>权利原件与补证历史（{ledger.amendments.length} 次补证）</summary><ul>{ledger.original_items.map((item) => <li key={item.id}>原件：{item.subject_ref} · {item.publication_status} · {item.rights_basis} · {item.allowed_use} · {item.attribution}</li>)}{ledger.amendments.map((a) => <li key={a.id} className="mt-2 break-words">v{a.revision} · {a.effective_item.subject_ref} · {a.effective_item.publication_status} · {a.reviewer_kind === 'delegated_ai' ? '用户委托 AI' : '本人核对'} / {a.reviewer}：{a.reason}。证据：{a.evidence_refs.join('；')}。依据：{a.effective_item.rights_basis}；用途：{a.effective_item.allowed_use}；署名：{a.effective_item.attribution}{a.asset_font_review ? <p>图片字体核对：{a.asset_font_review.text_status === 'no_text' ? '已检查无文字或字形' : '已识别字体来源'}；{a.asset_font_review.scope}；图片哈希：{a.asset_font_review.content_hash}；字体记录：{a.asset_font_review.fonts.map((font) => `${font.rights_item_id} / ${font.amendment_id || '原件'}`).join('；') || '无'}</p> : null}</li>)}</ul></details> : null}
    {editing && onAmend && !locked ? <RightsAmendmentForm key={`${editing.item.id}:${editing.previousId}`} item={editing.item} assetFontContext={editing.fonts} previousId={editing.previousId} manifestHash={workspace.build.manifest_hash} disabled={disabled} onSave={onAmend} onClose={() => setEditing(null)} /> : null}
    {!locked ? <form className="mt-4 grid gap-3" onSubmit={(event) => void submit(event).catch(() => undefined)}>
      <fieldset disabled={disabled} className="grid gap-3">
      <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_150px]">
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">作品引用</span><input required value={subjectRef} onChange={(event) => setSubjectRef(event.target.value)} placeholder="例如 chapter:01/prose 或 asset:diagram-2" className="w-full rounded-md border border-border bg-background px-3 py-2" /></label>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">作品类型</span><select value={workType} onChange={(event) => setWorkType(event.target.value as RightsWorkType)} className="w-full rounded-md border border-border bg-background px-3 py-2">{workTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">出版状态</span><select value={publicationStatus} onChange={(event) => setPublicationStatus(event.target.value as RightsStatus)} className="w-full rounded-md border border-border bg-background px-3 py-2"><option value="pending">待核对</option><option value="ready">已核对可用</option><option value="blocked">阻断</option></select></label>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">权利依据</span><textarea required value={rightsBasis} onChange={(event) => setRightsBasis(event.target.value)} className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" /></label>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">允许用途</span><textarea required value={allowedUse} onChange={(event) => setAllowedUse(event.target.value)} className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" /></label>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">署名要求</span><textarea required value={attribution} onChange={(event) => setAttribution(event.target.value)} className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" /></label>
      </div>
      <Button type="submit" disabled={disabled}>登记不可变权利项</Button>
      </fieldset>
    </form> : <p className="mt-3 text-xs text-muted-foreground">该构建已进入出版候选，权利证据已冻结。</p>}
  </section>
}
