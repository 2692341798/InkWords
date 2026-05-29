import type { BlogNode } from '@/store/blogStore'

function visitBlogSubtree(node: BlogNode, visitor: (current: BlogNode) => void) {
  visitor(node)
  node.children.forEach((child) => visitBlogSubtree(child, visitor))
}

/**
 * Why: Sidebar 在批量模式下需要保证“父节点勾选 = 整个子树一起勾选/取消”，
 * 这样用户导出系列时不会漏掉子文章，同时保留与其它无关勾选项的独立性。
 */
export function toggleBlogSubtreeSelection(previous: Set<string>, node: BlogNode) {
  const next = new Set(previous)
  const shouldSelect = !previous.has(node.id)

  visitBlogSubtree(node, (current) => {
    if (shouldSelect) {
      next.add(current.id)
      return
    }

    next.delete(current.id)
  })

  return next
}
