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
        onStartQuestionRecommendation={vi.fn()}
        onOpenPicker={vi.fn()}
      />,
    )

    expect(html).toContain('随机抽一篇')
    expect(html).toContain('用这篇开始')
    expect(html).toContain('提问开始')
    expect(html).toContain('再抽一篇')
    expect(html).not.toContain('推荐一篇')
    expect(html).not.toContain('开始今日复习')
    expect(html).toContain('选择文章复习')
  })

  it('disables the question-start action when the recommendation card does not support detailed qa', () => {
    const html = renderToStaticMarkup(
      <ReviewEntryCards
        recommendationCard={{
          note_path: 'wiki/concepts/light-only.md',
          title: '仅支持轻提示',
          source_title: '知识库',
          review_reason: '这篇题卡目前只支持轻提示复述',
          estimated_minutes: 4,
          available_modes: ['light_recall'],
        }}
        isLoadingRecommendation={false}
        onRefreshRecommendation={vi.fn()}
        onStartRecommendation={vi.fn()}
        onStartQuestionRecommendation={vi.fn()}
        onOpenPicker={vi.fn()}
      />,
    )

    expect(html).toMatch(/<button[^>]*disabled=""[^>]*>提问开始<\/button>/)
  })
})
