import type { BlueprintChapterUpdate, ProjectCourse } from '@/lib/projectCourse'
import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'
import { fetchEventSourceWithAuth } from './sse'

interface ApiResponse<T> { code: number; data: T; message?: string }

export interface CreateProjectCourseInput {
  repository_url: string
  requested_ref: string
  audience_level: 'foundation' | 'programming' | 'stack_familiar'
}

export interface CreateProjectCourseResponse {
  course: ProjectCourse
  task_id: string
  status: string
}

export const projectCourseService = {
  create(input: CreateProjectCourseInput) {
    return requestJson<ApiResponse<CreateProjectCourseResponse>>(apiRoutes.coreApi.projectCourses.collection, { method: 'POST', json: input, fallbackMessage: '创建项目课程失败' })
  },
  get(courseId: string) {
    return requestJson<ApiResponse<ProjectCourse>>(apiRoutes.coreApi.projectCourses.byId(courseId), { method: 'GET', fallbackMessage: '获取项目课程失败' })
  },
  updateBlueprint(courseId: string, expectedVersion: number, chapters: BlueprintChapterUpdate[]) {
    return requestJson<ApiResponse<{ blueprint_version: number }>>(apiRoutes.coreApi.projectCourses.blueprint(courseId), { method: 'PUT', json: { expected_version: expectedVersion, chapters }, fallbackMessage: '更新课程蓝图失败' })
  },
  approve(courseId: string, expectedVersion: number) {
    return requestJson<ApiResponse<{ status: string; task_id?: string }>>(apiRoutes.coreApi.projectCourses.approve(courseId), { method: 'POST', json: { expected_version: expectedVersion }, fallbackMessage: '批准课程蓝图失败' })
  },
}

export function streamProjectCourseTask(taskId: string, onPhase: (message: string) => void) {
  return fetchEventSourceWithAuth(apiRoutes.coreApi.tasks.stream(taskId), {
    method: 'GET',
    onmessage(event) {
      if (event.event === 'done') return
      try {
        const payload = JSON.parse(event.data) as { stage?: string; checkpoint?: string }
        if (payload.stage || payload.checkpoint) onPhase([payload.stage, payload.checkpoint].filter(Boolean).join(' · '))
      } catch {
        // Task event payloads are best-effort UI metadata; terminal status is reloaded from the API.
      }
    },
  })
}
