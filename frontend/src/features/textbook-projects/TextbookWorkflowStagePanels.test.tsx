// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { GenerationStagePanel } from '@/features/generation/GenerationStagePanel'
import { ManuscriptStagePanel } from '@/features/manuscript/ManuscriptStagePanel'
import { SourceLibraryPanel } from '@/features/source-library/SourceLibraryPanel'

describe('Textbook workflow stage summaries', () => {
  afterEach(cleanup)

  it('states source, generation and revision safeguards before the corresponding actions', () => {
    render(<><SourceLibraryPanel /><GenerationStagePanel /><ManuscriptStagePanel /></>)

    expect(screen.getByRole('heading', { name: '主资料与官方补充资料' })).toBeTruthy()
    expect(screen.getByText('仅主资料与人工确认的官方资料可进入生成证据集。第三方文章不会自动混入教材。')).toBeTruthy()
    expect(screen.getByRole('heading', { name: '样章与质量门禁' })).toBeTruthy()
    expect(screen.getByText(/缓存与真实用量在工作器执行后才能确认/)).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Revision、证据与人工批准' })).toBeTruthy()
    expect(screen.getByText('批准稿不会被 AI 静默覆盖；生成任务只能产出候选稿。')).toBeTruthy()
    expect(screen.queryByText('待导入')).toBeNull()
    expect(screen.queryByText('待蓝图')).toBeNull()
    expect(screen.queryByText('待章节')).toBeNull()
  })
})
