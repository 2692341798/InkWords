import { useState } from 'react'
import { Button } from '@/components/ui/button'
import type { HintResponse, ReviewSessionResponse } from '@/services/review'

export function ReviewRecallView({
  session,
  latestHint,
  pending = false,
  onRespond,
  onRequestHint,
}: {
  session: ReviewSessionResponse
  latestHint: HintResponse | string | null
  pending?: boolean
  onRespond: (answer: string) => Promise<void> | void
  onRequestHint: (answer: string) => Promise<void> | void
}) {
  const [answer, setAnswer] = useState('')
  const hintText = typeof latestHint === 'string' ? latestHint : latestHint?.hint_text
  const hintMeta = typeof latestHint === 'string' ? null : latestHint

  return (
    <section aria-labelledby="review-recall-title" className="mx-auto max-w-3xl">
      <header className="text-center">
        <p className="text-xs font-medium tracking-[0.18em] text-muted-foreground">关书复述</p>
        <h1 id="review-recall-title" className="mt-3 text-2xl font-semibold text-foreground">{session.title}</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">{session.session_outline.main_question || '不求逐字复现，先把文章的主线讲清楚。'}</p>
      </header>
      <label htmlFor="review-answer" className="mt-8 block text-sm font-medium text-foreground">用自己的话讲一遍</label>
      <textarea
        id="review-answer"
        autoFocus
        value={answer}
        onChange={(event) => setAnswer(event.target.value)}
        placeholder="原文已经收起。先写下你记得的主旨、关键关系和例子……"
        className="mt-2 min-h-72 w-full resize-y rounded-2xl border border-border bg-card px-5 py-4 text-base leading-7 text-foreground outline-none transition focus:border-[var(--brand)] focus:ring-2 focus:ring-[var(--brand-soft)]"
      />
      {hintText ? (
        <aside aria-live="polite" className="mt-4 rounded-xl border border-[color-mix(in_srgb,var(--brand)_20%,var(--border))] bg-[var(--brand-soft)] px-4 py-3 text-sm leading-6 text-foreground">
          {hintMeta?.target_gap ? <p className="mb-1 text-xs font-medium text-muted-foreground">聚焦：{hintMeta.target_gap}</p> : null}
          <p>{hintText}</p>
          {hintMeta?.next_action ? <p className="mt-2 font-medium">接下来：{hintMeta.next_action}</p> : null}
          {hintMeta?.source_anchor ? <p className="mt-1 text-xs text-muted-foreground">原文位置：{hintMeta.source_anchor}</p> : null}
        </aside>
      ) : null}
      <div className="mt-5 flex flex-wrap justify-end gap-3">
        <Button variant="outline" onClick={() => onRequestHint(answer)}>{pending ? '正在生成提示…' : '我卡住了'}</Button>
        <Button disabled={!answer.trim()} onClick={async () => { const value = answer.trim(); if (!value) return; await onRespond(value); setAnswer('') }}>提交复述</Button>
      </div>
    </section>
  )
}
