import { useState } from 'react'
import { CircleAlert, FileCode2, TerminalSquare } from 'lucide-react'
import { StatusPill } from '@/components/ui/workspace'
import type { ArtifactStatus, ChapterWorkspace, TextbookCodeArtifact, TextbookManuscriptAsset, TextbookRuntimeEvidence } from '@/services/textbook'
import { ArtifactVerificationControls } from './ArtifactVerificationControls'
import { RuntimeEvidenceDetails } from './RuntimeEvidenceDetails'

interface VerificationPanelProps {
  chapterId?: string
  revisionNumbers?: Record<string, number>
  currentSourceRevisionId?: string
  artifacts: TextbookCodeArtifact[]
  evidence: TextbookRuntimeEvidence[]
  assets: TextbookManuscriptAsset[]
}

const statusLabel: Record<ArtifactStatus, string> = {
  draft: '草稿',
  unverified: '未验证',
  verified: '已验证',
  blocked: '已阻断',
}

const statusTone = (status: ArtifactStatus) => status === 'verified' ? 'success' : 'warning'

const shortHash = (value: string) => value.length > 20 ? `${value.slice(0, 14)}…${value.slice(-6)}` : value

const visualPurposeLabel: Record<TextbookManuscriptAsset['visual_purpose'], string> = {
  layout: '页面布局', memory_map: '内存图', call_stack: '调用栈', network_flow: '网络流', rendered_ui: '实际界面状态', legacy_unclassified: '旧资产：未分类',
}

const isCurrentEvidence = (item: TextbookRuntimeEvidence, now: number) => item.status === 'verified' && !item.stale_reason && Boolean(item.expires_at) && new Date(item.expires_at!).getTime() > now

const evidenceStatus = (item: TextbookRuntimeEvidence, now: number): ArtifactStatus => item.status === 'verified' && !isCurrentEvidence(item, now) ? 'unverified' : item.status

const observationSummary = (structuredOutput?: string) => {
  if (!structuredOutput) return null
  try {
    const output = JSON.parse(structuredOutput) as { format?: string; observations?: { commands?: unknown[]; browser_pages?: Array<{ browser_name?: string; browser_version?: string; playwright_version?: string }> }; interpretations?: unknown[] }
    if (output.format !== 'inkwords.runtime-observation.v1' || !output.observations) return '运行输出为旧版或无效结构，不能据此区分观察与解释。'
    const interpretationCount = Array.isArray(output.interpretations) ? output.interpretations.length : 0
    const browserPage = Array.isArray(output.observations.browser_pages) ? output.observations.browser_pages[0] : undefined
    if (browserPage?.browser_name && browserPage.browser_version && browserPage.playwright_version) {
      return `浏览器观察：Playwright ${browserPage.playwright_version}；${browserPage.browser_name} ${browserPage.browser_version}；解释：${interpretationCount === 0 ? '未记录' : `${interpretationCount} 条（须标明依据）`}。`
    }
    const commandCount = Array.isArray(output.observations.commands) ? output.observations.commands.length : 1
    return `结构化观察：${commandCount} 条工具结果；解释：${interpretationCount === 0 ? '未记录' : `${interpretationCount} 条（须标明依据）`}。`
  } catch {
    return '运行输出为旧版或无效结构，不能据此区分观察与解释。'
  }
}

export function VerificationPanel({ artifacts: originalArtifacts, evidence: originalEvidence, assets: originalAssets, chapterId, revisionNumbers, currentSourceRevisionId }: VerificationPanelProps) {
  const [refreshed, setRefreshed] = useState<ChapterWorkspace | null>(null)
  const artifacts = refreshed?.code_artifacts ?? originalArtifacts
  const evidence = refreshed?.runtime_evidence ?? originalEvidence
  const assets = refreshed?.assets ?? originalAssets
  // A stable snapshot keeps this render pure while ensuring evidence that was
  // already stale when the workspace opened cannot be shown as current.
  const [now] = useState(() => Date.now())

  if (artifacts.length === 0) return null

  const hasCurrentVerifiedEvidence = evidence.some((item) => isCurrentEvidence(item, now))
  const evidenceByArtifact = new Map<string, TextbookRuntimeEvidence[]>()
  for (const item of evidence) evidenceByArtifact.set(item.code_artifact_id, [...(evidenceByArtifact.get(item.code_artifact_id) ?? []), item])

  return <section className="mt-6 border-t border-border pt-5" aria-label="教学代码运行证据">
    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div>
        <h3 className="flex items-center gap-2 font-medium"><TerminalSquare className="h-4 w-4" />教学代码与运行证据</h3>
        <p className="mt-1 text-sm text-muted-foreground">这里仅显示系统生成、哈希匹配的教学工件。生成完成不等于代码已经运行；未接入受控隔离执行器时，状态会保留为“未验证”。</p>
      </div>
      <StatusPill tone={hasCurrentVerifiedEvidence ? 'success' : 'warning'}>{hasCurrentVerifiedEvidence ? '含当前有效的已验证证据' : '尚无当前有效的运行证据'}</StatusPill>
    </div>
    <div className="mt-4 space-y-3">{artifacts.map((artifact) => {
      const related = evidenceByArtifact.get(artifact.id) ?? []
      const currentStatus = artifact.status === 'verified' && !related.some(item => isCurrentEvidence(item, now)) ? 'unverified' : artifact.status
      const sourceLabel = revisionNumbers?.[artifact.revision_id] ? `r${revisionNumbers[artifact.revision_id]}` : artifact.revision_id
      return <article key={artifact.id} aria-label={`教学工件 ${sourceLabel}`} className="rounded-md border border-border p-4 text-sm">
        <p className="mb-2 text-xs text-muted-foreground">来源修订：{sourceLabel}{artifact.revision_id === currentSourceRevisionId ? ' · 当前稿件的代码来源' : ' · 历史修订'}</p>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"><div className="flex items-center gap-2"><FileCode2 className="h-4 w-4" /><span className="font-medium">{artifact.language} 教学工件</span><span className="text-xs text-muted-foreground">入口：{artifact.entrypoint ?? '未声明'}</span></div><StatusPill tone={statusTone(currentStatus)}>{statusLabel[currentStatus]}</StatusPill></div>
        <dl className="mt-3 grid gap-x-5 gap-y-2 text-xs sm:grid-cols-2"><div><dt className="text-muted-foreground">代码树指纹</dt><dd className="font-mono">{shortHash(artifact.artifact_hash)}</dd></div><div><dt className="text-muted-foreground">命令清单指纹</dt><dd className="font-mono">{shortHash(artifact.manifest_hash)}</dd></div></dl>
        <ul className="mt-3 list-disc space-y-1 pl-5 text-xs text-muted-foreground">{artifact.limitations_json.map((limitation) => <li key={limitation}>{limitation}</li>)}</ul>
        {chapterId && artifact.kind === 'teaching_implementation' ? <ArtifactVerificationControls chapterId={chapterId} artifactId={artifact.id} onEvidenceLoaded={setRefreshed} /> : null}
        {related.length === 0 ? <p className="mt-3 rounded bg-muted/40 px-3 py-2 text-xs text-muted-foreground">尚未采集运行证据。请不要把此工件的讲解或预期输出标为“已实测”。</p> : <ul className="mt-3 space-y-2">{related.map((item) => { const current = isCurrentEvidence(item, now); const displayStatus = evidenceStatus(item, now); const expiryReason = item.status === 'verified' && !current ? '运行证据已过期，需重新验证后才能作为当前事实使用。' : null; const summary = observationSummary(item.structured_output); return <li key={item.id} className="rounded border border-border bg-muted/20 p-3 text-xs"><div className="flex flex-wrap items-center justify-between gap-2"><span>{item.kind === 'terminal_output' ? '终端输出' : item.kind === 'browser_page' ? '浏览器页面' : item.kind === 'ide_capture' ? 'IDE 人工截图' : '图像证据'}</span><StatusPill tone={statusTone(displayStatus)}>{statusLabel[displayStatus]}</StatusPill></div><p className="mt-2 text-muted-foreground">工具：{item.tool_name ?? '未记录'}{item.tool_version ? `（${item.tool_version}）` : ''}；工具链：{item.toolchain_version ?? '未记录'}；采集时间：{item.captured_at ? new Date(item.captured_at).toLocaleString() : '未采集'}。</p>{summary ? <p className="mt-1 text-muted-foreground">{summary}</p> : null}{item.stale_reason || expiryReason ? <p className="mt-2 flex items-start gap-1 text-[var(--warning)]"><CircleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" />{item.stale_reason || expiryReason}</p> : null}</li> })}</ul>}
        {assets.filter((item) => item.evidence_id && related.some((evidenceItem) => evidenceItem.id === item.evidence_id)).map((item) => <div key={item.id} className="mt-3 rounded border border-dashed border-border p-3 text-xs"><p className="font-medium">{item.kind === 'screenshot' ? item.generation_method === 'isolated_playwright_probe' ? '自动浏览器截图资产' : '人工截图资产' : item.kind}</p><p className="mt-1 text-muted-foreground">替代文本：{item.alt_text}</p><p className="mt-1 text-muted-foreground">视觉用途：{visualPurposeLabel[item.visual_purpose]}</p><p className="mt-1 text-muted-foreground">来源：{item.source}；方式：{item.generation_method}；权利：{item.rights_status}</p><p className="mt-1 break-all font-mono text-muted-foreground">{item.stable_ref}</p></div>)}
        {related.map(item => <RuntimeEvidenceDetails key={item.id} evidence={item} />)}
      </article>
    })}</div>
  </section>
}
