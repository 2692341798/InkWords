import { expect, test as base, type Page } from '@playwright/test'

const envelope = (data: unknown) => ({ code: 200, data })

async function installDeterministicApi(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname

    if (path === '/api/v1/blogs') return route.fulfill({ json: envelope([]) })
    if (path === '/api/v1/review/pick') {
      return route.fulfill({
        json: envelope({
          note_path: 'wiki/e2e/recommendation.md', title: 'E2E 推荐文章', source_title: 'E2E 知识库',
          review_reason: '用于稳定验证首页复习入口。', estimated_minutes: 5,
          available_modes: ['light_recall', 'detailed_qa'],
        }),
      })
    }
    if (path === '/api/v1/review/history') return route.fulfill({ json: envelope({ items: [], limit: 3 }) })
    if (path === '/api/v1/review/notes') return route.fulfill({ json: envelope({ items: [], total: 0, page: 1, page_size: 20 }) })
    if (path === '/api/v1/user/stats') {
      return route.fulfill({ json: envelope({ tokens_used: 0, estimated_cost: 0, total_articles: 0, total_words: 0, tech_stack_stats: [] }) })
    }
    if (path === '/api/v1/user/profile') {
      return route.fulfill({ json: envelope({ username: 'E2E 用户', email: 'e2e@example.com', avatar_url: '', subscription_tier: 0, token_limit: 0 }) })
    }

    return route.fulfill({ status: 404, json: { code: 404, message: `Unhandled E2E route: ${request.method()} ${path}` } })
  })
}

type AppFixtures = { appPage: Page }

export const test = base.extend<AppFixtures>({
  appPage: async ({ page }, runTest, testInfo) => {
    const errors: string[] = []
    page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()) })
    page.on('pageerror', (error) => errors.push(error.message))

    await installDeterministicApi(page)
    await page.goto(`/?token=e2e-${process.env.E2E_RUN_ID || 'local'}`)
    await expect(page.locator('#root')).not.toBeEmpty()
    await runTest(page)

    expect(errors, `Unexpected browser errors in ${testInfo.title}`).toEqual([])
  },
})

export { expect } from '@playwright/test'
