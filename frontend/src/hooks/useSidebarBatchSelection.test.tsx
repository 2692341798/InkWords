import { describe, expect, it } from 'vitest'

import type { BlogNode } from '@/store/blogStore'
import {
  createSidebarBatchSelectionState,
  deriveSelectedSeriesRoots,
  toggleSidebarBatchModeState,
} from './useSidebarBatchSelection'

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

describe('useSidebarBatchSelection helpers', () => {
  it('starts with an empty batch selection state', () => {
    const state = createSidebarBatchSelectionState()

    expect(state.isBatchMode).toBe(false)
    expect(Array.from(state.selectedForExport)).toEqual([])
  })

  it('toggles batch mode and clears previous selections for a fresh toolbar session', () => {
    const nextState = toggleSidebarBatchModeState({
      isBatchMode: false,
      selectedForExport: new Set<string>(['series-1', 'chapter-1']),
    })

    expect(nextState.isBatchMode).toBe(true)
    expect(Array.from(nextState.selectedForExport)).toEqual([])

    const closedState = toggleSidebarBatchModeState(nextState)

    expect(closedState.isBatchMode).toBe(false)
    expect(Array.from(closedState.selectedForExport)).toEqual([])
  })

  it('derives only selected series roots from the selected ids', () => {
    const seriesRoot = createBlogNode({
      id: 'series-2',
      children: [
        createBlogNode({
          id: 'chapter-2',
          parent_id: 'series-2',
        }),
      ],
    })
    const singleBlog = createBlogNode({
      id: 'single-1',
      title: '单篇博客',
    })

    const selectedSeriesRoots = deriveSelectedSeriesRoots(
      [seriesRoot, singleBlog],
      new Set<string>(['series-2', 'single-1']),
    )

    expect(selectedSeriesRoots.map((node) => node.id)).toEqual(['series-2'])
  })
})
