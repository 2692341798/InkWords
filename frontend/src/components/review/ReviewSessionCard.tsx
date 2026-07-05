import { useState } from 'react'
import type { FinalFeedback, HintResponse, ReviewMode, ReviewSessionResponse } from '@/services/review'
import { ReviewFeedbackView } from './ReviewFeedbackView'
import { ReviewProgress } from './ReviewProgress'
import { ReviewReadingView } from './ReviewReadingView'
import { ReviewRecallView } from './ReviewRecallView'
import { getReviewPhase } from './reviewPhase'

interface ReviewSessionCardProps {
  session: ReviewSessionResponse | null
  selectedMode: ReviewMode
  latestStageFeedback: string | null
  latestHint: HintResponse | string | null
  finalFeedback: FinalFeedback | null
  pending?: boolean
  onModeChange: (mode: ReviewMode) => void
  onCompleteReading?: () => Promise<void> | void
  onRespond: (answer: string) => Promise<void> | void
  onRequestHint: (answer?: string) => Promise<void> | void
  onFinish: () => Promise<void> | void
  onClose?: () => void
}

export function ReviewSessionCard({ session, latestStageFeedback, latestHint, finalFeedback, pending = false, onCompleteReading = () => {}, onRespond, onRequestHint, onFinish, onClose }: ReviewSessionCardProps) {
  const [revisingFromTurnIndex, setRevisingFromTurnIndex] = useState<number | null>(null)

  if (!session) return null

  const phase = getReviewPhase(session)
  const displayedPhase = phase === 'coaching' && revisingFromTurnIndex === session.turn_index ? 'recalling' : phase

  return (
    <fieldset disabled={pending} className={pending ? 'opacity-70' : ''} aria-busy={pending}>
      <div className="mb-8"><ReviewProgress phase={displayedPhase} /></div>
      {displayedPhase === 'reading' ? <ReviewReadingView session={session} onComplete={onCompleteReading} /> : null}
      {displayedPhase === 'recalling' ? <ReviewRecallView session={session} latestHint={latestHint} pending={pending} onRespond={onRespond} onRequestHint={onRequestHint} /> : null}
      {displayedPhase === 'coaching' || displayedPhase === 'completed' ? (
        <div aria-live="polite">
          {latestStageFeedback ? <p className="sr-only">{latestStageFeedback}</p> : null}
          <ReviewFeedbackView
            session={{ ...session, phase: displayedPhase }}
            latestHint={latestHint}
            finalFeedback={finalFeedback}
            onContinue={() => setRevisingFromTurnIndex(session.turn_index)}
            onDeepen={onRequestHint}
            onFinish={onFinish}
            onClose={onClose}
          />
        </div>
      ) : null}
    </fieldset>
  )
}
