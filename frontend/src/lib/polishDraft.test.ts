import { describe, expect, test } from 'vitest'
import { extractPolishedBody } from './polishDraft'

describe('extractPolishedBody', () => {
  test('存在分隔线时，返回分隔线后的正文并 trim', () => {
    const input = `## 标题建议
1. A

---

# 正文
内容
`

    expect(extractPolishedBody(input)).toBe(`# 正文
内容`)
  })

  test('找不到分隔线时，返回全文 trim', () => {
    expect(extractPolishedBody('  hello  ')).toBe('hello')
  })

  test('正文中包含更多分隔线时，不应截断', () => {
    const input = `x
---
y

---

z`

    expect(extractPolishedBody(input)).toBe(`y

---

z`)
  })
})

