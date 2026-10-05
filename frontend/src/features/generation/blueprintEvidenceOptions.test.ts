import { describe, expect, it } from 'vitest'
import { blueprintEvidenceOptions } from './blueprintEvidenceOptions'

describe('blueprintEvidenceOptions', () => {
  it('shows the saved declaration location after a fresh workspace load without retrieval', () => {
    const saved = { id: 'selected', document_id: 'doc', document_title: 'tree.go', canonical_locator: 'repo', ordinal: 23, locator: { path: 'tree.go', symbol: '(*node).getValue', start_line: 413, end_line: 665 } }
    expect(blueprintEvidenceOptions([saved], [], ['selected'])).toEqual([{ id: 'selected', label: 'tree.go · #23 · tree.go · (*node).getValue · 第 413–665 行' }])
  })
  it('deduplicates a retrieved declaration and displays its recorded symbol and physical lines', () => {
    const initial = [{ id: 'chunk', document_id: 'doc', document_title: 'tree.go', canonical_locator: 'repo', ordinal: 10 }]
    const plan = { id: 'plan', query: 'getValue', input_hash: 'hash', candidates: [], created_at: 'today', selected: [{ chunk_id: 'chunk', document_id: 'doc', document_title: 'tree.go', snapshot_id: 'snapshot', source_role: 'primary' as const, canonical_locator: 'repo', artifact_path: 'tree.go', ordinal: 10, score: 12, reasons: [], locator: { symbol: '(*node).getValue', start_line: 418, end_line: 679 } }] }
    expect(blueprintEvidenceOptions(initial, [plan, plan], ['chunk'])).toEqual([{ id: 'chunk', label: 'tree.go · #10 · tree.go · (*node).getValue · 第 418–679 行' }])
  })
})
