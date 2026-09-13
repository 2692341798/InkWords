// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { EditorialWorkspace } from './EditorialWorkspace'

const build = { id: 'build-1', project_id: 'project-1', book_contract_revision_id: 'contract-1', style_sheet_revision_id: 'style-1', approved_revision_ids: ['revision-1'], manifest_json: {}, manifest_hash: 'sha256:manifest', status: 'ready_for_review' as const, blockers_json: [], created_at: '2026-09-04T00:00:00Z' }

describe('EditorialWorkspace', () => {
  afterEach(() => { cleanup(); vi.restoreAllMocks() })

  it('keeps each form draft within its build without duplicate sibling identities', () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const workspace = { build, required_rights_subjects: [], rights_items: [], human_reviews: [], automated_checks: [], preflight: { passed: false, blockers: [] } }
    const actions = { onAddRight: vi.fn(), onCompleteReview: vi.fn(), onPromote: vi.fn() }
    const { rerender } = render(<EditorialWorkspace disabled={false} workspace={workspace} {...actions} />)
    fireEvent.change(screen.getByLabelText('权利依据'), { target: { value: '当前构建的权利草稿' } })
    fireEvent.change(screen.getByLabelText('真人审校范围'), { target: { value: '当前构建的审校草稿' } })
    rerender(<EditorialWorkspace disabled={false} workspace={{ ...workspace, build: { ...build } }} {...actions} />)
    expect((screen.getByLabelText('权利依据') as HTMLTextAreaElement).value).toBe('当前构建的权利草稿')
    expect((screen.getByLabelText('真人审校范围') as HTMLTextAreaElement).value).toBe('当前构建的审校草稿')
    rerender(<EditorialWorkspace disabled={false} workspace={{ ...workspace, build: { ...build, id: 'build-2', manifest_hash: 'sha256:second' } }} {...actions} />)
    expect(screen.getAllByLabelText('权利依据')).toHaveLength(1)
    expect(screen.getAllByLabelText('真人审校范围')).toHaveLength(1)
    expect((screen.getByLabelText('权利依据') as HTMLTextAreaElement).value).toBe('')
    expect((screen.getByLabelText('真人审校范围') as HTMLTextAreaElement).value).toBe('')
    expect(errors.mock.calls.filter((args) => args.some((arg) => String(arg).includes('same key')))).toHaveLength(0)
    expect(actions.onAddRight).not.toHaveBeenCalled()
    expect(actions.onCompleteReview).not.toHaveBeenCalled()
  })

  it('keeps automated checks separate and blocks explicit promotion while human evidence is missing', () => {
    render(<EditorialWorkspace disabled={false} workspace={{ build, required_rights_subjects: [{ subject_ref: 'chapter-revision:revision-1', work_type: 'prose' }], rights_items: [], human_reviews: [], automated_checks: [{ id: 'manifest', detector: 'core-api', status: 'pass' }], preflight: { passed: false, blockers: ['缺少人工技术审校记录。'] } }} onAddRight={vi.fn()} onCompleteReview={vi.fn()} onPromote={vi.fn()} />)
    expect(screen.getByText('八阶段人工审校')).toBeTruthy()
    expect(screen.getByText('manifest · core-api · pass')).toBeTruthy()
    expect(screen.getByText('chapter-revision:revision-1')).toBeTruthy()
    expect(screen.getByText('缺少人工技术审校记录。')).toBeTruthy()
    expect((screen.getByRole('button', { name: '显式标记为出版候选' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText(/不代表出版社、ISBN 或 CIP 批准/)).toBeTruthy()
  })
})
