import { cn } from '@/lib/utils'

export type StepStripItem = {
  key: string
  title: string
  description?: string
}

type StepStripProps = {
  title?: string
  description?: string
  steps: StepStripItem[]
  currentStepIndex?: number
  variant: 'preview' | 'progress'
  className?: string
}

/**
 * Why: 首页、生成页和复习页都需要“我现在在哪一步”的同一种视觉语言，
 * 但流程状态仍由各页面自己决定，所以这里仅共享展示层，不接管业务逻辑。
 */
export function StepStrip({
  title,
  description,
  steps,
  currentStepIndex,
  variant,
  className,
}: StepStripProps) {
  const gridClassName =
    steps.length >= 4 ? 'md:grid-cols-4' : steps.length === 3 ? 'md:grid-cols-3' : 'md:grid-cols-2'

  const getStepState = (index: number) => {
    if (variant === 'preview' || currentStepIndex === undefined) {
      return 'preview'
    }

    if (index < currentStepIndex) {
      return 'complete'
    }

    if (index === currentStepIndex) {
      return 'current'
    }

    return 'upcoming'
  }

  const getStepEmphasis = (stepState: ReturnType<typeof getStepState>) => {
    if (variant === 'preview') {
      return 'soft'
    }

    if (stepState === 'current') {
      return 'strong'
    }

    return 'soft'
  }

  return (
    <div
      data-variant={variant}
      className={cn(
        'space-y-5 transition-colors',
        className,
      )}
    >
      {title || description ? (
        <div className="space-y-1.5">
          {title ? <h2 className="section-title">{title}</h2> : null}
          {description ? (
            <p className="page-description">{description}</p>
          ) : null}
        </div>
      ) : null}

      <div className={cn('grid gap-3.5', gridClassName)}>
        {steps.map((step, index) => {
          const stepState = getStepState(index)
          const stepEmphasis = getStepEmphasis(stepState)

          return (
            <article
              key={step.key}
              data-variant={variant}
              data-step-state={stepState}
              data-step-emphasis={stepEmphasis}
              className={cn(
                'rounded-[20px] border px-4 py-5 transition-colors duration-200',
                variant === 'preview' &&
                  'border-border bg-secondary/35',
                variant === 'preview' &&
                  step.description &&
                  'min-h-[132px]',
                stepState === 'current' &&
                  'border-[color-mix(in_srgb,var(--brand)_32%,var(--border))] bg-[var(--brand-soft)]',
                stepState === 'complete' &&
                  'border-border bg-secondary/35',
                stepState === 'upcoming' &&
                  'border-border bg-card',
              )}
            >
              <div
                className={cn(
                  'text-[11px] font-medium uppercase tracking-[0.18em]',
                  stepState === 'current'
                    ? 'text-[var(--brand)]'
                    : 'text-muted-foreground',
                )}
              >
                步骤 {index + 1}
              </div>
              <div
                className={cn(
                  'mt-3 text-sm leading-6 text-foreground',
                  stepState === 'current' ? 'font-semibold' : variant === 'preview' ? 'font-medium' : 'font-semibold',
                )}
              >
                {step.title}
              </div>
              {step.description ? (
                <div className="mt-2 text-xs leading-5 text-muted-foreground">{step.description}</div>
              ) : null}
            </article>
          )
        })}
      </div>
    </div>
  )
}
