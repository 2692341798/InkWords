import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { ReviewSessionCard } from './ReviewSessionCard'

describe('ReviewSessionCard', () => {
  it('renders the session history inside a fixed-height scroll container', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: '五事七计',
          opening_prompt: '先讲主线',
          initial_hints: [],
          turn_index: 2,
          turns: [
            {
              turn_index: 1,
              role: 'system',
              turn_type: 'opening',
              content: '先别看原文',
            },
          ],
        }}
        selectedMode="light_recall"
        latestStageFeedback={null}
        latestHint={null}
        finalFeedback={null}
        onModeChange={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('data-slot="session-history-scroll"')
    expect(html).toContain('h-96')
    expect(html).toContain('overflow-y-scroll')
    expect(html).toContain('custom-scrollbar')
  })
})
