import { describe, expect, it } from 'vitest'
import { buildBlueprintUpdateRequest, flattenBlueprintChapters, type ProjectCourseStatus } from './projectCourse'

describe('projectCourse blueprint contracts', () => {
  it('accepts a partially blocked generation status returned by the backend', () => {
    const status: ProjectCourseStatus = 'partially_blocked'
    expect(status).toBe('partially_blocked')
  })

  const blueprint = {
    course_id: 'course-1', blueprint_version: 3, commit_sha: 'abc', audience_level: 'programming' as const,
    volumes: [{ volume_id: 'v1', title: '基础', sort: 1, chapters: [{ chapter_id: 'c1', title: '地图', sort: 1, enabled: true, chapter_type: 'project_map', prerequisite_ids: [], evidence_ids: ['ev-1'] }] }],
  }

  it('flattens chapters into the only editable fields', () => {
    expect(flattenBlueprintChapters(blueprint)).toEqual([{ chapter_id: 'c1', title: '地图', sort: 1, enabled: true }])
  })

  it('includes the optimistic-lock version and strips evidence fields', () => {
    expect(buildBlueprintUpdateRequest({ id: 'course-1', blueprint_version: 3 } as never, [{ chapter_id: 'c1', title: '新标题', sort: 2, enabled: false, }])).toEqual({ expected_version: 3, chapters: [{ chapter_id: 'c1', title: '新标题', sort: 2, enabled: false }] })
  })
})
