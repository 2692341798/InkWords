import type { ChapterRevision } from '@/services/textbook'

export function correctionProvenance(revision: ChapterRevision) {
  const value = revision.document_json?.correction
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const correction = value as Record<string, unknown>
  return correction.origin === 'automated_local_correction' ? correction : null
}
