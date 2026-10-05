import { ClipboardCheck, MonitorPlay } from 'lucide-react'
import { StatusPill } from '@/components/ui/workspace'
import type { VideoRunbookProjection, VideoRunbookStep } from '@/services/textbook'

interface VideoRunbookPanelProps {
  documentJson: Record<string, unknown>
}

const textFields: (keyof VideoRunbookStep)[] = ['tool', 'tool_version', 'start_state', 'action', 'shortcut_or_menu', 'input', 'expected_view', 'narration', 'capture_point', 'recovery', 'completion_signal']

const isRecord = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null && !Array.isArray(value)

function parseRunbook(documentJson: Record<string, unknown>): VideoRunbookProjection | null {
  const value = documentJson.video_runbook
  if (!isRecord(value) || value.format !== 'inkwords.video-runbook.v1' || typeof value.stack !== 'string' || typeof value.observation_goal !== 'string' || typeof value.manual_capture_pending !== 'boolean' || !['draft', 'unverified', 'verified', 'blocked'].includes(String(value.verification_status))) return null
  if (!isRecord(value.recommendation) || typeof value.recommendation.primary !== 'string' || typeof value.recommendation.alternative !== 'string' || typeof value.recommendation.reason !== 'string' || typeof value.recommendation.manual_capture_required !== 'boolean') return null
  if (!Array.isArray(value.steps) || !value.steps.every((step) => isRecord(step) && textFields.every((field) => typeof step[field] === 'string'))) return null
  if (!Array.isArray(value.capture_checklist) || !value.capture_checklist.every((item) => typeof item === 'string')) return null
  return value as unknown as VideoRunbookProjection
}

export function VideoRunbookPanel({ documentJson }: VideoRunbookPanelProps) {
  const runbook = parseRunbook(documentJson)
  if (!runbook) return null

  return <section className="mt-6 border-t border-border pt-5" aria-label="视频操作教案">
    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div>
        <h3 className="flex items-center gap-2 font-medium"><MonitorPlay className="h-4 w-4" />副屏视频操作教案</h3>
        <p className="mt-1 text-sm text-muted-foreground">按步骤录制即可：每一步都写明工具、操作、旁白、截图点和失败恢复。它是人工录制清单，不会自动操控 IDE 或把未录到的画面当作证据。</p>
      </div>
      <StatusPill tone={runbook.verification_status === 'verified' ? 'success' : 'warning'}>{runbook.verification_status === 'verified' ? '人工证据已验证' : '待人工采集证据'}</StatusPill>
    </div>
    <div className="mt-4 rounded-md border border-border bg-muted/30 p-3 text-sm">
      <p><span className="font-medium">推荐工具：</span>{runbook.recommendation.primary} <span className="text-muted-foreground">（备选：{runbook.recommendation.alternative}）</span></p>
      <p className="mt-1 text-muted-foreground">{runbook.recommendation.reason}</p>
      <p className="mt-2 text-xs text-muted-foreground">技术栈：{runbook.stack}；观察目标：{runbook.observation_goal}。</p>
    </div>
    <ol className="mt-4 space-y-3">{runbook.steps.map((step, index) => <li key={`${step.tool}-${index}`} className="rounded-md border border-border p-4 text-sm">
      <h4 className="font-medium">{index + 1}. {step.action}</h4>
      <dl className="mt-3 grid gap-x-5 gap-y-2 text-xs sm:grid-cols-2 [&>div]:min-w-0 [&_dd]:[overflow-wrap:anywhere]"><div><dt className="text-muted-foreground">工具与版本</dt><dd>{step.tool} · {step.tool_version}</dd></div><div><dt className="text-muted-foreground">快捷键或菜单</dt><dd>{step.shortcut_or_menu}</dd></div><div><dt className="text-muted-foreground">起始状态</dt><dd>{step.start_state}</dd></div><div><dt className="text-muted-foreground">输入</dt><dd>{step.input}</dd></div><div><dt className="text-muted-foreground">预期画面</dt><dd>{step.expected_view}</dd></div><div><dt className="text-muted-foreground">截图点</dt><dd>{step.capture_point}</dd></div><div><dt className="text-muted-foreground">旁白</dt><dd>{step.narration}</dd></div><div><dt className="text-muted-foreground">失败恢复</dt><dd>{step.recovery}</dd></div></dl>
      <p className="mt-3 rounded bg-background px-2 py-1 text-xs"><span className="font-medium">完成信号：</span>{step.completion_signal}</p>
    </li>)}</ol>
    <div className="mt-4 rounded-md border border-dashed border-border p-3 text-sm"><p className="flex items-center gap-2 font-medium"><ClipboardCheck className="h-4 w-4" />人工采集清单</p><ul className="mt-2 list-disc space-y-1 pl-5 text-muted-foreground">{runbook.capture_checklist.map((item) => <li key={item}>{item}</li>)}</ul></div>
  </section>
}
