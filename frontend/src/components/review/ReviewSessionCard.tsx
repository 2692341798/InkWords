import { useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import type { FinalFeedback, ReviewMode, ReviewSessionResponse } from '@/services/review'

interface ReviewSessionCardProps {
  session: ReviewSessionResponse | null
  selectedMode: ReviewMode
  latestStageFeedback: string | null
  latestHint: string | null
  finalFeedback: FinalFeedback | null
  onModeChange: (mode: ReviewMode) => void
  onStartAnswering?: () => void
  onRespond: (answer: string) => Promise<void> | void
  onRequestHint: () => Promise<void> | void
  onFinish: () => Promise<void> | void
  onClose?: () => void
}

const modeLabels: Record<ReviewMode, string> = {
  light_recall: '轻提示复述',
  detailed_qa: '细致提问',
}

export function ReviewSessionCard({
  session,
  selectedMode,
  latestStageFeedback,
  latestHint,
  finalFeedback,
  onModeChange,
  onStartAnswering,
  onRespond,
  onRequestHint,
  onFinish,
  onClose,
}: ReviewSessionCardProps) {
  const [answer, setAnswer] = useState('')
  const [isSourceDrawerOpen, setIsSourceDrawerOpen] = useState(false)

  const sessionTurns = useMemo(() => session?.turns ?? [], [session?.turns])
  const currentQuestion = session?.next_question || session?.session_outline.main_question || session?.opening_prompt || '先从你记得的主线开始。'
  const smartHint = latestHint || session?.initial_hints[0] || '需要线索时再请求一次，系统会给你当前最相关的一条提示。'
  const nextAction = session?.latest_review_feedback?.suggestion || session?.current_round_goal || '先围绕当前问题讲出主线，再补一个关键细节。'
  const focusTopics = useMemo(
    () => (session ? [...session.session_outline.core_concepts, ...session.session_outline.process_steps].filter(Boolean).slice(0, 4) : []),
    [session],
  )

  if (!session) {
    return (
      <section className="overflow-hidden rounded-[28px] border border-stone-200 bg-white shadow-[0_20px_48px_rgba(15,23,42,0.06)]">
        <div className="border-b border-stone-200 px-6 py-5">
          <p className="text-xs font-medium tracking-[0.28em] text-stone-500">阅读工作台</p>
          <h2 className="mt-3 text-2xl font-semibold text-stone-950">当前还没有开始复习</h2>
          <p className="mt-2 max-w-2xl text-sm leading-7 text-stone-600">
            先从“随机抽一篇”或“手动选文”开始。进入会话后，这里会变成更安静的复盘工作台，先帮助你整理主线，再补提示和反馈。
          </p>
        </div>
        <div className="p-6">
          <div className="flex flex-wrap gap-3">
            <span className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">极简阅读优先</span>
            <span className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">原文抽屉式阅读</span>
          </div>
        </div>
        <div className="border-t border-stone-200 px-6 py-5">
          <p className="text-sm font-medium text-stone-900">先选择复习模式</p>
          <div className="mt-4 flex gap-3">
            <Button variant={selectedMode === 'light_recall' ? 'default' : 'outline'} onClick={() => onModeChange('light_recall')}>
              轻提示复述
            </Button>
            <Button variant={selectedMode === 'detailed_qa' ? 'default' : 'outline'} onClick={() => onModeChange('detailed_qa')}>
              细致提问
            </Button>
          </div>
        </div>
      </section>
    )
  }

  const feedbackSummary = session.latest_review_feedback
  const sourceText = session.source_preview || session.session_outline.summary

  return (
    <section data-slot="review-reading-workspace" className="overflow-hidden rounded-[28px] border border-stone-200 bg-white shadow-[0_20px_48px_rgba(15,23,42,0.06)]">
      <div className="border-b border-stone-200 px-6 py-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-xs font-medium tracking-[0.28em] text-stone-500">阅读工作台</p>
            <h2 className="mt-3 text-2xl font-semibold text-stone-950">{session.title}</h2>
            <p className="mt-2 max-w-3xl text-sm leading-7 text-stone-600">
              当前模式：{modeLabels[session.mode]}。这里优先保留阅读与复述所需的信息，把原文收进抽屉，需要时再展开查看。
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <span className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">当前阶段：{session.ready_to_answer ? '开始会话' : '预览原文'}</span>
            <span className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">
              模式：{modeLabels[session.mode]}
            </span>
            <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-50" onClick={() => setIsSourceDrawerOpen((open) => !open)}>
              {isSourceDrawerOpen ? '收起原文' : '查看原文'}
            </Button>
            {onClose ? (
              <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-50" onClick={onClose}>
                返回入口
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      {!session.ready_to_answer ? (
        <div className="grid gap-6 p-6 lg:grid-cols-[minmax(0,1fr)_320px]">
          <article className="rounded-[24px] border border-stone-200 bg-stone-50 p-6">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-xs font-medium tracking-[0.24em] text-stone-500">原文预览</p>
                <h3 className="mt-3 text-xl font-semibold text-stone-900">{session.source_title || session.title}</h3>
              </div>
              <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-100" onClick={() => setIsSourceDrawerOpen((open) => !open)}>
                {isSourceDrawerOpen ? '收起全文' : '展开全文'}
              </Button>
            </div>
            <div className="mt-5 max-h-72 overflow-y-auto whitespace-pre-wrap rounded-[20px] border border-stone-200 bg-white px-5 py-4 text-sm leading-8 text-stone-700 custom-scrollbar">
              {sourceText}
            </div>
            <div className="mt-6 flex items-center justify-between gap-4 rounded-[20px] border border-stone-200 bg-white px-4 py-3">
              <p className="text-sm leading-6 text-stone-600">先快速浏览原文，再开始复述。若后面需要回看细节，可以随时通过原文抽屉返回全文。</p>
              {onStartAnswering ? <Button onClick={onStartAnswering}>开始复述</Button> : null}
            </div>
          </article>

          <aside className="space-y-4">
            <article className="rounded-[24px] border border-stone-200 bg-white p-5">
              <p className="text-xs font-medium tracking-[0.24em] text-stone-500">阅读提示</p>
              <p className="mt-4 text-sm leading-7 text-stone-600">先记住这篇内容在解决什么问题，再进入复述，会比一开始就追细节更轻松。</p>
            </article>
            <article className="rounded-[24px] border border-stone-200 bg-stone-50 p-5">
              <p className="text-sm font-medium text-stone-900">当前问题</p>
              <p className="mt-2 text-sm leading-7 text-stone-700">{currentQuestion}</p>
            </article>
            <article className="rounded-[24px] border border-stone-200 bg-white p-5">
              <p className="text-sm font-medium text-stone-900">进入前先留意</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {focusTopics.length > 0 ? (
                  focusTopics.map((item) => (
                    <span key={item} className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">
                      {item}
                    </span>
                  ))
                ) : (
                  <span className="text-sm text-stone-500">先抓主线，再补一个关键概念或例子。</span>
                )}
              </div>
            </article>
          </aside>
        </div>
      ) : (
        <div className="grid gap-6 p-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,360px)]">
          <div className="space-y-5">
            <article className="rounded-[24px] border border-stone-200 bg-stone-50 p-6">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p className="text-xs font-medium tracking-[0.24em] text-stone-500">当前问题</p>
                  <h3 className="mt-3 text-2xl font-semibold tracking-tight text-stone-950">{currentQuestion}</h3>
                </div>
                <span className="rounded-full border border-stone-200 bg-white px-3 py-1 text-xs text-stone-600">
                  第 {session.turn_index} 轮
                </span>
              </div>
              <p className="mt-4 text-sm leading-7 text-stone-600">{session.current_round_goal || '先围绕当前问题讲主线，再补一个更具体的细节。'}</p>
              {focusTopics.length > 0 ? (
                <div className="mt-4 flex flex-wrap gap-2">
                  {focusTopics.map((item) => (
                    <span key={item} className="rounded-full border border-stone-200 bg-white px-3 py-1 text-xs text-stone-600">
                      {item}
                    </span>
                  ))}
                </div>
              ) : null}
            </article>

            {latestStageFeedback ? (
              <article className="rounded-[24px] border border-amber-200 bg-amber-50/70 px-5 py-4 text-sm leading-7 text-amber-950">
                <p className="font-medium">阶段反馈</p>
                <p className="mt-2">{latestStageFeedback}</p>
              </article>
            ) : null}

            <article className="rounded-[24px] border border-stone-200 bg-white p-6">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-medium text-stone-900">你的复述</p>
                  <p className="mt-1 text-sm leading-6 text-stone-500">先说你现在最确定的一句，不必一次讲全。</p>
                </div>
                <span className="rounded-full border border-stone-200 bg-stone-50 px-3 py-1 text-xs text-stone-600">聚焦当前动作</span>
              </div>
              <textarea
                id="review-answer"
                value={answer}
                onChange={(event) => setAnswer(event.target.value)}
                placeholder="先从主线说起，再补一个你印象最深的关键点。"
                className="mt-5 min-h-44 w-full rounded-[20px] border border-stone-200 bg-stone-50 px-5 py-4 text-sm leading-7 text-stone-900 outline-none transition focus:border-stone-300 focus:bg-white"
              />
              <div className="mt-5 flex flex-wrap gap-3">
                <Button
                  onClick={async () => {
                    const trimmed = answer.trim()
                    if (!trimmed) {
                      return
                    }
                    await onRespond(trimmed)
                    setAnswer('')
                  }}
                >
                  提交回答
                </Button>
                <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-50" onClick={onRequestHint}>
                  换个角度提示
                </Button>
                <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-50" onClick={onFinish}>
                  结束本次复习
                </Button>
              </div>
            </article>
          </div>

          <aside className="space-y-4">
            <article className="rounded-[24px] border border-stone-200 bg-stone-50 p-5">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs font-medium tracking-[0.24em] text-stone-500">原文抽屉</p>
                  <h3 className="mt-2 text-lg font-semibold text-stone-900">{session.source_title || session.title}</h3>
                </div>
                <Button variant="outline" className="border-stone-200 bg-white text-stone-700 hover:bg-stone-100" onClick={() => setIsSourceDrawerOpen((open) => !open)}>
                  {isSourceDrawerOpen ? '收起原文' : '查看原文'}
                </Button>
              </div>
              <div
                data-slot="source-drawer-scroll"
                className={`mt-4 overflow-y-auto whitespace-pre-wrap rounded-[20px] border border-stone-200 bg-white px-4 py-4 text-sm leading-7 text-stone-700 custom-scrollbar transition-all ${
                  isSourceDrawerOpen ? 'max-h-[32rem] opacity-100' : 'max-h-28 opacity-90'
                }`}
              >
                {sourceText}
              </div>
              <p className="mt-3 text-xs leading-6 text-stone-500">原文面板自带独立滚动条，不会把整页继续向下撑长。</p>
            </article>

            <article className="rounded-[24px] border border-stone-200 bg-white p-5">
              <p className="text-xs font-medium tracking-[0.24em] text-stone-500">当前目标</p>
              <p className="mt-3 text-sm leading-7 text-stone-700">{session.current_round_goal || '先把当前问题讲清楚。'}</p>
              <div className="mt-4 rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                <p className="text-xs uppercase tracking-wide text-stone-500">下一步动作</p>
                <p className="mt-2 text-sm leading-7 text-stone-700">{nextAction}</p>
              </div>
            </article>

            <article className="rounded-[24px] border border-stone-200 bg-white p-5">
              <p className="text-xs font-medium tracking-[0.24em] text-stone-500">智能提示</p>
              <p className="mt-3 text-sm leading-7 text-stone-700">{smartHint}</p>
            </article>

            {feedbackSummary ? (
              <article className="rounded-[24px] border border-stone-200 bg-white p-5 text-stone-900">
                <p className="text-sm font-medium">{feedbackSummary.judgement}</p>
                <div className="mt-4 grid gap-3">
                  <div className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                    <p className="text-xs uppercase tracking-wide text-stone-500">答到的点</p>
                    <p className="mt-2 text-sm leading-7 text-stone-700">
                      {feedbackSummary.hit_points.length > 0 ? feedbackSummary.hit_points.join('；') : '这一轮还没有明显命中的关键点。'}
                    </p>
                  </div>
                  <div className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                    <p className="text-xs uppercase tracking-wide text-stone-500">还需补位</p>
                    <p className="mt-2 text-sm leading-7 text-stone-700">
                      {feedbackSummary.missed_points.length > 0 ? feedbackSummary.missed_points.join('；') : '当前关键点已经覆盖得比较完整。'}
                    </p>
                  </div>
                </div>
              </article>
            ) : null}
          </aside>
        </div>
      )}

      <div className="border-t border-stone-200 bg-stone-50/60 p-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-stone-900">最近轨迹</p>
            <p className="mt-1 text-sm text-stone-500">保留当前训练的关键过程，便于你回看自己是怎么一步步接近答案的。</p>
          </div>
          <span className="rounded-full border border-stone-200 bg-white px-3 py-1 text-xs text-stone-500">会话记录</span>
        </div>
        <div
          data-slot="session-history-scroll"
          className="mt-4 h-96 space-y-3 overflow-y-scroll rounded-[20px] border border-stone-200 bg-white p-3 pr-2 custom-scrollbar"
        >
          {sessionTurns.length === 0 ? (
            <p className="px-3 py-4 text-sm text-stone-500">当前还没有轮次记录。</p>
          ) : (
            sessionTurns.map((turn) => (
              <article key={`${turn.turn_index}-${turn.turn_type}`} className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-xs font-medium uppercase tracking-wide text-stone-500">
                    {turn.role === 'user' ? '你的回答' : '系统引导'}
                  </span>
                  <span className="text-xs text-stone-400">第 {turn.turn_index} 轮 · {turn.turn_type}</span>
                </div>
                <p className="mt-2 text-sm leading-7 text-stone-700">{turn.content}</p>
              </article>
            ))
          )}
        </div>
      </div>

      {finalFeedback ? (
        <div className="border-t border-stone-200 bg-stone-50/60 p-6">
          <div className="rounded-[24px] border border-stone-200 bg-white p-5">
            <p className="text-sm font-medium text-stone-900">最终反馈</p>
            <p className="mt-2 text-sm leading-7 text-stone-700">{finalFeedback.summary}</p>
            <div className="mt-4 grid gap-3 md:grid-cols-3">
              <div className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                <p className="text-xs uppercase tracking-wide text-stone-500">已经抓住</p>
                <p className="mt-2 text-sm leading-7 text-stone-700">{finalFeedback.strengths.join('；')}</p>
              </div>
              <div className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                <p className="text-xs uppercase tracking-wide text-stone-500">还需补强</p>
                <p className="mt-2 text-sm leading-7 text-stone-700">{finalFeedback.gaps.join('；')}</p>
              </div>
              <div className="rounded-[18px] border border-stone-200 bg-stone-50 px-4 py-3">
                <p className="text-xs uppercase tracking-wide text-stone-500">下次优先</p>
                <p className="mt-2 text-sm leading-7 text-stone-700">{finalFeedback.next_focus.join('；')}</p>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
