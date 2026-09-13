import { expect, test } from './fixtures/app'

test('@core @cross-browser renders the workspace and navigates primary views', async ({ appPage: page }) => {
  await expect(page.getByText('墨言博客助手').first()).toBeVisible()
  await expect(page.getByRole('heading', { name: '教材项目' })).toBeVisible()

  await page.getByRole('button', { name: '工作入口' }).click()
  await expect(page.getByRole('heading', { name: '从资料到教材，从教材到掌握' })).toBeVisible()

  await page.getByRole('button', { name: '知识复习' }).first().click()
  await expect(page.getByRole('heading', { name: '选一篇，读完，再用自己的话讲出来' })).toBeVisible()

  await page.getByRole('button', { name: '教材项目' }).click()
  await expect(page.getByRole('heading', { name: '教材项目' })).toBeVisible()
})

test('@core switches the home path and exposes the matching next action', async ({ appPage: page }) => {
  await page.getByRole('button', { name: '工作入口' }).click()
  const reviewChoice = page.getByRole('button', { name: /知识复习.*内化/ })
  await reviewChoice.click()
  await expect(reviewChoice).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('button', { name: /进入知识复习/ }).last()).toBeVisible()
})
