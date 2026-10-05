import type { EditorialWorkspace, PublicationRightsItem } from '@/services/textbook'
import type { AssetFontContext } from './AssetFontReviewFields'

// Snapshot the frozen asset and current font rights when editing opens. A
// refresh must not silently change the versions the reviewer is approving.
export function assetFontContext(workspace: EditorialWorkspace, item: PublicationRightsItem): AssetFontContext | undefined {
  if (item.work_type !== 'image' && item.work_type !== 'screenshot') return undefined
  const raw = workspace.build.manifest_json?.assets
  if (!Array.isArray(raw)) return undefined
  const assets = raw.filter((a): a is { id: string; content_hash: string; kind: string } => !!a && typeof a === 'object' && typeof a.id === 'string' && typeof a.content_hash === 'string' && ['screenshot', 'diagram'].includes(a.kind))
    .filter((a) => `asset:${a.id}` === item.subject_ref && /^sha256:[a-f0-9]{64}$/.test(a.content_hash))
  if (assets.length !== 1 || !workspace.rights_ledger) return undefined
  const ledger = workspace.rights_ledger
  return { asset: { id: assets[0].id, content_hash: assets[0].content_hash }, fonts: ledger.effective_items.filter((font) => font.work_type === 'font' && font.publication_status === 'ready' && font.build_id === item.build_id && font.project_id === item.project_id && /^font-file:sha256:[a-f0-9]{64}$/.test(font.subject_ref)).map((font) => ({ label: `${font.attribution}（${font.subject_ref.slice(17, 29)}…）`, file_hash: font.subject_ref.slice('font-file:'.length), rights_item_id: font.id, amendment_id: [...ledger.amendments].filter((a) => a.base_item_id === font.id).sort((a, b) => b.revision - a.revision)[0]?.id ?? '' })) }
}
