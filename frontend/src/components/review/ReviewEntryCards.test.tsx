import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ReviewEntryCards } from './ReviewEntryCards'

describe('ReviewEntryCards', () => {
  it('renders one random-pick card plus the manual picker', () => {
    const html = renderToStaticMarkup(
      <ReviewEntryCards
        recommendationCard={{
          note_path: 'wiki/concepts/random.md',
          title: '随机文章',
          source_title: '知识库',
          review_reason: '从随机文章开始复习',
          estimated_minutes: 6,
          available_modes: ['light_recall', 'detailed_qa'],
        }}
        isLoadingRecommendation={false}
        onRefreshRecommendation={vi.fn()}
        onStartRecommendation={vi.fn()}
        onOpenPicker={vi.fn()}
      />,
    )

    expect(html).toContain('随机抽一篇')
    expect(html).toContain('用这篇开始')
    expect(html).toContain('再抽一篇')
    expect(html).not.toContain('推荐一篇')
    expect(html).not.toContain('开始今日复习')
    expect(html).toContain('选择文章复习')
  })
})
