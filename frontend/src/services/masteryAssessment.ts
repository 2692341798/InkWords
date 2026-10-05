import { requestEnvelope } from './apiClient'
import { apiRoutes } from './apiRoutes'
import type { LearnerArtifact, NextMasteryTask } from './mastery'

export interface AssessmentPreview { input_hash: string; request_hash: string; provider: string; model: string; request_bytes: number; input_byte_limit: number; max_output_tokens: number; verification_run_id?: string; runtime_status?: 'passed' | 'failed' | 'timed_out' }
export interface CriterionAssessment { id: string; score: number | null; reason: string; answer_quote: string; evidence_ids: string[]; answer_path?: string }
export interface AssessmentFinding { text: string; evidence_ids: string[] }
export interface AssessmentDecision { criterion_id: string; requirement: string; status: 'satisfied' | 'unsatisfied' | 'unknown'; gaps: { requirement_quote: string; reason: string }[] }
export interface AssessmentTaskReference { task_id: string; practice_content_hash: string; expected_answer: string }
export interface AssessmentFeedback { criteria: CriterionAssessment[]; correct_points: AssessmentFinding[]; missing_points: AssessmentFinding[]; misconceptions: AssessmentFinding[]; next_hint: AssessmentFinding; remediation: AssessmentFinding[] }
export type AssessmentFindingCorrection = Omit<AssessmentFeedback, 'criteria'>
export interface AssessmentCorrectionInput { id: string; previous_hash: string; reason: string; changes: CriterionAssessment[]; findings?: AssessmentFindingCorrection }
export interface AssessmentApplyInput { id: string; expected_feedback_hash: string; previous_id?: string }
interface AssessmentApplication { id: string; job_id: string; sequence_no: number; feedback_hash: string; applied_at: string; algorithm_version: string; feedback: AssessmentFeedback }
export interface AssessmentJob {
  id: string; objective_id: string; attempt_id: string; retry_of?: string; created_at?: string; status: 'running' | 'succeeded' | 'failed' | 'cancelled' | 'interrupted'; error_code?: string
  preview: AssessmentPreview
  input: { answer: string; artifact_hash?: string; rubric: { id: string; description: string; requires_runtime: boolean }[]; evidence: { id: string; kind: string; excerpt: string; verification_run_id?: string; artifact_hash?: string }[]; learner_artifact?: LearnerArtifact; task_reference?: AssessmentTaskReference; decision_policy?: string }
  result?: { origin: 'automated'; provider_calls: number; latency_millis: number; usage: { known: boolean; input_tokens?: number; output_tokens?: number }; feedback?: AssessmentFeedback; decisions?: AssessmentDecision[] }
  corrections: (AssessmentCorrectionInput & { reviewer_id: string; corrected_at: string })[]
  effective_feedback?: AssessmentFeedback; effective_hash?: string
  applied_assessment?: AssessmentApplication; schedule?: NextMasteryTask
}

const attemptPath = apiRoutes.reviewService.masteryAssessmentAttempt
const jobPath = apiRoutes.reviewService.masteryAssessmentJob
export const masteryAssessmentService = {
  preview: (objective: string, attempt: string) => requestEnvelope<AssessmentPreview>(`${attemptPath(objective, attempt)}/assessment-preview`, { fallbackMessage: '准备评分依据失败' }),
  latest: (objective: string, attempt: string, signal?: AbortSignal) => requestEnvelope<AssessmentJob | null>(`${attemptPath(objective, attempt)}/assessment`, { signal, fallbackMessage: '读取评分状态失败' }),
  read: (objective: string, job: string, signal?: AbortSignal) => requestEnvelope<AssessmentJob>(jobPath(objective, job), { signal, fallbackMessage: '读取评分状态失败' }),
  start: (objective: string, attempt: string, input: { request_id: string; expected_input_hash: string; expected_request_hash: string; retry_of?: string }) => requestEnvelope<AssessmentJob>(`${attemptPath(objective, attempt)}/assessments`, { method: 'POST', json: input, fallbackMessage: '提交评分失败；可读取任务状态确认结果' }),
  cancel: (objective: string, job: string) => requestEnvelope<AssessmentJob>(`${jobPath(objective, job)}/cancel`, { method: 'POST', fallbackMessage: '取消评分失败' }),
  correct: (objective: string, job: string, input: AssessmentCorrectionInput) => requestEnvelope<AssessmentJob>(`${jobPath(objective, job)}/corrections`, { method: 'POST', json: input, fallbackMessage: '保存纠正失败，请保留内容并重新读取评分' }),
  apply: (objective: string, job: string, input: AssessmentApplyInput) => requestEnvelope<AssessmentJob>(`${jobPath(objective, job)}/apply`, { method: 'POST', json: input, fallbackMessage: '应用评分失败，请读取最新评分后核对' }),
}
