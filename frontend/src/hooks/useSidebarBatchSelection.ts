import { useCallback, useMemo, useState } from 'react'

import { toggleBlogSubtreeSelection } from '@/lib/sidebarSelection'
import type { BlogNode } from '@/store/blogStore'

export interface SidebarBatchSelectionState {
  isBatchMode: boolean
  selectedForExport: Set<string>
}

export interface SidebarBatchSelectionController extends SidebarBatchSelectionState {
  selectedSeriesRoots: BlogNode[]
  toggleNodeSelection: (node: BlogNode) => void
  toggleBatchMode: () => void
  closeBatchMode: () => void
  resetBatchSelection: () => void
}

export function createSidebarBatchSelectionState(): SidebarBatchSelectionState {
  return {
    isBatchMode: false,
    selectedForExport: new Set<string>(),
  }
}

export function toggleSidebarBatchModeState(
  state: SidebarBatchSelectionState,
): SidebarBatchSelectionState {
  return {
    isBatchMode: !state.isBatchMode,
    selectedForExport: new Set<string>(),
  }
}

export function deriveSelectedSeriesRoots(
  blogs: BlogNode[],
  selectedForExport: Set<string>,
): BlogNode[] {
  return blogs.filter((blog) => Boolean(blog.children?.length) && selectedForExport.has(blog.id))
}

/**
 * Why: Sidebar 容器同时承担历史树、批量操作和导出状态，
 * 如果批量选择状态继续散落在组件里，后续再拆树节点时会不断复制“清空/派生/切换”规则。
 */
export function useSidebarBatchSelection(blogs: BlogNode[]): SidebarBatchSelectionController {
  const [state, setState] = useState<SidebarBatchSelectionState>(() => createSidebarBatchSelectionState())

  const toggleNodeSelection = useCallback((node: BlogNode) => {
    // Why: 批量导出按树形语义工作，父节点勾选必须与整棵子树保持一致，
    // 这样系列导出、删除与后续状态提示才不会出现“父子勾选不一致”的歧义。
    setState((previous) => ({
      ...previous,
      selectedForExport: toggleBlogSubtreeSelection(previous.selectedForExport, node),
    }))
  }, [])

  const toggleBatchMode = useCallback(() => {
    setState((previous) => toggleSidebarBatchModeState(previous))
  }, [])

  const closeBatchMode = useCallback(() => {
    setState((previous) => ({
      ...previous,
      isBatchMode: false,
    }))
  }, [])

  const resetBatchSelection = useCallback(() => {
    setState(createSidebarBatchSelectionState())
  }, [])

  const selectedSeriesRoots = useMemo(
    () => deriveSelectedSeriesRoots(blogs, state.selectedForExport),
    [blogs, state.selectedForExport],
  )

  return {
    isBatchMode: state.isBatchMode,
    selectedForExport: state.selectedForExport,
    selectedSeriesRoots,
    toggleNodeSelection,
    toggleBatchMode,
    closeBatchMode,
    resetBatchSelection,
  }
}
