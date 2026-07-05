import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { ReviewSessionCard } from './ReviewSessionCard'

const session = {
  session_id: 'session-1', status: 'in_progress', mode: 'light_recall' as const,
  title: '五事七计', opening_prompt: '先讲主线', initial_hints: [],
  reading_content: '# 五事七计\n\n这是一段完整原文。',
  session_outline: { summary: '摘要', main_question: '它主要讲什么？', core_concepts: ['五事'], process_steps: [], application_cases: [], checkpoints: [] },
  turn_index: 1, turns: [],
}

const props = { selectedMode: 'light_recall' as const, latestStageFeedback: null, latestHint: null, finalFeedback: null, onModeChange: () => {}, onRespond: async () => {}, onRequestHint: async () => {}, onFinish: async () => {}, onCompleteReading: async () => {} }

describe('ReviewSessionCard', () => {
  it('renders only reading content and completion CTA during reading', () => {
    const html = renderToStaticMarkup(<ReviewSessionCard {...props} session={{ ...session, phase: 'reading' }} />)
    expect(html).toContain('沉浸阅读')
    expect(html).toContain('完整原文')
    expect(html).toContain('我已浏览完毕，开始复述')
    expect(html).not.toContain('<textarea')
    expect(html).toContain('aria-current="step"')
  })

  it('renders the focused recall editor only after reading completes', () => {
    const html = renderToStaticMarkup(<ReviewSessionCard {...props} session={{ ...session, phase: 'recalling' }} />)
    expect(html).toContain('关书复述')
    expect(html).toContain('<textarea')
    expect(html).toContain('我卡住了')
    expect(html).not.toContain('完整原文')
  })

  it('renders hit and missed points in coaching', () => {
    const html = renderToStaticMarkup(<ReviewSessionCard {...props} session={{ ...session, phase: 'coaching', latest_review_feedback: { judgement: '部分答对', hit_points: ['讲清主线'], missed_points: ['遗漏边界'], suggestion: '补充边界' } }} />)
    expect(html).toContain('已经讲清楚')
    expect(html).toContain('讲清主线')
    expect(html).toContain('遗漏边界')
    expect(html).toContain('继续补充')
  })
})
