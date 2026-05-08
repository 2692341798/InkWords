import { describe, expect, test } from 'vitest'
import { normalizeMarkdown } from './markdownNormalize'

describe('normalizeMarkdown', () => {
  test('adds a space after heading hashes', () => {
    expect(normalizeMarkdown('##2.printf函数')).toBe('## 2. printf函数')
  })

  test('moves fenced code block onto its own lines', () => {
    const input = '```cintmain(){\n  return 0\n}\n```'
    const output = normalizeMarkdown(input)
    expect(output).toBe('```c\nintmain(){\n  return 0\n}\n```')
  })

  test('splits multiple headings that are glued together', () => {
    const input = '# 标题##1.为什么需要输入和输出？'
    const output = normalizeMarkdown(input)
    expect(output).toBe('# 标题\n\n## 1. 为什么需要输入和输出？')
  })

  test('adds missing space after list dash for bold items', () => {
    const input = '-**输入**：内容'
    const output = normalizeMarkdown(input)
    expect(output).toBe('- **输入**：内容')
  })

  test('splits heading and table when glued by pipe', () => {
    const input = '### 2.3常用转义字符|转义字符|含义|示例效果|'
    const output = normalizeMarkdown(input)
    expect(output).toBe('### 2.3常用转义字符\n\n|转义字符|含义|示例效果|')
  })
})
