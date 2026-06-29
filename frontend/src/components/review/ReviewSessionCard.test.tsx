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
          ready_to_answer: true,
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
        onStartAnswering={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('当前目标')
    expect(html).toContain('部分答对')
    expect(html).toContain('答到的点')
    expect(html).toContain('还需补位')
  })

  it('renders the session history inside a fixed-height scroll container', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: '五事七计',
          ready_to_answer: true,
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
        onStartAnswering={() => {}}
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

  it('renders source preview first and hides the answer textarea until review starts', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'created',
          mode: 'light_recall',
          title: '并发控制与速率限制',
          source_title: 'InkWords 内容生成平台架构解析系列',
          source_preview: '第一段原文。第二段原文。',
          ready_to_answer: false,
          opening_prompt: '先看原文再复述',
          initial_hints: [],
          session_outline: {
            summary: '并发控制摘要',
            main_question: '它主要解决什么问题？',
            core_concepts: [],
            process_steps: [],
            application_cases: [],
            checkpoints: [],
          },
          turn_index: 1,
          turns: [],
        }}
        selectedMode="light_recall"
        latestStageFeedback={null}
        latestHint={null}
        finalFeedback={null}
        onModeChange={() => {}}
        onStartAnswering={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('原文预览')
    expect(html).toContain('开始复述')
    expect(html).toContain('第一段原文。第二段原文。')
    expect(html).not.toContain('用自己的话讲一遍')
  })

  it('renders a reading workspace with a smart hint panel when answering', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: '并发控制与速率限制',
          source_title: 'InkWords 内容生成平台架构解析系列',
          source_preview: '第一段原文。第二段原文。',
          ready_to_answer: true,
          opening_prompt: '先说主线',
          initial_hints: [],
          session_outline: {
            summary: '并发控制摘要',
            main_question: '它主要解决什么问题？',
            core_concepts: ['并发控制', '速率限制'],
            process_steps: ['先限制并发', '再控制请求频率'],
            application_cases: ['保护 API'],
            checkpoints: ['先理解并发控制', '再理解速率限制'],
          },
          current_round_goal: '先讲清楚并发控制为什么必要',
          next_question: '如果讲给新手，你会先从哪一句开始？',
          latest_review_feedback: {
            judgement: '部分答对',
            hit_points: ['提到了并发控制'],
            missed_points: ['没有提到速率限制'],
            suggestion: '补上速率限制和保护 API 的作用',
          },
          turn_index: 3,
          turns: [],
        }}
        selectedMode="light_recall"
        latestStageFeedback="你已经抓住了方向，但还需要把速率限制补进来。"
        latestHint="先从“为什么不能让所有请求同时冲进来”开始讲。"
        finalFeedback={null}
        onModeChange={() => {}}
        onStartAnswering={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('data-slot="review-reading-workspace"')
    expect(html).toContain('原文抽屉')
    expect(html).toContain('智能提示')
    expect(html).toContain('当前问题')
    expect(html).toContain('下一步动作')
  })

  it('renders a source drawer trigger and a scrollable source panel', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: '并发控制与速率限制',
          source_title: 'InkWords 内容生成平台架构解析系列',
          source_preview: '第一段原文。第二段原文。',
          ready_to_answer: true,
          opening_prompt: '先说主线',
          initial_hints: [],
          session_outline: {
            summary: '并发控制摘要',
            main_question: '它主要解决什么问题？',
            core_concepts: ['并发控制'],
            process_steps: [],
            application_cases: [],
            checkpoints: [],
          },
          turn_index: 3,
          turns: [],
        }}
        selectedMode="light_recall"
        latestStageFeedback={null}
        latestHint={null}
        finalFeedback={null}
        onModeChange={() => {}}
        onStartAnswering={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('查看原文')
    expect(html).toContain('data-slot="source-drawer-scroll"')
    expect(html).toContain('overflow-y-auto')
  })

  it('renders a reading-first workspace instead of a dark training hero', () => {
    const html = renderToStaticMarkup(
      <ReviewSessionCard
        session={{
          session_id: 'session-1',
          status: 'in_progress',
          mode: 'light_recall',
          title: '并发控制与速率限制',
          source_title: 'InkWords 内容生成平台架构解析系列',
          source_preview: '第一段原文。第二段原文。',
          ready_to_answer: true,
          opening_prompt: '先说主线',
          initial_hints: [],
          session_outline: {
            summary: '并发控制摘要',
            main_question: '它主要解决什么问题？',
            core_concepts: ['并发控制'],
            process_steps: [],
            application_cases: [],
            checkpoints: [],
          },
          turn_index: 3,
          turns: [],
        }}
        selectedMode="light_recall"
        latestStageFeedback={null}
        latestHint={null}
        finalFeedback={null}
        onModeChange={() => {}}
        onStartAnswering={() => {}}
        onRespond={async () => {}}
        onRequestHint={async () => {}}
        onFinish={async () => {}}
      />,
    )

    expect(html).toContain('阅读工作台')
    expect(html).not.toContain('训练工作台')
  })
})
