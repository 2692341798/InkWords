import { describe, expect, it } from 'vitest'

import type { BlogNode } from '@/store/blogStore'
import { syncExpandedNodesWithSelection } from './blogTreeSelection'

function createBlogNode(overrides: Partial<BlogNode> = {}): BlogNode {
  return {
    id: 'blog-1',
    title: '系列导读',
    content: '',
    source_type: 'file',
    status: 1,
    chapter_sort: 0,
    parent_id: null,
    created_at: '2026-05-25T00:00:00Z',
    updated_at: '2026-05-25T00:00:00Z',
    children: [],
    ...overrides,
  }
}

describe('syncExpandedNodesWithSelection', () => {
  it('expands the selected series parent so child blogs are visible in history', () => {
    const selectedBlog = createBlogNode({
      id: 'parent-1',
      children: [
        createBlogNode({
          id: 'child-1',
          title: '第一篇',
          parent_id: 'parent-1',
        }),
      ],
    })

    const expanded = syncExpandedNodesWithSelection(new Set<string>(), selectedBlog)

    expect(Array.from(expanded)).toEqual(['parent-1'])
  })

  it('expands the parent when a child blog is selected from the current task list', () => {
    const selectedBlog = createBlogNode({
      id: 'child-2',
      parent_id: 'parent-2',
      children: [],
    })

    const expanded = syncExpandedNodesWithSelection(new Set<string>(), selectedBlog)

    expect(Array.from(expanded)).toEqual(['parent-2'])
  })

  it('keeps the expanded set unchanged for standalone blogs', () => {
    const standalone = createBlogNode({
      id: 'solo-1',
      title: '单篇文章',
      children: [],
    })

    const original = new Set<string>(['existing-parent'])
    const expanded = syncExpandedNodesWithSelection(original, standalone)

    expect(Array.from(expanded)).toEqual(['existing-parent'])
  })
})
