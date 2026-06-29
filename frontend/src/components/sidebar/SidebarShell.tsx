import type { ReactNode } from 'react'

interface SidebarShellProps {
  header: ReactNode
  footer: ReactNode
  children: ReactNode
}

/**
 * Why: Sidebar 顶部入口、中部历史区和底部导航会持续演进，
 * 先抽出稳定壳层，后续局部拆分时就不会反复复制外层布局样式。
 */
export function SidebarShell({ header, footer, children }: SidebarShellProps) {
  return (
    <div className="flex max-h-[44vh] w-full shrink-0 flex-col border-b border-sidebar-border bg-sidebar text-sidebar-foreground transition-colors duration-200 print:hidden lg:max-h-none lg:w-80 lg:border-b-0 lg:border-r">
      <div className="shrink-0">{header}</div>
      <div className="flex-1 flex flex-col min-h-0 overflow-hidden">{children}</div>
      <div className="mt-auto shrink-0">{footer}</div>
    </div>
  )
}
