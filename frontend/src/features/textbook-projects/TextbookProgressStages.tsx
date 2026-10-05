import { StatusPill } from '@/components/ui/workspace'
import type { TextbookProjectProgress, TextbookStageState } from '@/services/textbook'

const labels: Record<TextbookStageState['key'], string> = {
  sources: '资料', blueprint: '蓝图', sample: '样章审批', chapters: '章节', verification: '验证', learning: '学习', publication: '出版',
}

const tones: Record<TextbookStageState['status'], 'default' | 'brand' | 'warning' | 'success'> = {
  blocked: 'default', ready: 'brand', in_progress: 'warning', approved: 'success', unavailable: 'default',
}

const statusLabels: Record<TextbookStageState['status'], string> = {
  blocked: '被阻塞', ready: '可开始', in_progress: '进行中', approved: '已批准', unavailable: '暂不可用',
}

// TextbookProgressStages renders only facts returned by core-api, never client-side completion guesses.
export function TextbookProgressStages({ progress }: { progress: TextbookProjectProgress | null }) {
  if (!progress) return null
  return <section className="mt-6 grid gap-2 sm:grid-cols-7" aria-label="教材工作台阶段">{progress.stages.map((stage) => <div key={stage.key} className="rounded-lg border border-border px-3 py-3 text-center text-xs"><span className="block">{labels[stage.key]}</span><div className="mt-1"><StatusPill tone={tones[stage.status]}>{statusLabels[stage.status]}</StatusPill></div><p className="mt-2 text-[10px] leading-4 text-muted-foreground">{stage.reason}</p></div>)}</section>
}
