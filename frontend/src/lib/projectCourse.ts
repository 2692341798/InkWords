export type ProjectCourseAudience = 'foundation' | 'programming' | 'stack_familiar'
export type ProjectCourseStatus = 'draft' | 'analyzing' | 'awaiting_approval' | 'approved' | 'generating' | 'completed' | 'blocked' | 'failed'

export interface ProjectCourseChapter {
  chapter_id: string
  title: string
  sort: number
  enabled: boolean
  chapter_type: string
  prerequisite_ids: string[]
  evidence_ids: string[]
}

export interface ProjectCourseVolume {
  volume_id: string
  title: string
  sort: number
  chapters: ProjectCourseChapter[]
}

export interface ProjectCourseBlueprint {
  course_id: string
  blueprint_version: number
  commit_sha: string
  audience_level: ProjectCourseAudience
  volumes: ProjectCourseVolume[]
}

export interface ProjectCourse {
  id: string
  repository_url: string
  requested_ref: string
  resolved_commit_sha: string
  audience_level: ProjectCourseAudience
  status: ProjectCourseStatus
  blueprint_version: number
  blueprint_json: ProjectCourseBlueprint | Record<string, unknown>
  coverage_json: Record<string, unknown>
}

export interface BlueprintChapterUpdate {
  chapter_id: string
  title: string
  sort: number
  enabled: boolean
}

export function buildBlueprintUpdateRequest(course: ProjectCourse, chapters: BlueprintChapterUpdate[]) {
  return {
    expected_version: course.blueprint_version,
    chapters: chapters.map(({ chapter_id, title, sort, enabled }) => ({ chapter_id, title, sort, enabled })),
  }
}

export function flattenBlueprintChapters(blueprint: ProjectCourseBlueprint): BlueprintChapterUpdate[] {
  return blueprint.volumes.flatMap((volume) => volume.chapters.map(({ chapter_id, title, sort, enabled }) => ({ chapter_id, title, sort, enabled })))
}
