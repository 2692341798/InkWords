import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'
import type { PublicationRightsItem, RightsStatus } from './textbook'

export interface AssetFontReview {
  contract_version: 'inkwords.asset-font-review.v1'
  asset_id: string
  content_hash: string
  surface: 'raster'
  text_status: 'no_text' | 'identified_fonts'
  scope: string
  fonts: Array<{ file_hash: string; rights_item_id: string; amendment_id: string }>
}

export interface RightsAmendmentInput {
  asset_font_review?: AssetFontReview
  id: string
  base_item_id: string
  previous_amendment_id: string | null
  manifest_hash: string
  reviewer_kind: 'human' | 'delegated_ai'
  reviewer: string
  delegation_note: string
  reason: string
  evidence_refs: string[]
  rights_basis: string
  allowed_use: string
  attribution: string
  publication_status: RightsStatus
}
export interface RightsAmendment extends Omit<RightsAmendmentInput, 'previous_amendment_id' | 'rights_basis' | 'allowed_use' | 'attribution' | 'publication_status'> {
  contract_version: 'inkwords.rights-amendment.v1'
  build_id: string
  previous_amendment_id: string
  revision: number
  effective_item: PublicationRightsItem
  completed_at: string
}
export interface RightsLedger {
  contract_version: 'inkwords.rights-ledger.v1'
  selection_rule: 'explicit-predecessor-chain-v1'
  original_items: PublicationRightsItem[]
  amendments: RightsAmendment[]
  effective_items: PublicationRightsItem[]
}
export const appendRightsAmendment = (buildId: string, input: RightsAmendmentInput) => requestJson<{ code: number; data: RightsAmendment }>(apiRoutes.coreApi.textbookProjects.rightsAmendments(buildId), { method: 'POST', json: input, fallbackMessage: '保存补证失败，请刷新核对当前版本后重试' })
