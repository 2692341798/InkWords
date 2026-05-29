import type { ReactNode } from 'react'

interface GeneratorOutlineStageProps {
  lockedScenarioLabel?: string | null
  outlineEditor: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

/**
 * Why: 大纲出现后，用户应只专注于修改结构和开始生成，而不是回看前序配置面板。
 */
export function GeneratorOutlineStage({
  lockedScenarioLabel,
  outlineEditor,
  progressPanel,
  onBack,
}: GeneratorOutlineStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">确认并调整大纲</h2>
          <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
            当前页面只保留大纲相关操作，确认无误后直接开始生成博客。
          </p>
          {lockedScenarioLabel ? (
            <div className="inline-flex items-center rounded-full border border-zinc-200 bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
              当前创作场景：{lockedScenarioLabel}
            </div>
          ) : null}
        </div>
        <button
          type="button"
          onClick={onBack}
          className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
        >
          返回上一步
        </button>
      </div>

      {outlineEditor}
      {progressPanel}
    </section>
  )
}
