import type { BlueprintChapterUpdate, ProjectCourse } from '@/lib/projectCourse'
import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'

interface ApiResponse<T> { code: number; data: T; message?: string }

export interface CreateProjectCourseInput {
  repository_url: string
  requested_ref: string
  resolved_commit_sha: string
  audience_level: 'foundation' | 'programming' | 'stack_familiar'
}

export const projectCourseService = {
  create(input: CreateProjectCourseInput) {
    return requestJson<ApiResponse<ProjectCourse>>(apiRoutes.coreApi.projectCourses.collection, { method: 'POST', json: input, fallbackMessage: '创建项目课程失败' })
  },
  get(courseId: string) {
    return requestJson<ApiResponse<ProjectCourse>>(apiRoutes.coreApi.projectCourses.byId(courseId), { method: 'GET', fallbackMessage: '获取项目课程失败' })
  },
  updateBlueprint(courseId: string, expectedVersion: number, chapters: BlueprintChapterUpdate[]) {
    return requestJson<ApiResponse<{ blueprint_version: number }>>(apiRoutes.coreApi.projectCourses.blueprint(courseId), { method: 'PUT', json: { expected_version: expectedVersion, chapters }, fallbackMessage: '更新课程蓝图失败' })
  },
  approve(courseId: string, expectedVersion: number) {
    return requestJson<ApiResponse<{ status: string }>>(apiRoutes.coreApi.projectCourses.approve(courseId), { method: 'POST', json: { expected_version: expectedVersion }, fallbackMessage: '批准课程蓝图失败' })
  },
}
