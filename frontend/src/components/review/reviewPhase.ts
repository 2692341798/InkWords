import type { ReviewPhase, ReviewSessionResponse } from '@/services/review'

export function getReviewPhase(session: ReviewSessionResponse): ReviewPhase {
  if (session.phase) {
    return session.phase
  }
  if (session.status === 'completed') {
    return 'completed'
  }
  if ((session.turns ?? []).some((turn) => turn.role === 'user')) {
    return 'coaching'
  }
  return 'reading'
}

export function getReadingContent(session: ReviewSessionResponse) {
  return session.reading_content || session.source_preview || session.session_outline.summary
}
