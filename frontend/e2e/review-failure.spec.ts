import { expect, test } from '@playwright/test'

test('@core review bootstrap failure is bounded and manually retryable', async ({ page }) => {
  let pickRequests = 0
  let historyRequests = 0
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/v1/blogs') return route.fulfill({ json: { code: 200, data: [] } })
    if (path === '/api/v1/review/pick') pickRequests += 1
    if (path === '/api/v1/review/history') historyRequests += 1
    return route.fulfill({ status: 502, json: { message: 'Knowledge bridge unavailable' } })
  })

  await page.goto('/?token=e2e-review-failure')
  await expect(page.getByRole('alert')).toContainText('复习摘要暂时加载失败')
  await page.waitForTimeout(500)
  expect({ pickRequests, historyRequests }).toEqual({ pickRequests: 1, historyRequests: 1 })

  await page.getByRole('button', { name: '重试加载' }).click()
  await expect.poll(() => ({ pickRequests, historyRequests })).toEqual({ pickRequests: 2, historyRequests: 2 })
})
