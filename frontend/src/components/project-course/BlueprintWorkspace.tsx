import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import { useProjectCourseStore } from '@/store/projectCourseStore'

export function BlueprintWorkspace() {
  const { course, chapters, isLoading, error, taskMessage, packageMessage, updateChapter, saveBlueprint, approve, packageCourse } = useProjectCourseStore()

  const coverage = course?.coverage_json ?? {}
  const coverageItems = ['modules', 'main_flows', 'technologies', 'files'].map((key) => {
    const items = Array.isArray(coverage[key]) ? coverage[key] as Array<{ covered?: boolean }> : []
    const covered = items.filter((item) => item.covered).length
    return { key, covered, total: items.length }
  })

  if (isLoading) return <Panel className="p-6 text-sm text-muted-foreground">正在加载课程蓝图...</Panel>
  if (error) return <Panel className="p-6 text-sm text-destructive" role="alert">{error}</Panel>
  if (!course) return <Panel className="p-6 text-sm text-muted-foreground">输入课程 ID 后加载蓝图。</Panel>

  const statusTone = course.status === 'approved' || course.status === 'completed'
    ? 'success'
    : course.status === 'partially_blocked' || course.status === 'blocked' || course.status === 'failed'
      ? 'warning'
      : 'brand'

  return (
    <div className="space-y-6">
      <Panel className="p-6">
        <SectionHeader
          eyebrow="固定源码快照"
          title={course.repository_url}
          description={`ref: ${course.requested_ref}`}
          action={<StatusPill tone={statusTone}>{course.status}</StatusPill>}
        />
        <p className="mt-4 break-all font-mono text-xs text-muted-foreground">commit: {course.resolved_commit_sha}</p>
        {course.status === 'analyzing' && taskMessage ? <p className="mt-3 text-sm text-muted-foreground" role="status">课程任务：{taskMessage}</p> : null}
      </Panel>

      <Panel className="p-6">
        <SectionHeader title="课程蓝图" description="只能编辑章节标题、顺序和启用状态；证据、学习目标与事实由系统维护。" />
        <div className="mt-4 grid gap-2 text-xs text-muted-foreground sm:grid-cols-4" aria-label="覆盖概览">
          {coverageItems.map((item) => <div key={item.key} className="rounded-md border border-border/70 px-3 py-2"><span className="capitalize">{item.key.replace('_', ' ')}</span><strong className="ml-2 text-foreground">{item.total ? `${item.covered}/${item.total}` : '—'}</strong></div>)}
        </div>
        <div className="mt-5 space-y-3">
          {chapters.map((chapter) => (
            <div key={chapter.chapter_id} className="grid gap-3 rounded-xl border border-border/70 bg-background/60 p-4 md:grid-cols-[minmax(0,1fr)_90px_auto] md:items-center">
              <label className="min-w-0">
                <span className="sr-only">章节标题</span>
                <input className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm" value={chapter.title} onChange={(event) => updateChapter(chapter.chapter_id, { title: event.target.value })} />
                <span className="mt-1 block text-xs text-muted-foreground">{chapter.chapter_id}</span>
              </label>
              <label className="text-sm text-muted-foreground">
                <span className="sr-only">排序</span>
                <input type="number" className="w-full rounded-md border border-border bg-background px-3 py-2" value={chapter.sort} onChange={(event) => updateChapter(chapter.chapter_id, { sort: Number(event.target.value) })} />
              </label>
              <label className="flex items-center gap-2 text-sm text-muted-foreground">
                <input type="checkbox" checked={chapter.enabled} onChange={(event) => updateChapter(chapter.chapter_id, { enabled: event.target.checked })} />
                启用
              </label>
            </div>
          ))}
        </div>
        <div className="mt-6 flex flex-wrap justify-end gap-3">
          <Button variant="outline" onClick={() => void saveBlueprint()} disabled={course.status === 'approved'}>保存蓝图</Button>
          <Button onClick={() => void approve()} disabled={course.status !== 'awaiting_approval'}>批准并继续生成</Button>
          <Button variant="outline" onClick={() => void packageCourse()} disabled={course.status !== 'completed'}>打包课程 ZIP</Button>
        </div>
        {packageMessage ? <p className="mt-3 text-right text-xs text-muted-foreground" role="status">{packageMessage}</p> : null}
      </Panel>
    </div>
  )
}
