import type { TextbookRuntimeEvidence } from '@/services/textbook'

type Observation = { exit_code?: unknown; output?: unknown }

function readObservation(raw?: string): Observation | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (parsed?.format !== 'inkwords.runtime-observation.v1' || !parsed.observations || typeof parsed.observations !== 'object' || Array.isArray(parsed.observations)) return null
    return parsed.observations
  } catch {
    return null
  }
}

// Render persisted observations as text. Inspecting a receipt neither executes
// its command nor promotes its currentness, review or verification status.
export function RuntimeEvidenceDetails({ evidence }: { evidence: TextbookRuntimeEvidence }) {
  const observation = readObservation(evidence.structured_output)
  const identities = [
    ['运行记录', evidence.id], ['来源修订', evidence.revision_id],
    ['代码工件', evidence.code_artifact_id], ['代码树指纹', evidence.code_artifact_hash],
    ['命令清单指纹', evidence.command_manifest_hash], ['验证输入指纹', evidence.input_hash],
    ['运行器镜像', evidence.runner_image_digest], ['原始证据引用', evidence.raw_evidence_ref],
  ]
  return <details className="mt-3 rounded border border-border bg-muted/20 p-3 text-xs" aria-label={`运行详情 ${evidence.id}`}>
    <summary className="cursor-pointer font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring">查看运行详情与原始记录</summary>
    <p className="mt-2 text-muted-foreground">这是已保存的观察记录。请核对该记录的状态、有效期和失效原因；展开不会重新执行代码。</p>
    <dl className="mt-3 grid gap-2 sm:grid-cols-[auto_1fr]">
      {identities.map(([label, value]) => <div key={label} className="contents"><dt className="text-muted-foreground">{label}</dt><dd className="min-w-0 break-all font-mono">{value || '未记录'}</dd></div>)}
    </dl>
    {evidence.output_truncated ? <p className="mt-3 text-[var(--warning)]">输出已截断，不能据此判断未显示的内容。</p> : null}
    {observation && typeof observation.exit_code === 'number' ? <p className="mt-3 font-medium">退出码：{observation.exit_code}</p> : null}
    {observation && typeof observation.output === 'string' ? <pre aria-label="原始工具输出" className="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-all rounded bg-background p-3 font-mono">{observation.output}</pre> : null}
    {evidence.structured_output ? <>
      {!observation ? <p className="mt-3 text-[var(--warning)]">无法解析为当前运行观察合同，请保留原始记录并核对来源。</p> : null}
      <p className="mt-3 text-muted-foreground">原始运行记录（JSON；保留存储文本）</p>
      <pre aria-label="原始运行记录" className="mt-2 max-h-96 overflow-auto whitespace-pre-wrap break-all rounded bg-background p-3 font-mono">{evidence.structured_output}</pre>
    </> : <p className="mt-3 text-muted-foreground">未保存结构化原始记录。</p>}
  </details>
}
