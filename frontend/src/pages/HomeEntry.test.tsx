import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { blogStoreState, reviewStoreState } = vi.hoisted(() => ({
  blogStoreState: {
    blogs: [
      {
        id: 1,
        title: '最近博客任务',
        updated_at: '2026-06-01T08:00:00Z',
        parent_id: null,
      },
    ],
    fetchBlogs: vi.fn(),
    setCurrentView: vi.fn(),
  },
  reviewStoreState: {
    recommendationCard: null,
    isLoadingRecommendation: false,
    historyItems: [],
    isLoadingHistory: false,
    currentSession: {
      session_id: 'session-1',
      status: 'completed',
      mode: 'detailed_qa' as const,
      title: '已经完成的知识复习',
      opening_prompt: '总结一下',
      initial_hints: [],
      turn_index: 3,
    },
    setShouldResumeSessionOnOpen: vi.fn(),
    loadRecommendation: vi.fn(),
    loadHistory: vi.fn(),
  },
}))

vi.mock('@/store/blogStore', () => ({
  useBlogStore: () => blogStoreState,
}))

vi.mock('@/store/reviewStore', () => ({
  useReviewStore: () => reviewStoreState,
}))

vi.mock('@/components/shared/StepStrip', () => ({
  StepStrip: () => <div>StepStripStub</div>,
}))

import { HomeEntry } from './HomeEntry'

describe('HomeEntry', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not render the resume-review card copy for a completed review session', () => {
    const html = renderToStaticMarkup(<HomeEntry />)

    expect(html).not.toContain('会话仍可继续')
    expect(html).not.toContain('继续知识复习')
    expect(html).toContain('最近博客任务')
    expect(html).toContain('进入博客生成')
  })
})
