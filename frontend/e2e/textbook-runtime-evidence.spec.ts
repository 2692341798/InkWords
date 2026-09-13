import { practiceSet } from './fixtures/practice'
import { expect, test } from './fixtures/app'

const projectID = '55555555-5555-5555-5555-555555555555'
const chapterID = '66666666-6666-6666-6666-666666666666'
const revisionID = '77777777-7777-7777-7777-777777777777'
const artifactID = '88888888-8888-8888-8888-888888888888'
const envelope = (data: unknown) => ({ code: 0, data })

test('@core @runtime-evidence keeps automatic observations distinct from required IDE capture', async ({ appPage: page }) => {
  let objectiveCreated = false
  const project = {
    id: projectID, title: '运行证据边界', audience_level: 'programming', status: 'approved', revision_version: 1, updated_at: '2026-09-03T00:00:00Z',
  }
  const chapter = {
    id: chapterID, project_id: projectID, sort_order: 1, title: '验证一条 Gin 路由', chapter_profile: 'hands_on', status: 'approved', revision_version: 1, current_revision_id: revisionID, approved_revision_id: revisionID, updated_at: '2026-09-03T00:00:00Z',
  }

  await page.route('**/api/v1/textbook-projects**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (request.method() === 'GET' && path === '/api/v1/textbook-projects') return route.fulfill({ json: envelope([project]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/workspace`) return route.fulfill({ json: envelope({ project, sources: [], chapters: [chapter] }) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/source-library`) return route.fulfill({ json: envelope([]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/source-evidence`) return route.fulfill({ json: envelope([]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/progress`) return route.fulfill({ json: envelope({ stages: [] }) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/chapters/${chapterID}/code-artifacts/${artifactID}/verify`) return route.fulfill({ json: envelope(null) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/chapters/${chapterID}/projections`) return route.fulfill({ json: envelope({ learning: { format: 'inkwords.learning-projection.v2', practice_set: practiceSet, revision_id: revisionID, chapter_id: chapterID, content_hash: 'sha256:learning', learning_arc: { stages: [{ objective: '独立说明路由登记。', prerequisites: [], success_evidence: ['evidence:gin-routing'] }] }, objectives: [{ id: `${revisionID}:mastery`, chapter_id: chapterID, text: '完成路由登记的六维练习。', required_modes: ['explain', 'complete', 'reproduce', 'transfer', 'diagnose', 'retain'] }] } }) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/chapters/${chapterID}/workspace`) {
      return route.fulfill({ json: envelope({
        chapter,
        revisions: [{
          id: revisionID, chapter_id: chapterID, revision_number: 1, kind: 'approved', markdown: '# Gin 路由验证', document_json: {
            video_runbook: {
              format: 'inkwords.video-runbook.v1', stack: 'go', observation_goal: '调用链',
              recommendation: { primary: 'GoLand', alternative: 'VS Code + Go 扩展', reason: '用断点观察调用链。', manual_capture_required: true },
              steps: [], capture_checklist: ['在 IDE 中人工截取断点调用栈，并记录版本。'], manual_capture_pending: true, verification_status: 'unverified',
            },
          }, content_hash: 'a'.repeat(64), created_by: 'manual', created_at: '2026-09-03T00:00:00Z',
        }],
        code_artifacts: [{
          id: artifactID, revision_id: revisionID, kind: 'teaching_implementation', language: 'go', entrypoint: 'main.go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: ['仅验证教材最小实现。'], status: 'verified', created_at: '2026-09-03T00:00:00Z',
        }],
        runtime_evidence: [
          { id: 'terminal-evidence', revision_id: revisionID, code_artifact_id: artifactID, code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', toolchain_version: 'go1.25.4', tool_name: 'course-runner', tool_version: 'fixture', structured_output: '{"format":"inkwords.runtime-observation.v1","observations":{"commands":[{"exit_code":0}]},"interpretations":[]}', output_truncated: false, captured_at: '2026-09-03T00:00:00Z', expires_at: '2099-09-03T00:00:00Z', created_at: '2026-09-03T00:00:00Z' },
          { id: 'browser-evidence', revision_id: revisionID, code_artifact_id: artifactID, code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'browser_page', status: 'verified', toolchain_version: 'go1.25.4', tool_name: 'Playwright/Chromium', tool_version: 'Playwright 1.62.1; Chromium 151.0.7922.34; runner-image:sha256:runner', structured_output: '{"format":"inkwords.runtime-observation.v1","observations":{"browser_pages":[{"url":"http://127.0.0.1:8080/health","final_url":"http://127.0.0.1:8080/health","browser_name":"Chromium","browser_version":"151.0.7922.34","playwright_version":"1.62.1","screenshot_ref":"fixture:browser-evidence","dom_assertions":[{"locator":"main","assertion":"has_text","expected":"healthy"}],"console":[],"network":[{"url":"http://127.0.0.1:8080/health","resource_type":"document","status":200}]}]},"interpretations":[]}', output_truncated: false, captured_at: '2026-09-03T00:00:00Z', expires_at: '2099-09-03T00:00:00Z', created_at: '2026-09-03T00:00:00Z' },
          { id: 'ide-evidence', revision_id: revisionID, code_artifact_id: artifactID, code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'ide_capture', status: 'unverified', output_truncated: false, created_at: '2026-09-03T00:00:00Z' },
        ],
        assets: [{
          id: 'browser-asset', revision_id: revisionID, evidence_id: 'browser-evidence', stable_ref: 'inkwords-asset:browser-evidence', kind: 'screenshot', content_hash: 'sha256:asset', alt_text: '受控教学页面', source: 'Bubblewrap 内 Playwright', generation_method: 'isolated_playwright_probe', visual_purpose: 'rendered_ui', rights_status: 'pending', status: 'verified', created_at: '2026-09-03T00:00:00Z',
        }],
      }) })
    }
    return route.fulfill({ status: 404, json: { code: 404, message: `Unhandled runtime-evidence route: ${request.method()} ${path}` } })
  })
  await page.route('**/api/v1/mastery/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (request.method() === 'POST' && path === '/api/v1/mastery/objectives') {
      objectiveCreated = true
      expect(request.postDataJSON()).toMatchObject({ chapter_id: `approved-revision:${chapterID}:${revisionID}`, skills: ['explain', 'complete', 'reproduce', 'transfer', 'diagnose', 'retain'] })
      return route.fulfill({ json: { code: 200, data: { id: 'learning-objective' } } })
    }
    if (request.method() === 'GET' && path === '/api/v1/mastery/due') return route.fulfill({ json: { code: 200, data: { tasks: objectiveCreated ? [{ objective_id: 'learning-objective', chapter_id: chapterID, title: '完成路由登记的六维练习。', skill: 'explain', due_at: '2026-09-03T00:00:00Z', reason: '新学习目标先从首个要求的练习开始。' }] : [] } } })
    if (request.method() === 'GET' && path === '/api/v1/mastery/objectives/learning-objective') return route.fulfill({ json: { code: 200, data: { objective: { id: 'learning-objective', chapter_id: chapterID, title: '完成路由登记的六维练习。', behavior: '独立说明路由登记。', required_skills: ['explain'], rubric: ['解释方法与路径'], key_points: ['注册与匹配'], prerequisites: [], evidence_refs: ['evidence:gin-routing'] }, attempts: [] } } })
    return route.fulfill({ status: 404, json: { code: 404, message: `Unhandled mastery route: ${request.method()} ${path}` } })
  })

  await page.reload()
  await page.getByRole('button', { name: '打开工作台' }).click()
  await page.getByRole('button', { name: '编辑' }).click()

  await expect(page.getByRole('heading', { name: '验证一条 Gin 路由' })).toBeVisible()
  await expect(page.getByRole('region', { name: '教学代码运行证据' }).getByText('终端输出')).toBeVisible()
  await expect(page.getByRole('region', { name: '教学代码运行证据' }).getByText('浏览器页面')).toBeVisible()
  await expect(page.getByRole('region', { name: '教学代码运行证据' }).getByText('IDE 人工截图')).toBeVisible()
  await expect(page.getByText('结构化观察：1 条工具结果；解释：未记录。').first()).toBeVisible()
  await expect(page.getByText('浏览器观察：Playwright 1.62.1；Chromium 151.0.7922.34；解释：未记录。')).toBeVisible()
  await expect(page.getByText('自动浏览器截图资产')).toBeVisible()
  await expect(page.getByText('待人工采集证据')).toBeVisible()
  await expect(page.getByText('在 IDE 中人工截取断点调用栈，并记录版本。')).toBeVisible()
  await page.getByRole('button', { name: '从批准教材创建学习任务' }).click()
  await expect(page.getByRole('region', { name: '掌握练习会话' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '完成路由登记的六维练习。', exact: true })).toBeVisible()
})
