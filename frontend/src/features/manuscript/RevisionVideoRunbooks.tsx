import type { ChapterRevision } from '@/services/textbook'
import { VideoRunbookPanel } from './VideoRunbookPanel'

interface Props {
  candidate: ChapterRevision | null
  approved: ChapterRevision | null
}

// A projection-only correction can have identical Markdown. Keep both revision
// identities visible so the approval screen still exposes what will change.
export function RevisionVideoRunbooks({ candidate, approved }: Props) {
  return <>
    {candidate ? <section aria-label={`所选候选稿 r${candidate.revision_number} 视频教案`} className="mt-6 rounded-md border border-border p-4">
      <h3 className="font-medium">所选候选稿 r{candidate.revision_number} · 视频教案预览</h3>
      <p className="mt-1 text-sm text-muted-foreground">预览只对应所选候选修订，审阅状态见上方。查看教案不代表批准、采集完成或运行验证；正式导出仍使用批准稿。</p>
      <VideoRunbookPanel documentJson={candidate.document_json} />
    </section> : null}
    {approved ? <section aria-label={`批准稿 r${approved.revision_number} 视频教案`} className="mt-6 rounded-md border border-border p-4">
      <h3 className="font-medium">批准稿 r{approved.revision_number} · 视频教案</h3>
      <p className="mt-1 text-sm text-muted-foreground">这是当前批准修订保存的教案。候选预览不会修改它，也不会继承其采集或验证结果。</p>
      <VideoRunbookPanel documentJson={approved.document_json} />
    </section> : null}
  </>
}
