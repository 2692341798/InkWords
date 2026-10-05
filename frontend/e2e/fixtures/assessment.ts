import type { Page } from '@playwright/test'
import type { AssessmentJob } from '../../src/services/masteryAssessment'
import type { PracticeTask } from '../../src/services/textbook'

// This fixture represents API results only; it never calls a real grading provider.
export async function installAssessmentFixture(page: Page, objective: string, task: PracticeTask, answer: string) {
  const observed = { calls: 0, applications: 0, correctionEvidence: [] as string[][] }
  const preview = { input_hash: 'sha256:fixture-input', request_hash: 'sha256:fixture-request', provider: 'fixture-provider', model: 'fixture-model', request_bytes: 2400, input_byte_limit: 32000, max_output_tokens: 3000 }
  const finding = { text: '回看注册与请求到达两个阶段。', evidence_ids: [task.evidence_ids[0]] }
  const feedback = { criteria: task.rubric.map((item) => ({ id: item.id, score: 3, reason: '作答说明了主要步骤。', answer_quote: answer, evidence_ids: [task.evidence_ids[0]] })), correct_points: [finding], missing_points: [], misconceptions: [], next_hint: finding, remediation: [finding] }
  let job: AssessmentJob | null = null
  await page.route('**/api/v1/mastery/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const prefix = `/api/v1/mastery/objectives/${objective}`
    if (path === `${prefix}/attempts/attempt-0/assessment-preview`) return route.fulfill({ json: { code: 200, data: preview } })
    if (path === `${prefix}/attempts/attempt-0/assessment`) return route.fulfill({ json: { code: 200, data: job } })
    if (path === `${prefix}/attempts/attempt-0/assessments`) {
      observed.calls++
      job = { id: 'job-fixture', objective_id: objective, attempt_id: 'attempt-0', status: 'succeeded', preview, input: { answer, rubric: task.rubric, evidence: [{ id: task.evidence_ids[0], kind: 'source', excerpt: '服务启动时登记条件，请求到达后按方法和路径查找。' }, { id: 'fixture-method-tree', kind: 'source', excerpt: '本次冻结补充片段：先按 HTTP 方法选择对应路由树，再按路径查找处理函数。' }] }, result: { origin: 'automated', provider_calls: 1, latency_millis: 600, usage: { known: true, input_tokens: 500, output_tokens: 300 }, feedback }, corrections: [], effective_feedback: feedback, effective_hash: 'sha256:original' }
      return route.fulfill({ json: { code: 200, data: { ...job, status: 'running', result: undefined, effective_feedback: undefined } } })
    }
    if (path === `${prefix}/assessments/job-fixture`) return route.fulfill({ json: { code: 200, data: job } })
    if (path === `${prefix}/assessments/job-fixture/apply` && job?.effective_feedback) {
      const input = route.request().postDataJSON()
      observed.applications++
      job = { ...job, applied_assessment: { id: input.id, job_id: job.id, sequence_no: observed.applications, feedback_hash: job.effective_hash!, applied_at: '2026-09-05T12:00:00Z', algorithm_version: 'fsrs-v4-assessment-v1', feedback: job.effective_feedback }, schedule: { skill: 'complete', due_at: '2026-09-06T12:00:00Z', reason: '按已应用评分安排下一题' } }
      return route.fulfill({ json: { code: 200, data: job } })
    }
    if (path === `${prefix}/assessments/job-fixture/corrections` && job) {
      const input = route.request().postDataJSON()
      observed.correctionEvidence.push(input.changes[0].evidence_ids)
      job = { ...job, corrections: [...job.corrections, { ...input, reviewer_id: 'local-fixture', corrected_at: '2026-09-05T12:00:00Z' }], effective_hash: 'sha256:corrected', effective_feedback: { ...feedback, criteria: feedback.criteria.map((item) => input.changes.find((change: { id: string }) => change.id === item.id) || item) } }
      return route.fulfill({ json: { code: 200, data: job } })
    }
    return route.fallback()
  })
  return observed
}
