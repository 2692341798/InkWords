import { expect, test } from './fixtures/app'

test('@core @cross-browser renders the workspace and navigates primary views', async ({ appPage: page }) => {
  await expect(page.getByText('墨言博客助手').first()).toBeVisible()
  await expect(page.getByRole('heading', { name: '从资料到博客，从博客到复习' })).toBeVisible()

  await page.getByRole('button', { name: '知识复习' }).first().click()
  await expect(page.getByRole('heading', { name: '选一篇，读完，再用自己的话讲出来' })).toBeVisible()

  await page.getByRole('button', { name: '个人中心' }).click()
  await expect(page.getByRole('heading', { name: '个人中心' })).toBeVisible()
  await expect(page.getByText('E2E 用户')).toBeVisible()
})

test('@core switches the home path and exposes the matching next action', async ({ appPage: page }) => {
  const reviewChoice = page.getByRole('button', { name: /知识复习.*内化/ })
  await reviewChoice.click()
  await expect(reviewChoice).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('button', { name: /进入知识复习/ }).last()).toBeVisible()
})

test('@core mobile keeps primary navigation reachable', async ({ appPage: page }, testInfo) => {
  test.skip(!testInfo.project.name.startsWith('mobile'), 'Mobile-only layout assertion')
  await expect(page.getByRole('button', { name: '新建' })).toBeVisible()
  await page.getByRole('button', { name: '知识复习' }).first().click()
  await expect(page.getByText('知识漫游复习')).toBeVisible()
})
