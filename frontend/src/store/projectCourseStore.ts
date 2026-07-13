import { create } from 'zustand'
import { flattenBlueprintChapters, type BlueprintChapterUpdate, type ProjectCourse, type ProjectCourseBlueprint } from '@/lib/projectCourse'
import { downloadTaskArtifact, projectCourseService, streamProjectCourseTask, type CreateProjectCourseInput } from '@/services/projectCourse'

interface ProjectCourseState {
  course: ProjectCourse | null
  chapters: BlueprintChapterUpdate[]
  isLoading: boolean
  error: string | null
  taskId: string | null
  taskMessage: string | null
  packageMessage: string | null
  create: (input: CreateProjectCourseInput) => Promise<void>
  load: (courseId: string) => Promise<void>
  updateChapter: (chapterId: string, updates: Partial<Omit<BlueprintChapterUpdate, 'chapter_id'>>) => void
  saveBlueprint: () => Promise<void>
  approve: () => Promise<void>
  packageCourse: () => Promise<void>
  reset: () => void
}

export const useProjectCourseStore = create<ProjectCourseState>((set, get) => ({
  course: null,
  chapters: [],
  isLoading: false,
  error: null,
  taskId: null,
  taskMessage: null,
  packageMessage: null,
  create: async (input) => {
    set({ isLoading: true, error: null, taskId: null })
    try {
      const response = await projectCourseService.create(input)
      const course = response.data.course
      const blueprint = course.blueprint_json as ProjectCourseBlueprint
      set({ course, chapters: blueprint?.volumes ? flattenBlueprintChapters(blueprint) : [], taskId: response.data.task_id, taskMessage: '分析任务已排队', isLoading: false })
      void streamProjectCourseTask(response.data.task_id, (message) => set({ taskMessage: message }))
        .then(() => get().load(course.id))
        .catch((error) => set({ taskMessage: error instanceof Error ? error.message : '课程任务流已断开' }))
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '创建项目课程失败' })
    }
  },
  load: async (courseId) => {
    set({ isLoading: true, error: null })
    try {
      const response = await projectCourseService.get(courseId)
      const blueprint = response.data.blueprint_json as ProjectCourseBlueprint
      set({ course: response.data, chapters: blueprint?.volumes ? flattenBlueprintChapters(blueprint) : [], isLoading: false })
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '获取项目课程失败' })
    }
  },
  updateChapter: (chapterId, updates) => set((state) => ({ chapters: state.chapters.map((chapter) => chapter.chapter_id === chapterId ? { ...chapter, ...updates } : chapter) })),
  saveBlueprint: async () => {
    const { course, chapters } = get()
    if (!course) return
    const response = await projectCourseService.updateBlueprint(course.id, course.blueprint_version, chapters)
    set((state) => ({ course: state.course ? { ...state.course, blueprint_version: response.data.blueprint_version } : null }))
  },
  approve: async () => {
    const { course } = get()
    if (!course) return
    const response = await projectCourseService.approve(course.id, course.blueprint_version)
    set((state) => ({ course: state.course ? { ...state.course, status: response.data.status as ProjectCourse['status'] } : null, taskId: response.data.task_id ?? null, taskMessage: response.data.task_id ? '正文生成任务已排队' : null }))
    if (response.data.task_id) {
      void streamProjectCourseTask(response.data.task_id, (message) => set({ taskMessage: message }))
        .then(() => get().load(course.id))
        .catch((error) => set({ taskMessage: error instanceof Error ? error.message : '课程生成任务流已断开' }))
    }
  },
  packageCourse: async () => {
    const { course } = get()
    if (!course) return
    set({ packageMessage: '课程包任务已提交' })
    try {
      const response = await projectCourseService.package(course.id)
      await streamProjectCourseTask(response.data.task_id, (message) => set({ packageMessage: message }))
      await downloadTaskArtifact(response.data.task_id, 'project-course.zip')
      set({ packageMessage: '课程包已准备完成' })
    } catch (error) {
      set({ packageMessage: error instanceof Error ? error.message : '课程包生成失败' })
    }
  },
  reset: () => set({ course: null, chapters: [], isLoading: false, error: null, taskId: null, taskMessage: null, packageMessage: null }),
}))
