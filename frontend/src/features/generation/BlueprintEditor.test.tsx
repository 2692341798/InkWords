// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { BlueprintEditor } from './BlueprintEditor'

const project = { id: 'project-1', title: 'Gin 教材', audience_level: 'foundation' as const, status: 'draft' as const, revision_version: 1, approved_book_contract_revision_id: 'book-1', approved_style_sheet_revision_id: 'style-1', updated_at: '2026-09-03T00:00:00Z' }
const workspace = { project, sources: [], chapters: [{ id: 'chapter-1', project_id: project.id, sort_order: 1, title: '为什么需要路由', chapter_profile: 'concept' as const, status: 'draft', revision_version: 0, updated_at: project.updated_at }] }
const evidence = [{ id: 'chunk-1', document_id: 'doc-1', document_title: 'routergroup.go', canonical_locator: 'routergroup.go', ordinal: 3 }]

describe('BlueprintEditor', () => {
  afterEach(cleanup)

  it('requires explicit evidence selection before saving a draft', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(<BlueprintEditor project={project} workspace={workspace} evidence={evidence} isBusy={false} onCreate={onCreate} onApprove={vi.fn()} onRetrieveEvidence={vi.fn()} />)
    const button = screen.getByRole('button', { name: '保存蓝图草稿' })
    expect((button as HTMLButtonElement).disabled).toBe(true)
    fireEvent.click(screen.getByRole('checkbox', { name: 'routergroup.go · #3 · routergroup.go' }))
		fireEvent.change(screen.getByRole('textbox', { name: '为什么需要路由 的关键事实' }), { target: { value: 'GET 会进入路由登记流程' } })
    fireEvent.click(button)
		await waitFor(() => expect(onCreate).toHaveBeenCalledWith({ volumes: [{ id: 'volume-project-1', title: '核心学习路径', sort: 1, chapters: [{ id: 'chapter-1', title: '为什么需要路由', sort: 1, profile: 'concept', evidence_ids: ['chunk-1'], critical_claims: [{ id: 'chapter-1-claim-1', label: 'GET 会进入路由登记流程', evidence_ids: ['chunk-1'] }] }] }] }))
  })

  it('keeps approval separate from a saved draft', async () => {
    const onApprove = vi.fn().mockResolvedValue(undefined)
    render(<BlueprintEditor project={project} workspace={{ ...workspace, blueprint: { id: 'blueprint-1', project_id: project.id, revision_number: 1, document_json: {}, content_hash: 'a'.repeat(64), status: 'draft' as const, created_at: project.updated_at } }} evidence={evidence} isBusy={false} onCreate={vi.fn()} onApprove={onApprove} onRetrieveEvidence={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: '我已审阅并批准' }))
    await waitFor(() => expect(onApprove).toHaveBeenCalledWith('blueprint-1'))
  })

  it('restores the saved volume, critical facts, and evidence mapping after a reload', () => {
    render(<BlueprintEditor project={project} workspace={{
      ...workspace,
      blueprint: {
        id: 'blueprint-1', project_id: project.id, revision_number: 1, content_hash: 'a'.repeat(64), status: 'approved' as const, created_at: project.updated_at,
        document_json: { volumes: [{ id: 'volume-1', title: '请求路由主线', sort: 1, chapters: [{ id: 'chapter-1', title: '为什么需要路由', sort: 1, profile: 'concept', evidence_ids: ['chunk-1'], critical_claims: [{ id: 'claim-1', label: 'GET 会进入路由登记流程', evidence_ids: ['chunk-1'] }] }] }] },
      },
    }} evidence={evidence} isBusy={false} onCreate={vi.fn()} onApprove={vi.fn()} onRetrieveEvidence={vi.fn()} />)

    expect((screen.getByRole('textbox', { name: '单元标题' }) as HTMLInputElement).value).toBe('请求路由主线')
    expect((screen.getByRole('textbox', { name: '为什么需要路由 的关键事实' }) as HTMLTextAreaElement).value).toBe('GET 会进入路由登记流程')
    expect((screen.getByRole('checkbox', { name: 'routergroup.go · #3 · routergroup.go' }) as HTMLInputElement).checked).toBe(true)
  })

  it('retrieves evidence candidates from each stated critical fact without auto-selecting them', async () => {
    const onRetrieveEvidence = vi.fn().mockResolvedValue({ id: 'retrieval-1', query: 'GET 会进入路由登记流程', input_hash: 'hash', candidates: [], selected: [{ chunk_id: 'chunk-1', document_id: 'doc-1', document_title: 'routergroup.go', snapshot_id: 'snapshot-1', source_role: 'primary', canonical_locator: 'routergroup.go', ordinal: 3, score: 8, reasons: ['主资料优先'] }], created_at: project.updated_at })
    render(<BlueprintEditor project={project} workspace={workspace} evidence={evidence} isBusy={false} onCreate={vi.fn()} onApprove={vi.fn()} onRetrieveEvidence={onRetrieveEvidence} />)
    fireEvent.change(screen.getByRole('textbox', { name: '为什么需要路由 的关键事实' }), { target: { value: 'GET 会进入路由登记流程' } })
    fireEvent.click(screen.getByRole('button', { name: '按关键事实查找候选资料' }))
    await waitFor(() => expect(onRetrieveEvidence).toHaveBeenCalledWith('GET 会进入路由登记流程'))
    expect(screen.getByLabelText('为什么需要路由 的候选资料').textContent).toContain('routergroup.go · #3')
    expect((screen.getByRole('checkbox', { name: 'routergroup.go · #3 · routergroup.go' }) as HTMLInputElement).checked).toBe(false)
  })

  it('makes a missing critical fact actionable when no registered evidence matches', async () => {
    const onRetrieveEvidence = vi.fn().mockResolvedValue({ id: 'retrieval-missing', query: '无法在资料中证明的事实', input_hash: 'hash', candidates: [], selected: [], created_at: project.updated_at })
    render(<BlueprintEditor project={project} workspace={workspace} evidence={evidence} isBusy={false} onCreate={vi.fn()} onApprove={vi.fn()} onRetrieveEvidence={onRetrieveEvidence} />)
    fireEvent.change(screen.getByRole('textbox', { name: '为什么需要路由 的关键事实' }), { target: { value: '无法在资料中证明的事实' } })
    fireEvent.click(screen.getByRole('button', { name: '按关键事实查找候选资料' }))
    await waitFor(() => expect(screen.getByLabelText('为什么需要路由 的候选资料').textContent).toContain('请补充主资料或已确认官方资料后重新检索'))
  })

  it('lets the author select a retrieved chunk outside the initial bounded list', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    const onRetrieveEvidence = vi.fn().mockResolvedValue({ id: 'retrieval-late', query: '路径前缀匹配', input_hash: 'hash', candidates: [], selected: [{ chunk_id: 'chunk-late', document_id: 'doc-tree', document_title: 'tree.go', snapshot_id: 'snapshot-tree', source_role: 'primary', canonical_locator: 'tree.go', ordinal: 59, score: 8, reasons: ['正文关键词:prefix'] }], created_at: project.updated_at })
    render(<BlueprintEditor project={project} workspace={workspace} evidence={evidence} isBusy={false} onCreate={onCreate} onApprove={vi.fn()} onRetrieveEvidence={onRetrieveEvidence} />)
    fireEvent.change(screen.getByRole('textbox', { name: '为什么需要路由 的关键事实' }), { target: { value: '路径前缀匹配' } })
    fireEvent.click(screen.getByRole('button', { name: '按关键事实查找候选资料' }))
    const checkbox = await screen.findByRole('checkbox', { name: 'tree.go · #59 · tree.go' })
    expect((checkbox as HTMLInputElement).checked).toBe(false)
    fireEvent.click(checkbox)
    fireEvent.click(screen.getByRole('button', { name: '保存蓝图草稿' }))
    await waitFor(() => expect(onCreate).toHaveBeenCalled())
    expect(onCreate.mock.calls[0][0].volumes[0].chapters[0].evidence_ids).toEqual(['chunk-late'])
  })

  it('shows a saved out-of-list selection after reload without inventing its locator', () => {
    render(<BlueprintEditor project={project} workspace={{ ...workspace, blueprint: {
      id: 'blueprint-1', project_id: project.id, revision_number: 1, content_hash: 'a'.repeat(64), status: 'draft' as const, created_at: project.updated_at,
      document_json: { volumes: [{ title: '路由', chapters: [{ id: 'chapter-1', evidence_ids: ['chunk-late'], critical_claims: [{ label: '路径前缀匹配' }] }] }] },
    } }} evidence={evidence} isBusy={false} onCreate={vi.fn()} onApprove={vi.fn()} onRetrieveEvidence={vi.fn()} />)
    const checkbox = screen.getByRole('checkbox', { name: '已保存的证据片段 · chunk-late' })
    expect((checkbox as HTMLInputElement).checked).toBe(true)
    fireEvent.click(checkbox)
    expect((screen.getByRole('button', { name: '保存蓝图草稿' }) as HTMLButtonElement).disabled).toBe(true)
  })
})
