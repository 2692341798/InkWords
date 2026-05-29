import { describe, expect, it } from 'vitest'

import type { BlogNode } from '@/store/blogStore'
import { toggleBlogSubtreeSelection } from './sidebarSelection'

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

describe('toggleBlogSubtreeSelection', () => {
  it('selects the clicked parent and all of its descendants together', () => {
    const node = createBlogNode({
      id: 'parent-1',
      children: [
        createBlogNode({
          id: 'child-1',
          parent_id: 'parent-1',
        }),
        createBlogNode({
          id: 'child-2',
          parent_id: 'parent-1',
        }),
      ],
    })

    const selected = toggleBlogSubtreeSelection(new Set<string>(), node)

    expect(Array.from(selected).sort()).toEqual(['child-1', 'child-2', 'parent-1'])
  })

  it('clears only the clicked subtree while preserving unrelated selections', () => {
    const node = createBlogNode({
      id: 'parent-2',
      children: [
        createBlogNode({
          id: 'child-3',
          parent_id: 'parent-2',
        }),
      ],
    })

    const selected = toggleBlogSubtreeSelection(
      new Set<string>(['parent-2', 'child-3', 'other-blog']),
      node,
    )

    expect(Array.from(selected)).toEqual(['other-blog'])
  })
})
