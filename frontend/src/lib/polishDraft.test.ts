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

  test('会剔除正文开头的对话式前言与角色自述', () => {
    const input = `## 标题建议
1. A

---

好的，收到你的需求。作为高级全栈架构师和技术博主，我将根据你提供的课程源内容和大纲，撰写一篇高质量博客。

# Python 基础语法速通

这里是正文。`

    expect(extractPolishedBody(input)).toBe(`# Python 基础语法速通

这里是正文。`)
  })
})
