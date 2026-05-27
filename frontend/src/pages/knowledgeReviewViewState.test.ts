import { describe, expect, it } from 'vitest'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { KnowledgeReview } from './KnowledgeReview'

describe('KnowledgeReview', () => {
  it('shows the three entry actions and recent history section', () => {
    const html = renderToStaticMarkup(createElement(KnowledgeReview))

    expect(html).toContain('开始今日复习')
    expect(html).toContain('随机抽一篇')
    expect(html).toContain('选择文章复习')
    expect(html).toContain('最近复习记录')
  })
})
