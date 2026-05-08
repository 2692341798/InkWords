import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, test } from 'vitest'

const readIndexCss = () => {
  const indexCssPath = path.resolve(__dirname, './index.css')
  return fs.readFileSync(indexCssPath, 'utf-8')
}

describe('index.css markdown prose styles', () => {
  test('keeps spacing between h2 and h3 (no glued headings)', () => {
    const css = readIndexCss()
    expect(css).toContain('div.prose h2 + h3')
  })

  test('improves markdown table usability', () => {
    const css = readIndexCss()
    expect(css).toContain('div.prose table')
    expect(css).toMatch(/overflow-x-auto|overflow-x:\s*auto/)
    expect(css).toContain('div.prose th')
    expect(css).toContain('div.prose td')
  })
})
