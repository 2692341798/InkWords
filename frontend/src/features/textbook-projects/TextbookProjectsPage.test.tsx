// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { TextbookWorkspace, TextbookProjectProgress } from '@/services/textbook'

const storeState = {
  projects: [], selectedWorkspace: null as TextbookWorkspace | null, selectedProjectProgress: null as TextbookProjectProgress | null, sourceLibrary: [], sourceEvidence: [], selectedChapterWorkspace: null, latestSampleTask: null, latestSourceImportTask: null, sourceRetrieval: null, isLoading: false, error: null,
  load: vi.fn(), create: vi.fn(), select: vi.fn(), addOfficialSource: vi.fn(), loadGinFixture: vi.fn(), importSourceFile: vi.fn(), importOfficialWeb: vi.fn(), retrieveSourceEvidence: vi.fn(), createChapter: vi.fn(), createBookContract: vi.fn(), createStyleSheet: vi.fn(), approveBookContract: vi.fn(), approveStyleSheet: vi.fn(), createBlueprint: vi.fn(), approveBlueprint: vi.fn(), openChapter: vi.fn(), closeChapter: vi.fn(), acquireChapterLock: vi.fn(), saveChapterDraft: vi.fn(), applyCandidate: vi.fn(), rejectCandidate: vi.fn(), prepareSampleGeneration: vi.fn(), generateSample: vi.fn(),
}

vi.mock('@/store/textbookStore', () => ({ useTextbookStore: () => storeState }))
vi.mock('@/services/mastery', () => ({ masteryService: { getDue: vi.fn().mockResolvedValue({ tasks: [] }) } }))

import { TextbookProjectsPage } from './TextbookProjectsPage'

describe('TextbookProjectsPage', () => {
  afterEach(() => {
    cleanup()
    storeState.selectedWorkspace = null
    storeState.selectedProjectProgress = null
    localStorage.clear()
  })

  it('uses server progress instead of a creation step when reopening a project', () => {
    storeState.selectedWorkspace = {
      project: { id: 'project-1', title: 'Gin 自学教材', audience_level: 'foundation', status: 'draft', revision_version: 1, updated_at: '2026-09-06T00:00:00Z' },
      sources: [], chapters: [],
    }
    storeState.selectedProjectProgress = { stages: [
      { key: 'sources', status: 'approved', reason: '已有可引用的固定资料。' },
      { key: 'sample', status: 'in_progress', reason: '候选稿等待人工审阅。' },
    ] }
    const { rerender } = render(<TextbookProjectsPage />)
    expect(screen.queryByRole('list', { name: '教材项目向导' })).toBeNull()
    expect(screen.getByText('已有可引用的固定资料。')).toBeTruthy()
    expect(screen.getByText('候选稿等待人工审阅。')).toBeTruthy()
    expect(screen.queryByText(/项目已创建：继续完成/)).toBeNull()
    expect(screen.queryByText('待导入')).toBeNull()

    storeState.selectedProjectProgress = { stages: [
      { key: 'sources', status: 'blocked', reason: '新项目还未导入资料。' },
    ] }
    rerender(<TextbookProjectsPage />)
    expect(screen.getByText('新项目还未导入资料。')).toBeTruthy()
    expect(screen.queryByText('已有可引用的固定资料。')).toBeNull()
  })

  it('guides a keyboard-accessible reader and project identity step before accepting a primary source', () => {
    render(<TextbookProjectsPage />)
    expect(screen.getByRole('list', { name: '教材项目向导' }).getAttribute('aria-label')).toBe('教材项目向导')
    expect(screen.getByRole('textbox', { name: '教材名称' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '下一步：主资料' }))
    expect(screen.getByRole('combobox', { name: '主资料类型' })).toBeTruthy()
    expect(screen.getByRole('textbox', { name: '主资料定位' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '创建并进入资料工作台' })).toBeTruthy()
  })

  it('keeps a recoverable message when the required title is missing', () => {
    render(<TextbookProjectsPage />)
    fireEvent.change(screen.getByRole('textbox', { name: '教材名称' }), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: '下一步：主资料' }))
    expect(screen.getByRole('alert').textContent).toContain('请先填写教材名称')
    expect(screen.queryByRole('combobox', { name: '主资料类型' })).toBeNull()
  })
})
