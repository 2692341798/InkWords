import { Panel, SectionHeader } from '@/components/ui/workspace'

export function GenerationStagePanel() {
  return <Panel className="p-5"><SectionHeader eyebrow="生成" title="样章与质量门禁" description="入队前会展示冻结证据数、保守 Token 估算和当前生成目标；缓存与真实用量在工作器执行后才能确认。" /><p className="mt-4 text-sm text-muted-foreground">批准蓝图后，在章节中预检并明确发起生成；候选稿仍需审阅和批准。</p></Panel>
}
