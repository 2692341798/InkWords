import type { SourceLibraryEvidence } from './textbook'

// Resolve only persisted selections; unavailable IDs stay visible through the
// editor's existing placeholder instead of being replaced by search matches.
export async function loadBlueprintEvidence(
  initial: SourceLibraryEvidence[],
  document: Record<string, unknown> | undefined,
  lookup: (ids: string[]) => Promise<SourceLibraryEvidence[]>,
): Promise<SourceLibraryEvidence[]> {
  const selected = new Set<string>()
  if (Array.isArray(document?.volumes)) {
    for (const volume of document.volumes) {
      if (!volume || !Array.isArray(volume.chapters)) continue
      for (const chapter of volume.chapters) {
        if (!chapter || !Array.isArray(chapter.evidence_ids)) continue
        for (const id of chapter.evidence_ids) {
          if (typeof id === 'string' && id.trim()) selected.add(id)
        }
      }
    }
  }
  const merged = new Map(initial.map((item) => [item.id, item]))
  const missing = [...selected].filter((id) => !merged.has(id))
  for (let offset = 0; offset < missing.length;) {
    const batch: string[] = []
    let queryBytes = 0
    // Stay below the gateway's request-line limit even for long Unicode IDs.
    while (offset < missing.length && batch.length < 200) {
      const id = missing[offset]
      const encodedBytes = new URLSearchParams({ chunk_id: id }).toString().length + 1
      if (batch.length > 0 && queryBytes + encodedBytes > 6000) break
      batch.push(id)
      queryBytes += encodedBytes
      offset++
    }
    for (const item of await lookup(batch)) {
      if (batch.includes(item.id)) merged.set(item.id, item)
    }
  }
  return [...merged.values()]
}
