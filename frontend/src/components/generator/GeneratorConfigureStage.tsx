import type { ReactNode } from 'react'

interface GeneratorConfigureStageProps {
  sourceLabel: string
  scenarioSelector: ReactNode
  modulePicker?: ReactNode
  fileSummary?: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

/**
 * Why: 选择来源之后，用户只需要处理“如何解析”这一件事，避免继续看到来源入口和大纲编辑。
 */
export function GeneratorConfigureStage({
  sourceLabel,
  scenarioSelector,
  modulePicker,
  fileSummary,
  progressPanel,
  onBack,
}: GeneratorConfigureStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="text-sm text-zinc-500 dark:text-zinc-400">当前来源：{sourceLabel}</div>
          <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">配置解析方式</h2>
          <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
            这一步只保留当前来源所需的配置项，确认后系统会进入处理页并生成可编辑大纲。
          </p>
        </div>
        <button
          type="button"
          onClick={onBack}
          className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
        >
          返回上一步
        </button>
      </div>

      {scenarioSelector}
      {modulePicker}
      {fileSummary}
      {progressPanel}
    </section>
  )
}
