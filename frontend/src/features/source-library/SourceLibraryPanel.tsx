import { Panel, SectionHeader } from '@/components/ui/workspace'

export function SourceLibraryPanel() {
  return <Panel className="p-5"><SectionHeader eyebrow="资料" title="主资料与官方补充资料" description="登记资料后，在下方导入固定快照；已解析资料库展示可供引用的文档和片段。" /><p className="mt-4 text-sm text-muted-foreground">仅主资料与人工确认的官方资料可进入生成证据集。第三方文章不会自动混入教材。</p></Panel>
}
