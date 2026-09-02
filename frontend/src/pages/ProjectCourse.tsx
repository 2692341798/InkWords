import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, Panel } from '@/components/ui/workspace'
import { BlueprintWorkspace } from '@/components/project-course/BlueprintWorkspace'
import { useProjectCourseStore } from '@/store/projectCourseStore'
import type { ProjectCourseAudience } from '@/lib/projectCourse'

export function ProjectCourse() {
  const [courseID, setCourseID] = useState('')
  const [repositoryURL, setRepositoryURL] = useState('https://github.com/2692341798/InkWords')
  const [requestedRef, setRequestedRef] = useState('main')
  const [audienceLevel, setAudienceLevel] = useState<ProjectCourseAudience>('programming')
  const load = useProjectCourseStore((state) => state.load)
  const create = useProjectCourseStore((state) => state.create)
  const course = useProjectCourseStore((state) => state.course)
  const taskId = useProjectCourseStore((state) => state.taskId)

  return (
    <PageShell wide>
      <PageHeader title="项目精通课程" description="围绕固定源码快照、证据引用和累积练习掌握一个 GitHub 项目。" />
      {!course ? (
        <Panel className="mt-6 p-6">
          <form className="grid gap-3" onSubmit={(event) => { event.preventDefault(); void create({ repository_url: repositoryURL.trim(), requested_ref: requestedRef.trim(), audience_level: audienceLevel }) }}>
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_180px]">
              <label className="text-sm"><span className="mb-1 block text-muted-foreground">GitHub 仓库</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={repositoryURL} onChange={(event) => setRepositoryURL(event.target.value)} /></label>
              <label className="text-sm"><span className="mb-1 block text-muted-foreground">源码 ref</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={requestedRef} onChange={(event) => setRequestedRef(event.target.value)} /></label>
              <label className="text-sm"><span className="mb-1 block text-muted-foreground">读者等级</span><select className="w-full rounded-md border border-border bg-background px-3 py-2" value={audienceLevel} onChange={(event) => setAudienceLevel(event.target.value as ProjectCourseAudience)}><option value="foundation">零基础</option><option value="programming">有编程基础</option><option value="stack_familiar">熟悉技术栈</option></select></label>
            </div>
            <div className="flex flex-wrap gap-3"><Button type="submit">开始分析并生成蓝图</Button><span className="self-center text-xs text-muted-foreground">先锁定 commit，再进入蓝图审批。</span></div>
          </form>
          {taskId ? <p className="mt-4 text-sm text-muted-foreground" role="status">分析任务已排队：{taskId}</p> : null}
          <div className="my-6 border-t border-border/70" />
          <form className="flex flex-col gap-3 sm:flex-row" onSubmit={(event) => { event.preventDefault(); if (courseID.trim()) void load(courseID.trim()) }}>
            <label className="sr-only" htmlFor="project-course-id">已有课程 ID</label>
            <input id="project-course-id" className="min-w-0 flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm" placeholder="已有课程 ID" value={courseID} onChange={(event) => setCourseID(event.target.value)} />
            <Button type="submit" variant="outline">加载已有蓝图</Button>
          </form>
        </Panel>
      ) : <div className="mt-6"><BlueprintWorkspace /></div>}
    </PageShell>
  )
}
