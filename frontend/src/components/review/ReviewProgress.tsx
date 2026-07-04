import type { ReviewPhase } from '@/services/review'

const steps = [
  { phase: 'reading', label: '阅读' },
  { phase: 'recalling', label: '复述' },
  { phase: 'coaching', label: '补缺' },
  { phase: 'completed', label: '完成' },
] as const

export function ReviewProgress({ phase }: { phase: ReviewPhase }) {
  const current = steps.findIndex((step) => step.phase === phase)
  return (
    <nav aria-label="复习进度" className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
      {steps.map((step, index) => (
        <div key={step.phase} className="flex items-center gap-2">
          <span aria-current={index === current ? 'step' : undefined} className={index <= current ? 'font-medium text-foreground' : ''}>
            {index + 1}. {step.label}
          </span>
          {index < steps.length - 1 ? <span aria-hidden="true" className="text-border">/</span> : null}
        </div>
      ))}
    </nav>
  )
}
