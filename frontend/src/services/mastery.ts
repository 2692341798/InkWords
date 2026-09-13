import { requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'
import type { LearningProjection, PracticeTask } from './textbook'

export type MasterySkill = 'explain' | 'complete' | 'reproduce' | 'transfer' | 'diagnose' | 'retain'

export interface DueMasteryTask {
  objective_id: string
  chapter_id: string
  title: string
  skill: MasterySkill
  due_at: string
  reason: string
}

export interface MasteryAttemptInput {
	code_files?: LearnerCodeFile[]
  skill: MasterySkill
  answer?: string
  practice_task_id?: string
  practice_content_hash?: string
  practice_session_id?: string
  correct: boolean
  independent: boolean
  hint_count: number
  took_millis: number
  confidence: number
  error_kinds: string[]
  attempted_at: string
}

export interface LearnerCodeFile { path: string; content: string }
export interface LearnerArtifact { format: 'inkwords.learner-artifact.v1'; workspace_id: string; objective_id: string; attempt_id: string; revision_id: string; session_id: string; task_id: string; practice_content_hash: string; skill: MasterySkill; submitted_at: string; files: LearnerCodeFile[]; snapshot_hash: string }
export interface LearnerVerificationPreview {
  capability: { format: 'inkwords.learner-verification-capability.v1'; accepted: true; available: boolean; profile: 'inkwords.learner-go-test-offline.v2'; runner?: { image_digest: string; toolchain_version: string }; reason?: string }
  input_hash?: string
  snapshot_hash?: string
  files_hash?: string
  execution_files_hash?: string
  derived_files?: LearnerCodeFile[]
  requires_explicit_start: true
}
type LearnerVerificationStatus = 'queued' | 'running' | 'passed' | 'failed' | 'timed_out' | 'cancelled' | 'unavailable' | 'runner_error' | 'interrupted'
interface LearnerExecutionPolicy {
  command: string[]; environment: string[]; network: 'unshared'; uid: number; gid: number; memory_bytes: number; pids: number
  file_bytes: number; output_bytes: number; cpu_seconds: number; timeout_millis: number; missing_module_policy: 'inject-inkwords-local-module-v1'
}
interface LearnerVerificationReport {
  format: 'inkwords.learner-verification-report.v1'; run_id: string; input_hash: string; snapshot_hash: string; execution_tree_hash?: string
  runner: { image_digest: string; toolchain_version: string }; profile: 'inkwords.learner-go-test-offline.v2'; policy: LearnerExecutionPolicy; status: Exclude<LearnerVerificationStatus, 'queued' | 'running' | 'interrupted'>
  execution_started: boolean; started_at?: string; completed_at: string; exit_code?: number; output?: string; output_bytes: number; output_truncated: boolean; reason?: string
}
export interface LearnerVerificationJob {
  id: string; objective_id: string; attempt_id: string; request_id: string; retry_of?: string; status: LearnerVerificationStatus
  preview: LearnerVerificationPreview; report?: LearnerVerificationReport; error_code?: string; created_at: string; started_at?: string; completed_at?: string
}

export interface NextMasteryTask {
  skill: MasterySkill
  due_at: string
  reason: string
}

export interface MasteryObjectiveInput {
  chapter_id: string
  title: string
  behavior: string
  skills: MasterySkill[]
  rubric: string[]
  key_points: string[]
  prerequisites: string[]
  evidence_refs: string[]
}

export interface MasteryObjective { id: string }

export interface PracticeSession {
  id: string
  objective_id: string
  skill: MasterySkill
  practice_task_id: string
  practice_content_hash: string
  started_at: string
  submitted_at?: string
  hints_shown: number
  answer_shown: boolean
  hint_count: number
  result?: NextMasteryTask
}

export interface PracticeHelpInput { kind: 'hint' | 'answer'; level: number }

export interface MasteryObjectiveWorkspace {
  objective: MasteryObjective & Omit<MasteryObjectiveInput, 'skills'> & { required_skills: MasterySkill[]; practice_revision_id?: string; practice_content_hash?: string; practice_projection?: LearningProjection }
  attempts: (MasteryAttemptInput & { id: string; learner_artifact_hash?: string })[]
  practice?: PracticeTask
  practice_session?: PracticeSession
  practice_error?: string
}

interface DueMasteryResponse {
	tasks?: DueMasteryTask[] | null
}

// Why: 到期复习是独立于 Obsidian 会话的掌握进度投影；首页只在一次主动打开时读取它。
export const masteryService = {
  learnerVerificationPreview(objectiveID: string, attemptID: string, signal?: AbortSignal) {
    return requestEnvelope<LearnerVerificationPreview>(`${apiRoutes.reviewService.masteryAssessmentAttempt(objectiveID, attemptID)}/verification-preview`, { signal, fallbackMessage: '读取代码隔离预检失败' })
  },
  learnerVerificationLatest(objectiveID: string, attemptID: string, signal?: AbortSignal) {
    return requestEnvelope<LearnerVerificationJob | null>(`${apiRoutes.reviewService.masteryAssessmentAttempt(objectiveID, attemptID)}/verification`, { signal, fallbackMessage: '读取代码验证状态失败' })
  },
  startLearnerVerification(objectiveID: string, attemptID: string, input: { request_id: string; expected_input_hash: string; retry_of?: string }, signal?: AbortSignal) {
    return requestEnvelope<LearnerVerificationJob>(`${apiRoutes.reviewService.masteryAssessmentAttempt(objectiveID, attemptID)}/verifications`, { method: 'POST', json: input, signal, fallbackMessage: '启动代码验证失败' })
  },
  readLearnerVerification(objectiveID: string, runID: string, signal?: AbortSignal) {
    return requestEnvelope<LearnerVerificationJob>(`${apiRoutes.reviewService.masteryObjective(objectiveID)}/verifications/${encodeURIComponent(runID)}`, { signal, fallbackMessage: '刷新代码验证状态失败' })
  },
  cancelLearnerVerification(objectiveID: string, runID: string, signal?: AbortSignal) {
    return requestEnvelope<LearnerVerificationJob>(`${apiRoutes.reviewService.masteryObjective(objectiveID)}/verifications/${encodeURIComponent(runID)}/cancel`, { method: 'POST', signal, fallbackMessage: '取消代码验证失败' })
  },
  learnerArtifact(objectiveID: string, attemptID: string, signal?: AbortSignal) {
    return requestEnvelope<LearnerArtifact | null>(`${apiRoutes.reviewService.masteryAssessmentAttempt(objectiveID, attemptID)}/code`, { signal, fallbackMessage: '读取本次代码快照失败' })
  },
  async getDue(): Promise<{ tasks: DueMasteryTask[] }> {
    const response = await requestEnvelope<DueMasteryResponse>(apiRoutes.reviewService.masteryDue, {
      fallbackMessage: '请求到期巩固任务失败',
    })
    return { tasks: Array.isArray(response.tasks) ? response.tasks : [] }
  },
  createObjective(input: MasteryObjectiveInput) {
    return requestEnvelope<MasteryObjective>(apiRoutes.reviewService.masteryObjectives, {
      method: 'POST', json: input, fallbackMessage: '创建学习目标失败',
    })
  },
  async getWorkspace(objectiveID: string, skill?: MasterySkill, sessionID?: string): Promise<MasteryObjectiveWorkspace> {
    const workspace = await requestEnvelope<MasteryObjectiveWorkspace>(apiRoutes.reviewService.masteryObjective(objectiveID), {
      fallbackMessage: '读取学习目标与作答记录失败',
    })
    if (!workspace.objective.chapter_id.startsWith('approved-revision:') || !skill) return workspace
    const identity = workspace.objective.chapter_id.split(':')
    const learning = workspace.objective.practice_projection
    if (identity.length !== 3 || !learning || learning.chapter_id !== identity[1] || learning.revision_id !== identity[2] || workspace.objective.practice_revision_id !== identity[2] || workspace.objective.practice_content_hash !== learning.content_hash) return { ...workspace, practice_error: '该目标缺少匹配的服务端题目依据；原作答记录仍保留，请从已批准教材创建学习目标。' }
    if (learning.format !== 'inkwords.learning-projection.v2' || learning.practice_set?.version !== 'inkwords.practice-set.v1') return { ...workspace, practice_error: '该批准稿尚无具体六维题目，请先补齐并审阅教材练习。' }
    const practice = learning.practice_set.tasks.find((item) => item.mode === skill)
    if (!practice) return { ...workspace, practice_error: '批准稿缺少本次练习类型。' }
    try {
      const session = await requestEnvelope<PracticeSession>(sessionID ? apiRoutes.reviewService.masterySession(objectiveID, sessionID) : apiRoutes.reviewService.masterySessions(objectiveID), sessionID
        ? { fallbackMessage: '恢复练习会话失败' }
        : { method: 'POST', json: { skill }, fallbackMessage: '开始练习会话失败' })
      if (session.objective_id !== objectiveID || session.skill !== skill || session.practice_task_id !== practice.id || session.practice_content_hash !== learning.content_hash) throw new Error('练习会话与批准稿依据不匹配。')
      return { ...workspace, practice, practice_session: session }
    } catch (cause) {
      return { ...workspace, practice_error: cause instanceof Error ? cause.message : '读取练习会话失败' }
    }
  },
  revealHelp(objectiveID: string, sessionID: string, input: PracticeHelpInput) {
    return requestEnvelope<PracticeSession>(`${apiRoutes.reviewService.masterySession(objectiveID, sessionID)}/help`, {
      method: 'POST', json: input, fallbackMessage: '保存帮助记录失败，内容尚未展开，请重试',
    })
  },
  recordAttempt(objectiveID: string, input: MasteryAttemptInput) {
    return requestEnvelope<NextMasteryTask>(apiRoutes.reviewService.masteryAttempts(objectiveID), {
      method: 'POST', json: input, fallbackMessage: '记录学习表现失败',
    })
  },
}
