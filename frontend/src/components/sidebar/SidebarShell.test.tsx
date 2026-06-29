import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'

import { SidebarShell } from './SidebarShell'

describe('SidebarShell', () => {
  it('renders header, content and footer slots inside the sidebar frame', () => {
    const html = renderToStaticMarkup(
      <SidebarShell
        header={<div>顶部区域</div>}
        footer={<div>底部区域</div>}
      >
        <div>内容区域</div>
      </SidebarShell>,
    )

    expect(html).toContain('顶部区域')
    expect(html).toContain('内容区域')
    expect(html).toContain('底部区域')
    expect(html).toContain('w-72')
    expect(html).toContain('print:hidden')
  })
})
