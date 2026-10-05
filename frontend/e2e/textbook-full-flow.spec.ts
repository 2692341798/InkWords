import { expect, test } from './fixtures/app'

const projectID = '11111111-1111-1111-1111-111111111111'
const buildID = '55555555-5555-5555-5555-555555555555'
const stages = ['developmental', 'technical', 'self_study', 'consistency', 'copy_editing', 'layout', 'rights', 'reader_trial'] as const
const envelope = (data: unknown) => ({ code: 0, data })

test('@core @textbook-full-flow records human publication evidence and requires explicit promotion', async ({ appPage: page }) => {
  let status: 'ready_for_review' | 'publication_candidate' = 'ready_for_review'
  const rightsItems: Array<Record<string, unknown>> = []
  const humanReviews: Array<Record<string, unknown>> = []
  const project = { id: projectID, title: 'Gin 出版编辑验收', audience_level: 'foundation', status: 'approved', revision_version: 3, updated_at: '2026-09-04T00:00:00Z' }
  const build = () => ({
    id: buildID, project_id: projectID, book_contract_revision_id: 'contract-1', style_sheet_revision_id: 'style-1',
    approved_revision_ids: ['revision-1'], tool_versions_json: { canonical_ast: 'inkwords.canonical-book.v1' },
    manifest_json: { format: 'inkwords.book-build.v1' }, manifest_hash: `sha256:${'a'.repeat(64)}`,
    status, blockers_json: status === 'publication_candidate' ? [] : ['等待出版编辑工作台的显式预检。'], created_at: '2026-09-04T00:00:00Z',
  })
  const editorial = () => {
    const blockers = [
      ...(rightsItems.length === 0 ? ['缺少权利清单。'] : []),
      ...stages.filter((stage) => !humanReviews.some((review) => review.stage === stage)).map((stage) => `缺少人工 ${stage} 审校记录。`),
    ]
    return {
      build: build(), required_rights_subjects: [{ subject_ref: 'chapter-revision:revision-1', work_type: 'prose' }], rights_items: rightsItems, human_reviews: humanReviews,
      automated_checks: [
        { id: 'frozen-manifest-identity', detector: 'core-api-book-build-v1', status: 'pass' },
        { id: 'approved-revisions-frozen', detector: 'core-api-book-build-v1', status: 'pass' },
      ],
      preflight: { passed: blockers.length === 0, blockers },
    }
  }

  await page.route('**/api/v1/textbook-projects**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (request.method() === 'GET' && path === '/api/v1/textbook-projects') return route.fulfill({ json: envelope([project]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/workspace`) return route.fulfill({ json: envelope({ project, sources: [], chapters: [], latest_book_build: build() }) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/source-library`) return route.fulfill({ json: envelope([]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/source-evidence`) return route.fulfill({ json: envelope([]) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/${projectID}/progress`) return route.fulfill({ json: envelope({ stages: [{ key: 'publication', status: status === 'publication_candidate' ? 'approved' : 'in_progress', reason: '由服务端冻结构建状态派生。' }] }) })
    if (request.method() === 'GET' && path === `/api/v1/textbook-projects/book-builds/${buildID}/editorial`) return route.fulfill({ json: envelope(editorial()) })
    if (request.method() === 'POST' && path === `/api/v1/textbook-projects/book-builds/${buildID}/rights`) {
      const input = request.postDataJSON()
      rightsItems.push({ id: 'rights-1', build_id: buildID, project_id: projectID, ...input, created_at: '2026-09-04T00:01:00Z' })
      return route.fulfill({ status: 201, json: envelope(rightsItems[0]) })
    }
    if (request.method() === 'POST' && path === `/api/v1/textbook-projects/book-builds/${buildID}/reviews`) {
      const input = request.postDataJSON()
      humanReviews.push({ id: `review-${humanReviews.length + 1}`, build_id: buildID, ...input, automated: false, completed_at: '2026-09-04T00:02:00Z', created_at: '2026-09-04T00:02:00Z' })
      return route.fulfill({ status: 201, json: envelope(humanReviews.at(-1)) })
    }
    if (request.method() === 'POST' && path === `/api/v1/textbook-projects/book-builds/${buildID}/publication-candidate`) {
      if (!editorial().preflight.passed) return route.fulfill({ status: 400, json: { code: 'INVALID_STATE', data: null, message: '出版证据尚未齐全。' } })
      status = 'publication_candidate'
      return route.fulfill({ json: envelope(build()) })
    }
    return route.fulfill({ status: 404, json: { code: 404, message: `Unhandled textbook route: ${request.method()} ${path}` } })
  })

  await page.reload()
  await page.getByRole('button', { name: '打开工作台' }).click()
  await expect(page.getByRole('heading', { name: '以冻结构建为单位记录审阅证据' })).toBeVisible()
  await expect(page.getByRole('button', { name: '显式标记为出版候选' })).toBeDisabled()

  await page.getByRole('button', { name: '登记此对象' }).click()
  await expect(page.getByRole('textbox', { name: '作品引用' })).toHaveValue('chapter-revision:revision-1')
  await page.getByRole('combobox', { name: '出版状态' }).selectOption('ready')
  await page.getByRole('textbox', { name: '权利依据' }).fill('作者原创并已逐项核对引用范围')
  await page.getByRole('textbox', { name: '允许用途' }).fill('本地教材编辑、导出与送审')
  await page.getByRole('textbox', { name: '署名要求' }).fill('InkWords 本地作者')
  await page.getByRole('button', { name: '登记不可变权利项' }).click()
  await expect(page.getByText('chapter-revision:revision-1').last()).toBeVisible()

  await page.getByRole('textbox', { name: '人工验收者' }).fill('人工验收者')
  for (let index = 0; index < stages.length; index += 1) {
    await page.getByRole('textbox', { name: '审校记录（至少 8 个字符）' }).fill(`第 ${index + 1} 阶段已按冻结构建逐项人工检查并记录结论。`)
    await page.getByRole('button', { name: '确认这项人工审校已完成' }).click()
    await expect(page.getByText('已记录')).toHaveCount(index + 1)
  }

  await expect(page.getByText('预检证据齐全')).toBeVisible()
  await expect(page.getByRole('button', { name: '显式标记为出版候选' })).toBeEnabled()
  await page.getByRole('button', { name: '显式标记为出版候选' }).click()
  await expect(page.getByText('系统出版预检通过')).toBeVisible()
  await expect(page.getByText(/不代表出版社批准/)).toBeVisible()
  await expect(page.getByRole('link', { name: '下载审校包 ZIP' })).toHaveAttribute('href', `/api/v1/textbook-projects/book-builds/${buildID}/export/review-bundle`)

  await page.reload()
  await page.getByRole('button', { name: '打开工作台' }).click()
  await expect(page.getByText('系统出版预检通过')).toBeVisible()
  await expect(page.getByText('该构建已进入出版候选，权利证据已冻结。')).toBeVisible()
})
