import { expect, test } from '@playwright/test'

const realMode = process.env.E2E_EXTERNAL_MODE === 'real'

test('@local textbook workspace opens without login', async ({ page }) => {
  test.skip(!realMode, 'Set E2E_EXTERNAL_MODE=real for local acceptance')
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '教材项目' })).toBeVisible()
  await page.getByRole('button', { name: '工作入口' }).click()
  await expect(page.getByRole('heading', { name: '从资料到教材，从教材到掌握' })).toBeVisible()
  await expect(page.getByRole('button', { name: /项目精通课程/ })).toHaveCount(0)
})

test('@obsidian real review notes can be listed', async ({ page }) => {
  test.skip(!realMode, 'Set E2E_EXTERNAL_MODE=real for local acceptance')
  await page.goto('/')
  await page.getByRole('button', { name: '工作入口' }).click()
  await page.getByRole('button', { name: '知识复习' }).first().click()
  await page.getByRole('button', { name: /手动/ }).first().click()
  await expect(page.getByRole('heading', { name: '选择文章复习' })).toBeVisible()
})
