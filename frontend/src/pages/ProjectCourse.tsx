import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, Panel } from '@/components/ui/workspace'
import { BlueprintWorkspace } from '@/components/project-course/BlueprintWorkspace'
import { useProjectCourseStore } from '@/store/projectCourseStore'

export function ProjectCourse() {
  const [courseID, setCourseID] = useState('')
  const load = useProjectCourseStore((state) => state.load)
  const course = useProjectCourseStore((state) => state.course)

  return (
    <PageShell wide>
      <PageHeader title="项目精通课程" description="围绕固定源码快照、证据引用和累积练习掌握一个 GitHub 项目。" />
      {!course ? (
        <Panel className="mt-6 p-6">
          <form className="flex flex-col gap-3 sm:flex-row" onSubmit={(event) => { event.preventDefault(); if (courseID.trim()) void load(courseID.trim()) }}>
            <label className="sr-only" htmlFor="project-course-id">课程 ID</label>
            <input id="project-course-id" className="min-w-0 flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm" placeholder="输入课程 ID" value={courseID} onChange={(event) => setCourseID(event.target.value)} />
            <Button type="submit">加载蓝图</Button>
          </form>
        </Panel>
      ) : <div className="mt-6"><BlueprintWorkspace /></div>}
    </PageShell>
  )
}
