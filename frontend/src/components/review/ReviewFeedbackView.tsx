import { Button } from '@/components/ui/button'
import type { FinalFeedback, HintResponse, ReviewSessionResponse } from '@/services/review'

export function ReviewFeedbackView({ session, latestHint, finalFeedback, onContinue, onDeepen, onFinish, onClose }: {
  session: ReviewSessionResponse
  latestHint: HintResponse | string | null
  finalFeedback: FinalFeedback | null
  onContinue: () => void
  onDeepen: () => Promise<void> | void
  onFinish: () => Promise<void> | void
  onClose?: () => void
}) {
  const feedback = session.latest_review_feedback
  const completed = session.phase === 'completed' || session.status === 'completed'
  const hits = finalFeedback?.strengths ?? feedback?.hit_points ?? []
  const gaps = finalFeedback?.gaps ?? feedback?.missed_points ?? []
  const hint = typeof latestHint === 'string' ? latestHint : latestHint?.hint_text

  return (
    <section aria-labelledby="review-feedback-title" className="mx-auto max-w-3xl">
      <header className="text-center">
        <p className="text-xs font-medium tracking-[0.18em] text-muted-foreground">{completed ? '复习完成' : '针对性补缺'}</p>
        <h1 id="review-feedback-title" className="mt-3 text-2xl font-semibold text-foreground">{session.title}</h1>
        <p className="mt-3 text-sm text-muted-foreground">{finalFeedback?.summary ?? feedback?.judgement ?? '看看这一轮已经讲清楚了什么。'}</p>
      </header>
      <div className="mt-8 grid gap-4 md:grid-cols-2">
        <article className="rounded-2xl border border-emerald-200 bg-emerald-50/70 p-5 dark:border-emerald-900 dark:bg-emerald-950/30">
          <h2 className="text-sm font-semibold text-emerald-900 dark:text-emerald-100">已经讲清楚</h2>
          <ul className="mt-3 space-y-2 text-sm leading-6 text-emerald-900 dark:text-emerald-100">{hits.length ? hits.map((item) => <li key={item}>✓ {item}</li>) : <li>这一轮还没有明确命中的关键点。</li>}</ul>
        </article>
        <article className="rounded-2xl border border-amber-200 bg-amber-50/70 p-5 dark:border-amber-900 dark:bg-amber-950/30">
          <h2 className="text-sm font-semibold text-amber-900 dark:text-amber-100">下一轮只补这些</h2>
          <ul className="mt-3 space-y-2 text-sm leading-6 text-amber-900 dark:text-amber-100">{gaps.length ? gaps.map((item) => <li key={item}>○ {item}</li>) : <li>关键点已覆盖得比较完整。</li>}</ul>
        </article>
      </div>
      {hint ? <p className="mt-4 rounded-xl border border-border bg-secondary/40 px-4 py-3 text-sm leading-6 text-foreground">提示：{hint}</p> : null}
      <div className="mt-6 flex flex-wrap justify-end gap-3">
        {completed && onClose ? <Button variant="outline" onClick={onClose}>返回选文</Button> : null}
        {!completed ? <Button variant="outline" onClick={onDeepen}>深入追问</Button> : null}
        {!completed ? <Button variant="outline" onClick={onFinish}>完成本次复习</Button> : null}
        {!completed ? <Button onClick={onContinue}>继续补充</Button> : null}
      </div>
    </section>
  )
}
