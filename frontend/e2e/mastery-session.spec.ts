import { expect, test } from './fixtures/app'
import { practiceSet } from './fixtures/practice'
import { installAssessmentFixture } from './fixtures/assessment'

test('@core @mastery-session recovers learner work and preserves explicit grading corrections', async ({ appPage: page }, testInfo) => {
  const observedAttempts: unknown[] = []
  let session = { id: '88888888-8888-8888-8888-888888888888', objective_id: '99999999-9999-9999-9999-999999999999', skill: 'explain', practice_task_id: practiceSet.tasks[0].id, practice_content_hash: 'sha256:fixture', started_at: '2026-09-05T00:00:00Z', hints_shown: 0, answer_shown: false, hint_count: 0, result: undefined as { skill: string; due_at: string; reason: string } | undefined }
  let sessionStarts = 0
  await page.route('**/api/v1/textbook-projects/chapters/chapter-1/projections', (route) => route.fulfill({ json: { code: 200, data: { learning: { format: 'inkwords.learning-projection.v2', chapter_id: 'chapter-1', revision_id: 'revision-1', practice_set: practiceSet } } } }))
  await page.route('**/api/v1/mastery/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path.endsWith('/practice-sessions')) {
      sessionStarts++
      return route.fulfill({ json: { code: 200, data: session } })
    }
    if (path.endsWith(`/practice-sessions/${session.id}`)) return route.fulfill({ json: { code: 200, data: session } })
    if (path.endsWith(`/practice-sessions/${session.id}/help`)) {
      const help = request.postDataJSON()
      session = { ...session, hints_shown: help.kind === 'hint' ? help.level : session.hints_shown, answer_shown: help.kind === 'answer' || session.answer_shown, hint_count: session.hint_count + 1 }
      return route.fulfill({ json: { code: 200, data: session } })
    }
    if (request.method() === 'GET' && path === '/api/v1/mastery/due') {
      return route.fulfill({ json: { code: 200, data: { tasks: [{ objective_id: '99999999-9999-9999-9999-999999999999', chapter_id: 'chapter-1', title: '解释 Gin 路由登记', skill: 'explain', due_at: '2026-09-03T00:00:00Z', reason: '先建立因果链。' }] } } })
    }
    if (request.method() === 'GET' && path === '/api/v1/mastery/objectives/99999999-9999-9999-9999-999999999999') {
      return route.fulfill({ json: { code: 200, data: { objective: { id: '99999999-9999-9999-9999-999999999999', chapter_id: 'approved-revision:chapter-1:revision-1', practice_revision_id: 'revision-1', practice_content_hash: 'sha256:fixture', practice_projection: { format: 'inkwords.learning-projection.v2', chapter_id: 'chapter-1', revision_id: 'revision-1', content_hash: 'sha256:fixture', practice_set: practiceSet }, title: '解释 Gin 路由登记', behavior: '解释方法和路径如何共同选择处理函数。', required_skills: ['explain'], rubric: ['解释注册与请求匹配的区别'], key_points: ['先选择方法树，再查找路径'], prerequisites: [], evidence_refs: ['evidence:gin-routergroup-get'] }, attempts: observedAttempts.map((attempt, index) => ({ ...(attempt as object), id: `attempt-${index}` })) } } })
    }
    if (request.method() === 'POST' && path === '/api/v1/mastery/objectives/99999999-9999-9999-9999-999999999999/attempts') {
      observedAttempts.push(request.postDataJSON())
      session.result = { skill: 'diagnose', due_at: '2026-09-04T00:00:00Z', reason: '需要练习定位根因。' }
      // Simulate a committed response truncated in transit, without suppressing
      // the fixture's checks for unexpected browser errors.
      return route.fulfill({ status: 200, contentType: 'application/json', body: '{"code":200,"data":' })
    }
    return route.fulfill({ status: 404, json: { code: 404, message: `Unhandled mastery route: ${request.method()} ${path}` } })
  })

  const assessments = await installAssessmentFixture(page, session.objective_id, practiceSet.tasks[0], '注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。')
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await expect(page.getByText(practiceSet.tasks[0].prompt)).toBeVisible()
  const initialStarts = sessionStarts
  await page.getByRole('button', { name: '查看第 1 层提示（记 1 次）' }).click()
  await expect(page.getByText('提示 1：先区分服务启动前后两个时间点。')).toBeVisible()
  await expect(page.getByText('提示 2：比较方法树和路径分段。')).toHaveCount(0)
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await expect(page.getByText('提示 1：先区分服务启动前后两个时间点。')).toBeVisible()
  await expect(page.getByLabel('提示次数')).toHaveValue('1')
  await page.getByLabel('本次作答', { exact: true }).fill('注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。')
  await page.getByRole('radio', { name: '达到', exact: true }).check()
  await page.getByLabel('独立完成，没有依赖完整答案').check()
  await page.getByLabel('提示次数').fill('1')
  await page.getByLabel('错误类别').fill('遗漏边界')
  await page.getByRole('button', { name: '记录本次表现' }).click()
  await expect(page.getByRole('alert')).toBeVisible()
  await expect(page.getByLabel('本次作答', { exact: true })).toHaveValue('注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。')
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await expect(page.getByText('作答保存时的安排：定位故障 · 需要练习定位根因。')).toBeVisible()
  await expect(page.getByLabel('本次作答', { exact: true })).toHaveValue('注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。')
  expect(sessionStarts).toBe(initialStarts)
  expect(observedAttempts).toEqual([expect.objectContaining({ practice_session_id: session.id, practice_task_id: practiceSet.tasks[0].id, practice_content_hash: 'sha256:fixture', skill: 'explain', answer: '注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。', correct: true, independent: true, hint_count: 1, error_kinds: ['遗漏边界'] })])
  const history = page.getByRole('region', { name: '已保存作答' })
  await history.locator('summary').click()
  await expect(history.getByText('注册时保存方法、路径与处理函数；请求到达时按方法和路径查找。', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '准备模型评分', exact: true })).toBeVisible()
  expect(assessments.calls).toBe(0)
  await page.getByRole('button', { name: '准备模型评分', exact: true }).click()
  await expect(page.getByRole('button', { name: '调用模型评分一次' })).toBeVisible()
  expect(assessments.calls).toBe(0)
  await page.getByRole('button', { name: '调用模型评分一次' }).click()
  await expect(page.getByText('自动评分建议', { exact: true })).toBeVisible()
  expect(assessments.calls).toBe(1)
  await page.getByRole('button', { name: '纠正此项' }).first().click()
  await page.getByLabel('更正分数').selectOption('4')
  await page.getByLabel('更正说明').fill('我的原作答包含注册与查找的明确区别。')
  await page.getByLabel(`引用 ${practiceSet.tasks[0].evidence_ids[0]}`, { exact: true }).uncheck()
  await page.getByLabel('引用 fixture-method-tree', { exact: true }).check()
  await page.getByRole('group', { name: '纠正引用依据' }).getByText('查看依据原文').last().click()
  await expect(page.getByText('本次冻结补充片段：先按 HTTP 方法选择对应路由树，再按路径查找处理函数。', { exact: true }).first()).toBeVisible()
  await page.getByRole('group', { name: '纠正引用依据' }).screenshot({ path: testInfo.outputPath('assessment-correction-evidence.png') })
  await page.getByRole('button', { name: '保存纠正' }).click()
  await expect(page.getByText('自动评分建议 · 已追加 1 次用户纠正')).toBeVisible()
  expect(assessments.correctionEvidence).toEqual([['fixture-method-tree']])
  expect(assessments.applications).toBe(0)
  await page.getByRole('button', { name: '应用当前评分并更新复习安排' }).click()
  await expect(page.getByText(/当前评分已应用/)).toBeVisible()
  expect(assessments.applications).toBe(1)
  await expect(page.getByText('依据：fixture-method-tree', { exact: true })).toBeVisible()
  await page.getByText('本项引用原文', { exact: true }).first().click()
  await expect(page.getByText('本次冻结补充片段：先按 HTTP 方法选择对应路由树，再按路径查找处理函数。', { exact: true }).first()).toBeVisible()
  await page.reload()
  await page.getByRole('button', { name: '开始本次练习' }).click()
  await history.locator('summary').first().click()
  await expect(page.getByText('自动评分建议 · 已追加 1 次用户纠正')).toBeVisible()
  expect(assessments.calls).toBe(1)
  await expect(page.getByText('依据：fixture-method-tree', { exact: true })).toBeVisible()
  await expect(page.getByText(/当前评分已应用/)).toBeVisible()
  expect(assessments.applications).toBe(1)
  await page.setViewportSize({ width: 1280, height: 1600 })
  await page.getByRole('region', { name: '掌握练习会话' }).screenshot({ path: testInfo.outputPath('mastery-answer-history.png') })
  await page.setViewportSize({ width: 390, height: 1800 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.getByRole('region', { name: '掌握练习会话' }).screenshot({ path: testInfo.outputPath('mastery-answer-history-mobile.png') })
  await history.scrollIntoViewIfNeeded()
  await page.screenshot({ path: testInfo.outputPath('mastery-recovered-result-mobile.png') })
  await page.getByText('自动评分建议 · 已追加 1 次用户纠正').scrollIntoViewIfNeeded()
  await page.screenshot({ path: testInfo.outputPath('mastery-assessment-mobile.png') })
  await page.setViewportSize({ width: 1280, height: 1400 })
  await page.getByText('自动评分建议 · 已追加 1 次用户纠正').scrollIntoViewIfNeeded()
  await page.screenshot({ path: testInfo.outputPath('mastery-assessment-desktop.png') })
  await page.getByRole('region', { name: '评分与复习安排' }).screenshot({ path: testInfo.outputPath('mastery-assessment-application.png') })
})
