import type { BlogNode } from '@/store/blogStore'

export function syncExpandedNodesWithSelection(
  expandedNodes: Set<string>,
  selectedBlog: BlogNode | null,
): Set<string> {
  if (!selectedBlog) {
    return expandedNodes
  }

  const nextExpanded = new Set(expandedNodes)

  if (selectedBlog.children.length > 0) {
    nextExpanded.add(selectedBlog.id)
    return nextExpanded
  }

  if (selectedBlog.parent_id) {
    nextExpanded.add(selectedBlog.parent_id)
  }

  return nextExpanded
}
