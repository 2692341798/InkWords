import { expect, test } from '@playwright/test'

const realMode = process.env.E2E_EXTERNAL_MODE === 'real'

const requireTestToken = () => {
  const token = process.env.E2E_TEST_TOKEN
  if (!token) throw new Error('E2E_TEST_TOKEN is required for protected real acceptance')
  return token
}

test('@deepseek real generation entry is available', async ({ page }) => {
  test.skip(!realMode, 'Set E2E_EXTERNAL_MODE=real for protected acceptance')
  await page.goto(`/?token=${encodeURIComponent(requireTestToken())}`)
  await page.getByRole('button', { name: /进入博客生成/ }).click()
  await expect(page.getByText(/来源|仓库|文档/).first()).toBeVisible()
})

test('@obsidian real review notes can be listed', async ({ page }) => {
  test.skip(!realMode, 'Set E2E_EXTERNAL_MODE=real for protected acceptance')
  await page.goto(`/?token=${encodeURIComponent(requireTestToken())}`)
  await page.getByRole('button', { name: '知识复习' }).first().click()
  await page.getByRole('button', { name: /手动/ }).first().click()
  await expect(page.getByRole('heading', { name: '选择文章复习' })).toBeVisible()
})

test('@oauth GitHub OAuth entry redirects to the gateway', async ({ page }) => {
  test.skip(!realMode, 'Set E2E_EXTERNAL_MODE=real for protected acceptance')
  await page.goto('/')
  await page.getByRole('button', { name: /使用 GitHub 登录/ }).click()
  await expect(page).toHaveURL(/github\.com|\/api\/v1\/auth\/oauth\/github/)
})
