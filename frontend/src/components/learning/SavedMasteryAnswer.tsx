import { useState } from 'react'
import { AssessmentPanel } from './AssessmentPanel'
import { LearnerCodeSnapshot } from './LearnerCodeSnapshot'
import type { MasteryAttemptInput, NextMasteryTask } from '@/services/mastery'

export function SavedMasteryAnswer({ objectiveID, attempt, label, onApplied }: { objectiveID: string; attempt: MasteryAttemptInput & { id: string; learner_artifact_hash?: string }; label: string; onApplied?: (schedule: NextMasteryTask) => void }) {
  const [open, setOpen] = useState(false)
  return <details className="rounded-md border border-border p-3" onToggle={(event) => setOpen(event.currentTarget.open)}><summary className="cursor-pointer">{new Date(attempt.attempted_at).toLocaleString()} · {label} · 自评{attempt.correct ? '达到' : '未达到'}</summary><pre className="mt-3 whitespace-pre-wrap break-words font-mono text-xs">{attempt.answer || '此历史记录未保存作答正文。'}</pre>{open && attempt.learner_artifact_hash ? <LearnerCodeSnapshot objectiveID={objectiveID} attemptID={attempt.id} hash={attempt.learner_artifact_hash} /> : null}{open && attempt.practice_session_id ? <AssessmentPanel objectiveID={objectiveID} attemptID={attempt.id} onApplied={onApplied} /> : null}</details>
}
