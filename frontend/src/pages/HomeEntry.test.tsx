// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { blogStoreState, reviewStoreState, masteryServiceState } = vi.hoisted(() => ({
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
      session_outline: {
        summary: '已完成复习摘要',
        main_question: '这次复习的主线是什么？',
        core_concepts: ['主线'],
        process_steps: [],
        application_cases: [],
        checkpoints: ['总结一下主线'],
      },
      turn_index: 3,
    },
    setShouldResumeSessionOnOpen: vi.fn(),
    loadRecommendation: vi.fn(),
    loadHistory: vi.fn(),
  },
  masteryServiceState: {
    getDue: vi.fn(),
  },
}))

vi.mock('@/store/blogStore', () => ({
  useBlogStore: () => blogStoreState,
}))

vi.mock('@/store/reviewStore', () => ({
  useReviewStore: Object.assign(() => reviewStoreState, {
    getState: () => reviewStoreState,
  }),
}))

vi.mock('@/services/mastery', () => ({
  masteryService: masteryServiceState,
}))

vi.mock('@/components/shared/StepStrip', () => ({
  StepStrip: () => <div>StepStripStub</div>,
}))

import { HomeEntry } from './HomeEntry'

describe('HomeEntry', () => {
	afterEach(() => cleanup())

  beforeEach(() => {
    vi.clearAllMocks()
    reviewStoreState.loadRecommendation.mockResolvedValue(undefined)
    reviewStoreState.loadHistory.mockResolvedValue(undefined)
    masteryServiceState.getDue.mockResolvedValue({ tasks: [] })
  })

  it('checks due mastery work once when the home entry opens, without a polling loop', async () => {
    masteryServiceState.getDue.mockResolvedValue({
      tasks: [{ objective_id: 'objective-1', chapter_id: 'chapter-1', title: '理解 Gin 路由', skill: 'explain', due_at: '2026-01-01T10:00:00Z', reason: '需要巩固原理' }],
    })

    render(<HomeEntry />)

    expect(await screen.findByText('理解 Gin 路由')).toBeTruthy()
    await waitFor(() => expect(masteryServiceState.getDue).toHaveBeenCalledTimes(1))
    expect(screen.getByText('仅在打开首页时检查，不会后台提醒。')).toBeTruthy()
  })

  it('does not render the resume-review card copy for a completed review session', () => {
    render(<HomeEntry />)

    expect(screen.queryByText(/会话仍可继续/)).toBeNull()
    expect(screen.queryByText('继续知识复习')).toBeNull()
    expect(screen.getAllByText('最近博客任务')).toHaveLength(2)
    expect(screen.getAllByText('进入博客生成')).toHaveLength(3)
  })

  it('does not retry review bootstrap in a render loop and exposes a manual retry', async () => {
    reviewStoreState.loadRecommendation.mockRejectedValue(new Error('obsidian unavailable'))
    reviewStoreState.loadHistory.mockRejectedValue(new Error('obsidian unavailable'))

    render(<HomeEntry />)

    expect((await screen.findByRole('alert')).textContent).toContain('复习摘要暂时加载失败')
    await waitFor(() => {
      expect(reviewStoreState.loadRecommendation).toHaveBeenCalledTimes(1)
      expect(reviewStoreState.loadHistory).toHaveBeenCalledTimes(1)
    })

    screen.getByRole('button', { name: '重试加载' }).click()

    await waitFor(() => {
      expect(reviewStoreState.loadRecommendation).toHaveBeenCalledTimes(2)
      expect(reviewStoreState.loadHistory).toHaveBeenCalledTimes(2)
    })
  })

  it('uses the textbook workspace as the only project-course entry', () => {
    render(<HomeEntry />)

    fireEvent.click(screen.getByRole('button', { name: /教材项目/ }))

    expect(blogStoreState.setCurrentView).toHaveBeenCalledWith('textbook-projects')
    expect(screen.queryByRole('button', { name: /项目精通课程/ })).toBeNull()
  })
})
