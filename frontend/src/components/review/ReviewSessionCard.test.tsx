import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'

import { ReviewSessionCard } from './ReviewSessionCard'

describe('ReviewSessionCard', () => {
  it('locks the mode after a session has started so the UI does not imply a live mode switch', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: 'PostgreSQL 数据库设计',
          opening_prompt: '先讲讲主线。',
          initial_hints: ['先说明问题'],
          next_question: '它的关键步骤是什么？',
          turn_index: 2,
          turns: [],
        }}
        selectedMode="detailed_qa"
        latestStageFeedback={null}
        latestHint={null}
        finalFeedback={null}
        onModeChange={vi.fn()}
        onRespond={vi.fn()}
        onRequestHint={vi.fn()}
        onFinish={vi.fn()}
      />,
    )

    expect(html).toContain('当前主题：PostgreSQL 数据库设计 · 模式：轻提示复述')
    expect(html).toContain('会话开始后模式已锁定')
  })
})

