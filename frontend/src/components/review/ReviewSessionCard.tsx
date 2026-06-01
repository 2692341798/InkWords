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
  onRespond,
  onRequestHint,
  onFinish,
  onClose,
}: ReviewSessionCardProps) {
  const [answer, setAnswer] = useState('')

  const sessionTurns = useMemo(() => session?.turns ?? [], [session?.turns])

  if (!session) {
    return (
      <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-zinc-900">当前会话</h2>
          <p className="text-sm leading-6 text-zinc-600">
            先从“随机抽一篇 / 选择文章复习”任一入口开始，系统会在这里承接会话、提示和复盘。
          </p>
        </div>
        <div className="mt-5 flex gap-3">
          <Button variant={selectedMode === 'light_recall' ? 'default' : 'outline'} onClick={() => onModeChange('light_recall')}>
            轻提示复述
          </Button>
          <Button variant={selectedMode === 'detailed_qa' ? 'default' : 'outline'} onClick={() => onModeChange('detailed_qa')}>
            细致提问
          </Button>
        </div>
      </section>
    )
  }

  return (
    <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-zinc-900">本次会话</h2>
          <p className="mt-1 text-sm text-zinc-600">
            当前主题：{session.title} · 模式：{modeLabels[session.mode]}
          </p>
          <p className="mt-2 text-xs leading-5 text-zinc-500">
            会话开始后模式已锁定，如需切换请返回入口后重新开始。
          </p>
        </div>
        <div className="flex gap-3">
          {onClose ? (
            <Button variant="outline" onClick={onClose}>
              返回入口
            </Button>
          ) : null}
        </div>
      </div>

      <div className="mt-5 rounded-2xl border border-zinc-200 bg-zinc-50 p-4">
        <p className="text-sm font-medium text-zinc-900">开场提示</p>
        <p className="mt-2 text-sm leading-6 text-zinc-600">{session.opening_prompt}</p>
        {session.initial_hints.length > 0 ? (
          <div className="mt-3 flex flex-wrap gap-2">
            {session.initial_hints.map((hint) => (
              <span key={hint} className="rounded-full bg-white px-3 py-1 text-xs text-zinc-600">
                {hint}
              </span>
            ))}
          </div>
        ) : null}
        {session.next_question ? <p className="mt-3 text-sm font-medium text-indigo-700">当前追问：{session.next_question}</p> : null}
      </div>

      <div className="mt-5">
        <label htmlFor="review-answer" className="mb-2 block text-sm font-medium text-zinc-900">
          用自己的话讲一遍
        </label>
        <textarea
          id="review-answer"
          value={answer}
          onChange={(event) => setAnswer(event.target.value)}
          placeholder="先别追求完整，先把你记得的主线讲出来。"
          className="min-h-32 w-full rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3 text-sm leading-6 text-zinc-900 outline-none transition focus:border-indigo-300 focus:bg-white"
        />
        <div className="mt-4 flex flex-wrap gap-3">
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
          <Button variant="outline" onClick={onRequestHint}>
            请求提示
          </Button>
          <Button variant="outline" onClick={onFinish}>
            结束本次复习
          </Button>
        </div>
      </div>

      {latestStageFeedback ? (
        <div className="mt-5 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-900">
          <p className="font-medium">阶段反馈</p>
          <p className="mt-1">{latestStageFeedback}</p>
        </div>
      ) : null}

      {latestHint ? (
        <div className="mt-4 rounded-2xl border border-indigo-200 bg-indigo-50 px-4 py-3 text-sm leading-6 text-indigo-900">
          <p className="font-medium">最新提示</p>
          <p className="mt-1">{latestHint}</p>
        </div>
      ) : null}

      <div className="mt-5 space-y-3">
        <p className="text-sm font-medium text-zinc-900">会话记录</p>
        {/* Why: 会话轮次会持续增长，这里固定可视高度并始终保留纵向滚动能力，避免页面继续被向下撑长。 */}
        <div
          data-slot="session-history-scroll"
          className="h-96 space-y-3 overflow-y-scroll rounded-2xl border border-zinc-200 bg-zinc-50 p-3 pr-2 custom-scrollbar"
        >
          {sessionTurns.length === 0 ? (
            <p className="text-sm text-zinc-500">当前还没有轮次记录。</p>
          ) : (
            sessionTurns.map((turn) => (
              <article key={`${turn.turn_index}-${turn.turn_type}`} className="rounded-2xl border border-zinc-200 bg-white px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-xs font-medium uppercase tracking-wide text-zinc-500">
                    {turn.role === 'user' ? '你的回答' : '系统引导'}
                  </span>
                  <span className="text-xs text-zinc-400">第 {turn.turn_index} 轮 · {turn.turn_type}</span>
                </div>
                <p className="mt-2 text-sm leading-6 text-zinc-700">{turn.content}</p>
              </article>
            ))
          )}
        </div>
      </div>

      {finalFeedback ? (
        <div className="mt-5 rounded-2xl border border-emerald-200 bg-emerald-50 p-4">
          <p className="text-sm font-medium text-emerald-900">最终反馈</p>
          <p className="mt-2 text-sm leading-6 text-emerald-900">{finalFeedback.summary}</p>
          <p className="mt-3 text-xs font-medium uppercase tracking-wide text-emerald-700">你已经抓住的部分</p>
          <p className="mt-1 text-sm text-emerald-900">{finalFeedback.strengths.join('；')}</p>
          <p className="mt-3 text-xs font-medium uppercase tracking-wide text-emerald-700">你还欠缺的部分</p>
          <p className="mt-1 text-sm text-emerald-900">{finalFeedback.gaps.join('；')}</p>
          <p className="mt-3 text-xs font-medium uppercase tracking-wide text-emerald-700">下次优先补哪一点</p>
          <p className="mt-1 text-sm text-emerald-900">{finalFeedback.next_focus.join('；')}</p>
        </div>
      ) : null}
    </section>
  )
}
