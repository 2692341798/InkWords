import type { TextbookGenerationTaskSnapshot } from '@/services/textbook'
import { UsagePanel } from './UsagePanel'

const record = (value: unknown): Record<string, unknown> | undefined =>
  typeof value === 'object' && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : undefined
const text = (value: unknown) => typeof value === 'string' && value.trim() ? value : '未知'

const failureExplanation = (failure: string) => {
  if (failure.startsWith('undefined_repeated_acronym: ')) return `缩写首次出现时缺少中文释义：${failure.slice('undefined_repeated_acronym: '.length)}`
  if (failure.startsWith('code_origin_missing: ')) return '代码块缺少来源说明，需要区分教学实现与上游源码。'
  if (failure === 'teaching_go_files_invalid') return '教学实现未按要求提供两个完整的 Go 代码块：main.go 和 main_test.go。'
  return '该项未满足自动质量合同，请结合失败代码核对生成约束。'
}

// Older tasks and other task subtypes do not carry this rejected-candidate
// contract. Render only its diagnostic fields, never arbitrary task output.
export function GenerationFailureDetails({ task }: { task: TextbookGenerationTaskSnapshot }) {
  const result = record(task.result)
  const quality = record(result?.quality_report_json)
  if (task.status !== 'failed' || task.task_subtype !== 'textbook_sample_generate'
    || result?.result_version !== 1 || result.task_subtype !== task.task_subtype
    || result.final_status !== 'failed' || result.candidate_persisted !== false || quality?.passed !== false) return null

  const failures = Array.isArray(quality.failures) ? quality.failures.filter((value): value is string => typeof value === 'string') : []
  const diagnostics = (Array.isArray(quality.failure_diagnostics) ? quality.failure_diagnostics : [])
    .slice(0, 10).map(record).filter((value): value is Record<string, unknown> => Boolean(value)
      && typeof value?.failure === 'string' && failures.includes(value.failure)
      && Number.isInteger(value.line) && Number(value.line) > 0
      && Number.isInteger(value.occurrences) && Number(value.occurrences) >= 2
      && ['manuscript', 'practice_set'].includes(String(value.area))
      && typeof value.excerpt === 'string' && Array.from(value.excerpt).length <= 160)
  return <section aria-label="失败生成诊断" className="mt-3 rounded-md border border-border bg-background p-3">
    <h3 className="text-sm font-medium">自动质量检查未通过</h3>
    <p className="mt-2 text-xs text-muted-foreground">未生成可审阅候选稿。以下是生成任务的诊断和用量，不代表教学代码已经运行或验证。</p>
    <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-2">
      <div><dt className="text-muted-foreground">生成器</dt><dd>{text(result.provider_name)} / {text(result.model_name)}</dd></div>
      <div><dt className="text-muted-foreground">本次自动质量合同</dt><dd>{text(quality.contract_version)}</dd></div>
    </dl>
    {failures.length > 0 ? <ul aria-label="自动检查失败项" className="mt-3 list-disc space-y-2 break-words pl-5 text-xs">{failures.map((failure, index) => <li key={`${index}:${failure}`}><p>{failureExplanation(failure)}</p><code className="text-muted-foreground">{failure}</code></li>)}</ul> : <p className="mt-3 text-xs text-muted-foreground">未记录具体失败项。</p>}
    {diagnostics.length > 0 && <div aria-label="失败位置与片段" className="mt-3 space-y-3 text-xs">
      <p className="text-muted-foreground">以下仅是未通过检查的短片段，用于定位问题，不是已批准教材。</p>
      {diagnostics.map((diagnostic, index) => <div key={index}>
        <p>{diagnostic.area === 'practice_set' ? '练习与答案' : '教材正文'} · 原稿第 {Number(diagnostic.line)} 行 · 出现 {Number(diagnostic.occurrences)} 次</p>
        <blockquote className="mt-1 whitespace-pre-wrap break-words border-l-2 border-border pl-3">{String(diagnostic.excerpt)}</blockquote>
      </div>)}
    </div>}
    <UsagePanel usage={record(result.provider_usage_json)} />
  </section>
}
