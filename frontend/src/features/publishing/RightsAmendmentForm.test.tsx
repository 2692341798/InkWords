// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { EditorialWorkspace, PublicationRightsItem } from '@/services/textbook'
import { RightsAmendmentForm } from './RightsAmendmentForm'
import { RightsPanel } from './RightsPanel'

const item: PublicationRightsItem = { id: 'base-1', build_id: 'build-1', project_id: 'project-1', subject_ref: 'chapter:r1', work_type: 'prose', rights_basis: '许可待核对', allowed_use: '本地审校', attribution: '保留署名', publication_status: 'pending', created_at: '' }

describe('rights amendments', () => {
  afterEach(cleanup)
  it('binds raster assessment to the frozen asset and keeps font version on retry', async () => {
    const save = vi.fn().mockRejectedValueOnce(new Error('临时失败')).mockResolvedValue(undefined)
    const context = { asset: { id: 'image-1', content_hash: `sha256:${'a'.repeat(64)}` }, fonts: [{ label: '测试字体', file_hash: `sha256:${'b'.repeat(64)}`, rights_item_id: 'font-1', amendment_id: 'font-version-2' }] }
    render(<RightsAmendmentForm item={{ ...item, subject_ref: 'asset:image-1', work_type: 'screenshot' }} previousId={null} manifestHash="sha256:frozen" disabled={false} assetFontContext={context} onSave={save} onClose={vi.fn()} />)
    for (const [label, value] of [['补证审阅者', '测试核对者'], ['补证理由', '补齐完整截图字体来源证据'], ['补证证据引用（每行一项）', 'fixture:font-inventory'], ['图片文字核对', 'identified_fonts'], ['图片字体核对范围', '检查全部界面和代码显示所用字体']]) fireEvent.change(screen.getByLabelText(label), { target: { value } })
    fireEvent.click(screen.getByRole('button', { name: '保存追加补证' }))
    expect(save).not.toHaveBeenCalled()
    expect(screen.getByRole('alert').textContent).toContain('请选出已核对的字体')
    fireEvent.click(screen.getByLabelText('测试字体'))
    fireEvent.click(screen.getByRole('button', { name: '保存追加补证' }))
    await screen.findByText('临时失败')
    fireEvent.click(screen.getByRole('button', { name: '保存追加补证' }))
    await waitFor(() => expect(save).toHaveBeenCalledTimes(2))
    expect(save.mock.calls[0][0]).toEqual(save.mock.calls[1][0])
    expect(save.mock.calls[0][0].asset_font_review).toMatchObject({ asset_id: 'image-1', content_hash: context.asset.content_hash, fonts: [{ rights_item_id: 'font-1', amendment_id: 'font-version-2', file_hash: context.fonts[0].file_hash }] })
  })
  it('preserves retry identity and predecessor after a failed save', async () => {
    const save = vi.fn().mockRejectedValueOnce(new Error('版本冲突，请刷新核对')).mockResolvedValue(undefined)
    const close = vi.fn()
    render(<RightsAmendmentForm item={item} previousId="amendment-1" manifestHash="sha256:frozen" disabled={false} onSave={save} onClose={close} />)
    fireEvent.change(screen.getByLabelText('补证审阅来源'), { target: { value: 'delegated_ai' } })
    for (const [label, value] of [['补证审阅者', 'Codex'], ['补证授权说明', '用户明确委托核对权利证据'], ['补证理由', '补齐固定来源许可证据引用'], ['补证证据引用（每行一项）', 'license:fixed-snapshot']]) fireEvent.change(screen.getByLabelText(label), { target: { value } })
    fireEvent.click(screen.getByRole('button', { name: '保存追加补证' }))
    await screen.findByRole('alert')
    expect(close).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '保存追加补证' }))
    await waitFor(() => expect(close).toHaveBeenCalledOnce())
    expect(save.mock.calls[0][0]).toEqual(save.mock.calls[1][0])
    expect(save.mock.calls[0][0]).toMatchObject({ base_item_id: item.id, previous_amendment_id: 'amendment-1', reviewer_kind: 'delegated_ai', publication_status: 'pending' })
    expect(save.mock.calls[0][0]).not.toHaveProperty('subject_ref')
  })
  it('uses effective state while retaining the original, and locks promoted builds', () => {
    const workspace = { build: { id: 'build-1', status: 'ready_for_review', manifest_hash: 'sha256:frozen' }, required_rights_subjects: [{ subject_ref: item.subject_ref, work_type: 'prose' }], rights_items: [item], rights_ledger: { contract_version: 'inkwords.rights-ledger.v1', selection_rule: 'explicit-predecessor-chain-v1', original_items: [item], amendments: [], effective_items: [{ ...item, publication_status: 'blocked' }] } } as unknown as EditorialWorkspace
    const props = { disabled: false, onAdd: vi.fn(), onAmend: vi.fn() }
    const view = render(<RightsPanel {...props} workspace={workspace} />)
    expect(screen.getByText('prose · blocked')).toBeTruthy()
    expect(screen.getByText('已登记，待补证')).toBeTruthy()
    expect(screen.queryByRole('button', { name: '登记此对象' })).toBeNull()
    expect(item.publication_status).toBe('pending')
    view.rerender(<RightsPanel {...props} disabled={true} workspace={workspace} />)
    expect(screen.getByLabelText('作品引用').matches(':disabled')).toBe(true)
    expect(screen.getByLabelText('权利依据').matches(':disabled')).toBe(true)
    view.rerender(<RightsPanel {...props} workspace={{ ...workspace, build: { ...workspace.build, status: 'publication_candidate' } }} />)
    expect(screen.queryByRole('button', { name: '为此对象追加补证' })).toBeNull()
  })
})
