import type { SourceLibraryEvidence, SourceRetrievalPlan } from '@/services/textbook'

// Retrieval remains advisory. Adding an option never adds it to a chapter's
// selected IDs, and saved IDs remain removable when the initial list is capped.
export function blueprintEvidenceOptions(initial: SourceLibraryEvidence[], plans: SourceRetrievalPlan[], selectedIDs: string[]) {
  const options = new Map(initial.map((item) => [item.id, {
    id: item.id, label: `${item.document_title} · #${item.ordinal} · ${item.locator?.path || item.canonical_locator}${item.locator?.symbol ? ` · ${item.locator.symbol}${item.locator.start_line ? ` · 第 ${item.locator.start_line}–${item.locator.end_line ?? item.locator.start_line} 行` : ''}` : ''}`,
  }]))
  for (const plan of plans) {
    for (const item of plan.selected) {
      const symbol = item.locator?.symbol
      const location = symbol ? ` · ${symbol}${item.locator?.start_line ? ` · 第 ${item.locator.start_line}–${item.locator.end_line ?? item.locator.start_line} 行` : ''}` : ''
      options.set(item.chunk_id, {
        id: item.chunk_id, label: `${item.document_title} · #${item.ordinal} · ${item.artifact_path || item.canonical_locator}${location}`,
      })
    }
  }
  for (const id of selectedIDs) {
    if (!options.has(id)) options.set(id, { id, label: `已保存的证据片段 · ${id}` })
  }
  return [...options.values()]
}
