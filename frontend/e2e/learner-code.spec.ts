import { expect, test } from './fixtures/app'
import { practiceSet } from './fixtures/practice'

test('@core @learner-code freezes submitted files and restores them after a lost response', async ({ appPage: page }, testInfo) => {
  const objective = '99999999-9999-9999-9999-999999999999'
  const task = practiceSet.tasks[2]
  const session = { id: '88888888-8888-4888-8888-888888888888', objective_id: objective, skill: task.mode, practice_task_id: task.id, practice_content_hash: 'sha256:practice', started_at: '2026-09-05T00:00:00Z', hints_shown: 0, answer_shown: false, hint_count: 0, result: undefined as object | undefined }
  let submitted: Record<string, unknown> | null = null
  let writes = 0
  let codeReads = 0
  let gradingWrites = 0
  let gradingJob: Record<string, unknown> | null = null
  const preview = { input_hash: 'sha256:input', request_hash: 'sha256:request', provider: 'fixture', model: 'fixture', request_bytes: 5000, input_byte_limit: 32000, max_output_tokens: 3000 }
  await page.route('**/api/v1/mastery/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path.endsWith('/due')) return route.fulfill({ json: { code: 200, data: { tasks: [{ objective_id: objective, chapter_id: 'chapter', title: '从零实现路由', skill: task.mode, due_at: '2026-09-01T00:00:00Z', reason: '写出最小实现' }] } } })
    if (path.endsWith('/practice-sessions') || path.endsWith(`/practice-sessions/${session.id}`)) return route.fulfill({ json: { code: 200, data: session } })
    if (path.endsWith(`/objectives/${objective}`)) return route.fulfill({ json: { code: 200, data: { objective: { id: objective, chapter_id: 'approved-revision:chapter:revision', title: '从零实现路由', behavior: task.prompt, required_skills: [task.mode], rubric: [], key_points: [], prerequisites: [], evidence_refs: task.evidence_ids, practice_revision_id: 'revision', practice_content_hash: 'sha256:practice', practice_projection: { format: 'inkwords.learning-projection.v2', chapter_id: 'chapter', revision_id: 'revision', content_hash: 'sha256:practice', practice_set: practiceSet } }, attempts: submitted ? [{ ...submitted, id: 'answer', learner_artifact_hash: 'sha256:learner' }] : [] } } })
    if (path.endsWith('/attempts') && route.request().method() === 'POST') {
      writes++; submitted = route.request().postDataJSON()
      session.result = { skill: 'explain', due_at: '2026-09-06T00:00:00Z', reason: '回顾实现思路' }
      return route.fulfill({ status: 200, contentType: 'application/json', body: '{"code":200,"data":' })
    }
    if (path.endsWith('/attempts/answer/code')) {
      codeReads++
      return route.fulfill({ json: { code: 200, data: { format: 'inkwords.learner-artifact.v1', objective_id: objective, attempt_id: 'answer', snapshot_hash: 'sha256:learner', submitted_at: '2026-09-05T00:01:00Z', files: submitted?.code_files } } })
    }
    if (path.endsWith('/attempts/answer/assessment')) return route.fulfill({ json: { code: 200, data: gradingJob } })
    if (path.endsWith('/attempts/answer/verification')) return route.fulfill({ json: { code: 200, data: null } })
    if (path.endsWith('/attempts/answer/verification-preview')) return route.fulfill({ json: { code: 200, data: { capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: false, profile: 'inkwords.learner-go-test-offline.v2', reason: '本夹具未启动隔离运行器' }, requires_explicit_start: true } } })
    if (path.endsWith('/attempts/answer/assessment-preview')) return route.fulfill({ json: { code: 200, data: preview } })
    if (path.endsWith('/attempts/answer/assessments') && route.request().method() === 'POST') {
      gradingWrites++
      const finding = { text: '检查查找失败时的返回值。', evidence_ids: task.evidence_ids }
      const feedback = { criteria: task.rubric.map((criterion) => ({ id: criterion.id, score: criterion.requires_runtime ? null : 2, reason: criterion.requires_runtime ? '尚无本次执行证据' : '原文件仅有包声明，缺少查找实现', answer_path: 'router.go', answer_quote: 'package main', evidence_ids: task.evidence_ids })), correct_points: [], missing_points: [finding], misconceptions: [], next_hint: finding, remediation: [finding] }
      gradingJob = { id: 'grade-code', objective_id: objective, attempt_id: 'answer', status: 'succeeded', preview, input: { answer: submitted?.answer, rubric: task.rubric, evidence: [{ id: task.evidence_ids[0], kind: 'source', excerpt: '按请求方法与路径查找。' }], learner_artifact: { files: submitted?.code_files } }, result: { origin: 'automated', provider_calls: 1, latency_millis: 10, usage: { known: false }, feedback }, effective_feedback: feedback, effective_hash: 'sha256:effective', corrections: [] }
      return route.fulfill({ json: { code: 200, data: gradingJob } })
    }
    return route.fallback()
  })
  await page.goto('/')
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await page.getByText('随作答保存代码（可选）', { exact: true }).click()
  const code = 'package main\r\n// 我写的路由实现 <script>不执行</script>\r\n'
  await page.getByLabel('选择本次代码文件').setInputFiles([{ name: 'router.go', mimeType: 'text/plain', buffer: Buffer.from(code) }])
  await expect(page.getByText('router.go', { exact: true })).toBeVisible()
  await page.getByLabel('本次作答', { exact: true }).fill('我先实现按路径查找，再补失败分支。')
  await page.getByLabel('未达到', { exact: true }).check()
  await page.getByRole('button', { name: '记录本次表现' }).click()
  await expect(page.getByRole('alert')).toBeVisible()
  expect(submitted).toEqual(expect.objectContaining({ code_files: [{ path: 'router.go', content: code }], practice_session_id: session.id }))
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await expect(page.getByText(/作答保存时的安排/)).toBeVisible()
  expect(codeReads).toBe(0)
  await page.getByRole('region', { name: '已保存作答' }).locator('summary').first().click()
  const snapshot = page.getByRole('region', { name: '本次代码快照' })
  await expect(snapshot.getByText('router.go', { exact: true })).toBeVisible()
  await snapshot.getByText('router.go', { exact: true }).click()
  await expect(snapshot.locator('pre')).toContainText('// 我写的路由实现 <script>不执行</script>')
  expect(writes).toBe(1)
  await page.getByRole('button', { name: '准备模型评分', exact: true }).click()
  await expect(page.getByRole('button', { name: '调用模型评分一次' })).toBeVisible()
  expect(gradingWrites).toBe(0)
  await page.getByRole('button', { name: '调用模型评分一次' }).click()
  await expect(page.getByText('代码原文（router.go）：package main').first()).toBeVisible()
  await expect(page.getByText('尚无本次执行证据').first()).toBeVisible()
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await page.getByRole('region', { name: '已保存作答' }).locator('summary').first().click()
  await expect(page.getByText('代码原文（router.go）：package main').first()).toBeVisible()
  expect(gradingWrites).toBe(1)
  expect(writes).toBe(1)
  await snapshot.screenshot({ path: testInfo.outputPath('learner-code-snapshot.png') })
  await page.getByText('尚无本次执行证据').first().scrollIntoViewIfNeeded()
  await page.screenshot({ path: testInfo.outputPath('learner-code-grading.png') })
})
