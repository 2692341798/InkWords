import { describe, expect, it, vi } from 'vitest'
import { loadBlueprintEvidence } from './blueprintEvidence'

describe('loadBlueprintEvidence', () => {
  it('resolves saved selections beyond the first page in bounded batches without loading other evidence', async () => {
    const ids = Array.from({ length: 201 }, (_, i) => `selected-${i}`)
    const initial = [{ id: 'known', document_id: 'doc', document_title: 'file', canonical_locator: 'repo', ordinal: 1 }]
    const document = { volumes: [{ chapters: [{ evidence_ids: ['known', ...ids, ids[0]] }] }] }
    const lookup = vi.fn(async (chunkIDs: string[]) => chunkIDs.map((id) => ({ ...initial[0], id })))
    const result = await loadBlueprintEvidence(initial, document, lookup)
    expect(lookup.mock.calls.map(([batch]) => batch.length)).toEqual([200, 1])
    expect(result.map((item) => item.id)).toEqual(['known', ...ids])
  })

  it('keeps unavailable saved IDs unresolved and never replaces them with unrelated matches', async () => {
    const lookup = vi.fn(async () => [])
    expect(await loadBlueprintEvidence([], { volumes: [{ chapters: [{ evidence_ids: ['deleted'] }] }] }, lookup)).toEqual([])
    expect(lookup).toHaveBeenCalledWith(['deleted'])
    lookup.mockClear()
    expect(await loadBlueprintEvidence([], { volumes: [null, { chapters: [null, { evidence_ids: [null, ''] }] }] }, lookup)).toEqual([])
    expect(lookup).not.toHaveBeenCalled()
  })

  it('also bounds encoded query size for long identifiers', async () => {
    const ids = Array.from({ length: 50 }, (_, i) => `${'中'.repeat(60)}-${i}`)
    const lookup = vi.fn(async () => [])
    await loadBlueprintEvidence([], { volumes: [{ chapters: [{ evidence_ids: ids }] }] }, lookup)
    expect(lookup.mock.calls.length).toBeGreaterThan(1)
    for (const [batch] of lookup.mock.calls as unknown as [string[]][]) {
      expect(new URLSearchParams(batch.map((id) => ['chunk_id', id])).toString().length).toBeLessThan(6000)
    }
  })
})
