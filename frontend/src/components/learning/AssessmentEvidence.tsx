import type { AssessmentJob } from '@/services/masteryAssessment'

type Evidence = AssessmentJob['input']['evidence'][number]

export function AssessmentEvidence({ evidence, ids }: { evidence: Evidence[]; ids: string[] }) {
  if (!ids.length) return null
  return <details className="mt-2">
    <summary className="cursor-pointer text-sm">本项引用原文</summary>
    {ids.map((id) => {
      const item = evidence.find((entry) => entry.id === id)
      return <div key={id} className="mt-2">
        <p className="break-all text-xs text-muted-foreground">{id}</p>
        <pre className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs">{item?.excerpt ?? '此引用不在本次冻结依据中，请读取最新评分后核对。'}</pre>
      </div>
    })}
  </details>
}

export function CorrectionEvidence({ evidence, selected, onChange }: { evidence: Evidence[]; selected: string[]; onChange: (ids: string[]) => void }) {
  return <fieldset className="space-y-2 rounded-md border border-border p-3">
    <legend>纠正引用依据</legend>
    <p className="text-xs text-muted-foreground">只选择支持本项判断的冻结来源。核对本项要求、作答原文与所选来源；来源中还有其他细节，不代表本题必须回答。</p>
    {evidence.map((item) => <div key={item.id}>
      <label className="flex items-start gap-2 text-sm">
        <input type="checkbox" aria-label={`引用 ${item.id}`} checked={selected.includes(item.id)} onChange={(event) => onChange(event.target.checked ? [...selected, item.id] : selected.filter((id) => id !== item.id))} />
        <span className="break-all">{item.kind === 'runtime' ? '运行证据' : '来源原文'} · {item.id}</span>
      </label>
      <details className="ml-5 mt-1">
        <summary className="cursor-pointer text-xs">查看依据原文</summary>
        <pre className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap break-words text-xs">{item.excerpt}</pre>
      </details>
    </div>)}
  </fieldset>
}
