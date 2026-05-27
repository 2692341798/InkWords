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
        'space-y-6 transition-colors',
        variant === 'preview' ? 'rounded-3xl' : 'rounded-3xl',
        className,
      )}
    >
      {title || description ? (
        <div className="space-y-1.5">
          {title ? <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{title}</h2> : null}
          {description ? (
            <p className="max-w-3xl text-sm leading-6 text-zinc-500 dark:text-zinc-400">{description}</p>
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
                  'border-zinc-200/90 bg-zinc-50/90 dark:border-zinc-800 dark:bg-zinc-900/95',
                variant === 'preview' &&
                  step.description &&
                  'min-h-[132px]',
                stepState === 'current' &&
                  'border-zinc-900/90 bg-white shadow-[0_10px_30px_rgba(15,23,42,0.06)] dark:border-zinc-100 dark:bg-zinc-900',
                stepState === 'complete' &&
                  'border-zinc-200 bg-zinc-50/75 dark:border-zinc-700 dark:bg-zinc-800/35',
                stepState === 'upcoming' &&
                  'border-zinc-200/90 bg-white/95 dark:border-zinc-800 dark:bg-zinc-900/90',
              )}
            >
              <div
                className={cn(
                  'text-[11px] font-medium uppercase tracking-[0.18em]',
                  stepState === 'current'
                    ? 'text-zinc-700 dark:text-zinc-300'
                    : 'text-zinc-400 dark:text-zinc-500',
                )}
              >
                步骤 {index + 1}
              </div>
              <div
                className={cn(
                  'mt-3 text-sm leading-6 text-zinc-900 dark:text-zinc-100',
                  stepState === 'current' ? 'font-semibold' : variant === 'preview' ? 'font-medium' : 'font-semibold',
                )}
              >
                {step.title}
              </div>
              {step.description ? (
                <div className="mt-2 text-xs leading-5 text-zinc-500 dark:text-zinc-400">{step.description}</div>
              ) : null}
            </article>
          )
        })}
      </div>
    </div>
  )
}
