import { create } from 'zustand'
import { flattenBlueprintChapters, type BlueprintChapterUpdate, type ProjectCourse, type ProjectCourseBlueprint } from '@/lib/projectCourse'
import { projectCourseService } from '@/services/projectCourse'

interface ProjectCourseState {
  course: ProjectCourse | null
  chapters: BlueprintChapterUpdate[]
  isLoading: boolean
  error: string | null
  load: (courseId: string) => Promise<void>
  updateChapter: (chapterId: string, updates: Partial<Omit<BlueprintChapterUpdate, 'chapter_id'>>) => void
  saveBlueprint: () => Promise<void>
  approve: () => Promise<void>
  reset: () => void
}

export const useProjectCourseStore = create<ProjectCourseState>((set, get) => ({
  course: null,
  chapters: [],
  isLoading: false,
  error: null,
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
    set((state) => ({ course: state.course ? { ...state.course, status: response.data.status as ProjectCourse['status'] } : null }))
  },
  reset: () => set({ course: null, chapters: [], isLoading: false, error: null }),
}))
