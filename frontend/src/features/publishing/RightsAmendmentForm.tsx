import { useRef, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import type { PublicationRightsItem, RightsStatus } from '@/services/textbook'
import type { AssetFontReview, RightsAmendmentInput } from '@/services/rightsAmendments'
import { AssetFontReviewFields, type AssetFontContext } from './AssetFontReviewFields'

const control = 'w-full rounded-md border border-border bg-background px-3 py-2'

export function RightsAmendmentForm({ item, manifestHash, previousId, disabled, onSave, onClose, assetFontContext }: {
  item: PublicationRightsItem; manifestHash: string; previousId: string | null; disabled: boolean
  onSave: (input: RightsAmendmentInput) => Promise<void>; onClose: () => void
  assetFontContext?: AssetFontContext
}) {
  const [basis, setBasis] = useState(item.rights_basis)
  const [allowed, setAllowed] = useState(item.allowed_use)
  const [attribution, setAttribution] = useState(item.attribution)
  const [status, setStatus] = useState<RightsStatus>(item.publication_status)
  const [kind, setKind] = useState<'human' | 'delegated_ai'>('human')
  const [reviewer, setReviewer] = useState('')
  const [delegation, setDelegation] = useState('')
  const [reason, setReason] = useState('')
  const [evidence, setEvidence] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [assetFonts, setAssetFonts] = useState<AssetFontReview | undefined>()
  const request = useRef<{ fingerprint: string; id: string } | null>(null)
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (disabled || saving) return
    if (assetFonts?.text_status === 'identified_fonts' && assetFonts.fonts.length === 0) { setError('请选出已核对的字体；来源未齐时请保留“尚未核对”。'); return }
    const input = { base_item_id: item.id, previous_amendment_id: previousId, manifest_hash: manifestHash, reviewer_kind: kind, reviewer: reviewer.trim(), delegation_note: kind === 'delegated_ai' ? delegation.trim() : '', reason: reason.trim(), evidence_refs: evidence.split('\n').map((line) => line.trim()).filter(Boolean), rights_basis: basis.trim(), allowed_use: allowed.trim(), attribution: attribution.trim(), publication_status: status }
    const payload = assetFonts ? { ...input, asset_font_review: assetFonts } : input
    const fingerprint = JSON.stringify(payload)
    if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, id: crypto.randomUUID() }
    setSaving(true); setError(null)
    try { await onSave({ ...payload, id: request.current.id }); onClose() }
    catch (cause) { setError(cause instanceof Error ? cause.message : '补证保存失败，原记录保留。请重试或刷新核对。') }
    finally { setSaving(false) }
  }
  return <form className="mt-3 grid gap-3 rounded border border-border p-3" aria-label="追加权利补证" onSubmit={(event) => void submit(event)}>
    <p className="text-sm break-words">补证对象：{item.subject_ref}。原记录与历史版本保留；提交后需要重新进行权利审阅。</p>
    <fieldset className="grid gap-3" disabled={disabled || saving}>
    <label>补证后状态<select className={control} value={status} onChange={(e) => setStatus(e.target.value as RightsStatus)}><option value="pending">待核对</option><option value="ready">已核对可用</option><option value="blocked">阻断</option></select></label>
    <label>补证权利依据<textarea className={control} required maxLength={4000} value={basis} onChange={(e) => setBasis(e.target.value)} /></label>
    <label>补证允许用途<textarea className={control} required maxLength={4000} value={allowed} onChange={(e) => setAllowed(e.target.value)} /></label>
    <label>补证署名要求<textarea className={control} required maxLength={4000} value={attribution} onChange={(e) => setAttribution(e.target.value)} /></label>
    <label>补证审阅来源<select className={control} value={kind} onChange={(e) => setKind(e.target.value as typeof kind)}><option value="human">本人核对</option><option value="delegated_ai">用户委托 AI 核对</option></select></label>
    <label>补证审阅者<input className={control} required maxLength={128} value={reviewer} onChange={(e) => setReviewer(e.target.value)} /></label>
    {kind === 'delegated_ai' ? <label>补证授权说明<textarea className={control} required minLength={8} maxLength={4000} value={delegation} onChange={(e) => setDelegation(e.target.value)} /></label> : null}
    <label>补证理由<textarea className={control} required minLength={8} maxLength={4000} value={reason} onChange={(e) => setReason(e.target.value)} /></label>
    <label>补证证据引用（每行一项）<textarea className={control} required value={evidence} onChange={(e) => setEvidence(e.target.value)} /></label>
    {assetFontContext ? <AssetFontReviewFields context={assetFontContext} value={assetFonts} onChange={setAssetFonts} /> : null}
    </fieldset>
    {error ? <p role="alert" className="text-destructive">{error}</p> : null}
    <div className="flex gap-2"><Button type="submit" disabled={disabled || saving}>{saving ? '正在保存补证…' : '保存追加补证'}</Button><Button type="button" variant="outline" disabled={saving} onClick={onClose}>关闭补证</Button></div>
  </form>
}
