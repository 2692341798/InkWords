import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { ReviewSessionCard } from './ReviewSessionCard'

describe('ReviewSessionCard', () => {
  it('renders round goal and structured review feedback', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'detailed_qa',
          title: '并发控制与速率限制',
          opening_prompt: '先说主线',
          initial_hints: [],
          session_outline: {
            summary: '并发控制与速率限制的摘要',
            main_question: '它主要解决什么问题？',
            core_concepts: ['并发控制', '速率限制'],
            process_steps: [],
            application_cases: [],
            checkpoints: ['并发控制限制同时执行的任务数量'],
          },
          current_round_goal: '先讲清楚这篇文章的主线问题',
          latest_review_feedback: {
            judgement: '部分答对',
            hit_points: ['答到了并发控制限制任务数量'],
            missed_points: ['没有提到速率限制控制请求频率'],
            suggestion: '下一轮补上速率限制和为什么需要它',
          },
          turn_index: 2,
          turns: [],
        }}
        selectedMode="detailed_qa"
        latestStageFeedback="你已经抓到主线。"
        latestHint={null}
        finalFeedback={null}
        onModeChange={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('本轮目标')
    expect(html).toContain('部分答对')
    expect(html).toContain('你答到的点')
    expect(html).toContain('你还漏掉的点')
  })

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
          session_outline: {
            summary: '五事七计摘要',
            main_question: '五事七计主要讲什么？',
            core_concepts: ['五事', '七计'],
            process_steps: [],
            application_cases: [],
            checkpoints: ['先理解五事七计的主线'],
          },
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
