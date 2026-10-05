import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'

type UsageData = Record<string, unknown>

const numberValue = (value: unknown) => typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : '未知'
const booleanValue = (value: unknown) => value === true

// UsagePanel makes provenance telemetry readable without treating absent
// provider fields as zero-cost or zero-token facts.
export function UsagePanel({ usage }: { usage?: UsageData }) {
  const known = booleanValue(usage?.known)
  const calls = numberValue(usage?.provider_call_count)
  const latencyMS = numberValue(usage?.provider_latency_ms)
  const costKnown = booleanValue(usage?.estimated_cost_known)
  const cacheStatus = usage?.local_cache_hit === true
    ? calls === 0 ? '本地缓存命中，未发起新的供应商调用。' : '本地缓存命中。'
    : usage?.local_cache_hit === false ? '本次没有本地缓存命中。' : '本地缓存状态未知。'

  return <Panel className="mt-3 p-3" aria-label="生成使用量">
    <SectionHeader eyebrow="生成谱系" title="使用量与成本" description="展示当前记录中的供应商遥测；未返回的字段标为未知。" action={<StatusPill tone={known ? 'success' : 'warning'}>{known ? 'Token 已记录' : 'Token 未知'}</StatusPill>} />
    {known ? <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-3"><div><dt className="text-muted-foreground">输入 Token</dt><dd>{numberValue(usage?.input_tokens)}</dd></div><div><dt className="text-muted-foreground">输出 Token</dt><dd>{numberValue(usage?.output_tokens)}</dd></div><div><dt className="text-muted-foreground">缓存 Token</dt><dd>{numberValue(usage?.cached_tokens)}</dd></div></dl> : <p className="mt-3 text-xs text-muted-foreground">供应商未返回 Token 使用量（未知，不按 0 计）。</p>}
    <p className="mt-3 text-xs text-muted-foreground">供应商调用：{calls} 次 · 调用耗时：{latencyMS} ms · {cacheStatus}</p>
    <p className="mt-1 text-xs text-muted-foreground">估算费用：{costKnown ? '已记录' : '未知（尚未配置版本化定价规则）'}。</p>
  </Panel>
}
