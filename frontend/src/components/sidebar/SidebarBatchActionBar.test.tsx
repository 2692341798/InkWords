import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'

import { SidebarBatchActionBar } from './SidebarBatchActionBar'

describe('SidebarBatchActionBar', () => {
  it('renders the selected count and batch actions in Chinese', () => {
    const html = renderToStaticMarkup(
      <SidebarBatchActionBar
        selectedCount={2}
        canExportSeries={true}
        isExporting={false}
        isDeleting={false}
        isSyncingSeriesToObsidian={false}
        isExportingPDF={false}
        onCancel={() => {}}
        onExportZip={() => {}}
        onExportSeriesPdf={() => {}}
        onSyncSeriesToObsidian={() => {}}
        onDelete={() => {}}
      />,
    )

    expect(html).toContain('已选 2 项')
    expect(html).toContain('取消')
    expect(html).toContain('导出 / 同步')
    expect(html).toContain('批量删除')
  })

  it('disables destructive and export actions when there is no valid selection or work is in progress', () => {
    const html = renderToStaticMarkup(
      <SidebarBatchActionBar
        selectedCount={0}
        canExportSeries={false}
        isExporting={true}
        isDeleting={false}
        isSyncingSeriesToObsidian={false}
        isExportingPDF={false}
        onCancel={() => {}}
        onExportZip={() => {}}
        onExportSeriesPdf={() => {}}
        onSyncSeriesToObsidian={() => {}}
        onDelete={() => {}}
      />,
    )

    expect(html).toContain('disabled=""')
  })
})
