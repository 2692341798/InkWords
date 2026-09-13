import { Panel, SectionHeader } from '@/components/ui/workspace'

export function ManuscriptStagePanel() {
  return <Panel className="p-5"><SectionHeader eyebrow="母稿" title="Revision、证据与人工批准" description="章节正文会保留 revision、锁、差异和证据；候选稿必须由你明确应用。" /><p className="mt-4 text-sm text-muted-foreground">批准稿不会被 AI 静默覆盖；生成任务只能产出候选稿。</p></Panel>
}
