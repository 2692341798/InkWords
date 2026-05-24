import { describe, expect, it } from 'vitest'
import { buildAnalyzeGitRequest } from './useProjectAnalyzer'
import {
  buildSeriesGenerateRequest,
  buildSingleGenerateRequest,
} from './useSeriesGenerator'

describe('stream request builders', () => {
  it('includes scenario mode in the analyze request payload', () => {
    expect(
      buildAnalyzeGitRequest(
        'https://github.com/inkwords/demo',
        ['frontend/src', 'backend/internal'],
        'beginner_walkthrough',
      ),
    ).toEqual({
      git_url: 'https://github.com/inkwords/demo',
      selected_modules: ['frontend/src', 'backend/internal'],
      scenario_mode: 'beginner_walkthrough',
    })
  })

  it('includes scenario mode in the series generate request payload', () => {
    expect(
      buildSeriesGenerateRequest({
        sourceType: 'git',
        gitUrl: 'https://github.com/inkwords/demo',
        sourceContent: 'repo summary',
        seriesTitle: 'InkWords 入门',
        outline: [{ sort: 1, title: '准备环境', summary: '安装依赖' }],
        parentBlogId: 'parent-1',
        scenarioMode: 'open_book_exam_review',
      }),
    ).toEqual({
      source_type: 'git',
      git_url: 'https://github.com/inkwords/demo',
      source_content: 'repo summary',
      series_title: 'InkWords 入门',
      outline: [{ sort: 1, title: '准备环境', summary: '安装依赖' }],
      parent_id: 'parent-1',
      scenario_mode: 'open_book_exam_review',
    })
  })

  it('includes scenario mode in the single generate request payload', () => {
    expect(
      buildSingleGenerateRequest('parsed pdf content', 'ebook_interpretation'),
    ).toEqual({
      source_type: 'file',
      source_content: 'parsed pdf content',
      outline: [],
      scenario_mode: 'ebook_interpretation',
    })
  })
})
