import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useBlogStore, type BlogNode } from './blogStore'
import { blogService } from '@/services/blog'

vi.mock('@/services/blog', () => ({
  blogService: {
    fetchBlogTree: vi.fn(),
    createDraftBlog: vi.fn(),
    updateBlog: vi.fn(),
    batchDeleteBlogs: vi.fn(),
  },
}))

const mockedBlogService = vi.mocked(blogService)

const makeBlog = (overrides: Partial<BlogNode> = {}): BlogNode => ({
  id: 'blog-1',
  title: 'InkWords Guide',
  content: 'content',
  source_type: 'manual',
  status: 1,
  chapter_sort: 1,
  parent_id: null,
  created_at: '2026-05-28T00:00:00Z',
  updated_at: '2026-05-28T00:00:00Z',
  children: [],
  ...overrides,
})

describe('useBlogStore', () => {
  beforeEach(() => {
    useBlogStore.setState({
      blogs: [],
      isLoading: false,
      selectedBlog: null,
      currentView: 'home-entry',
    })
    vi.clearAllMocks()
    vi.stubGlobal(
      'fetch',
      vi.fn(() => {
        throw new Error('unexpected direct fetch')
      }),
    )
  })

  it('delegates blog tree loading to blogService', async () => {
    mockedBlogService.fetchBlogTree.mockResolvedValue([makeBlog()])

    await useBlogStore.getState().fetchBlogs()

    expect(mockedBlogService.fetchBlogTree).toHaveBeenCalledTimes(1)
    expect(useBlogStore.getState().blogs).toEqual([makeBlog()])
    expect(useBlogStore.getState().isLoading).toBe(false)
  })

  it('delegates draft creation to blogService and selects the new draft', async () => {
    const draft = makeBlog({ id: 'draft-1', title: '新草稿' })
    mockedBlogService.createDraftBlog.mockResolvedValue(draft)

    await expect(useBlogStore.getState().createDraftBlog()).resolves.toEqual(draft)

    expect(mockedBlogService.createDraftBlog).toHaveBeenCalledTimes(1)
    expect(useBlogStore.getState().selectedBlog?.id).toBe('draft-1')
    expect(useBlogStore.getState().blogs[0]?.id).toBe('draft-1')
  })

  it('delegates updates after applying the optimistic state change', async () => {
    const existing = makeBlog()
    mockedBlogService.updateBlog.mockResolvedValue(undefined)
    useBlogStore.setState({ blogs: [existing], selectedBlog: existing })

    await useBlogStore.getState().updateBlog('blog-1', {
      title: 'Updated title',
      content: 'Updated content',
    })

    expect(mockedBlogService.updateBlog).toHaveBeenCalledWith('blog-1', {
      title: 'Updated title',
      content: 'Updated content',
    })
    expect(useBlogStore.getState().selectedBlog?.title).toBe('Updated title')
  })

  it('delegates batch delete then refreshes the list', async () => {
    const selected = makeBlog({ id: 'blog-2' })
    mockedBlogService.batchDeleteBlogs.mockResolvedValue(undefined)
    mockedBlogService.fetchBlogTree.mockResolvedValue([])
    useBlogStore.setState({ blogs: [selected], selectedBlog: selected })

    await useBlogStore.getState().batchDeleteBlogs(['blog-2'])

    expect(mockedBlogService.batchDeleteBlogs).toHaveBeenCalledWith(['blog-2'])
    expect(mockedBlogService.fetchBlogTree).toHaveBeenCalledTimes(1)
    expect(useBlogStore.getState().selectedBlog).toBeNull()
  })
})
