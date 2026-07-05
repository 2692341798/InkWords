import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Button } from '@/components/ui/button'
import type { ReviewSessionResponse } from '@/services/review'
import { getReadingContent } from './reviewPhase'

export function ReviewReadingView({ session, onComplete }: { session: ReviewSessionResponse; onComplete: () => Promise<void> | void }) {
  return (
    <section aria-labelledby="review-reading-title" className="mx-auto max-w-3xl pb-28">
      <header className="mb-10 border-b border-border pb-7 text-center">
        <p className="text-xs font-medium tracking-[0.18em] text-muted-foreground">沉浸阅读</p>
        <h1 id="review-reading-title" className="mt-3 text-3xl font-semibold tracking-tight text-foreground">{session.title}</h1>
        <p className="mt-3 text-sm text-muted-foreground">先读懂主线。读完后我们会收起原文，再请你用自己的话讲一遍。</p>
      </header>
      <article className="prose prose-zinc mx-auto max-w-none text-[15px] leading-8 dark:prose-invert">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{getReadingContent(session)}</ReactMarkdown>
      </article>
      <div className="fixed inset-x-0 bottom-0 z-20 border-t border-border bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80">
        <div className="mx-auto flex max-w-3xl justify-end">
          <Button size="lg" onClick={onComplete}>我已浏览完毕，开始复述</Button>
        </div>
      </div>
    </section>
  )
}
