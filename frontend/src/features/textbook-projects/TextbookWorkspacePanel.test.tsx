// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TextbookWorkspacePanel } from './TextbookWorkspacePanel'

const workspace = {
  project: {
    id: 'project-1', title: 'Gin 自学教材', audience_level: 'foundation' as const, status: 'draft' as const, revision_version: 1, updated_at: '2026-09-02T00:00:00Z',
  },
  sources: [{
    id: 'source-1', project_id: 'project-1', kind: 'git_repository' as const, role: 'primary' as const, locator: 'https://github.com/gin-gonic/gin', official_confirmed: false, license_status: 'pending', created_at: '2026-09-02T00:00:00Z',
  }],
  chapters: [{
    id: 'chapter-1', project_id: 'project-1', sort_order: 1, title: '为什么需要 Web 框架', chapter_profile: 'concept' as const, status: 'draft', revision_version: 0, updated_at: '2026-09-02T00:00:00Z',
  }],
}

describe('TextbookWorkspacePanel', () => {
  afterEach(cleanup)

  it('renders persisted inputs and sends explicit official-source confirmation', async () => {
    const onAddOfficialSource = vi.fn().mockResolvedValue(undefined)
    render(<TextbookWorkspacePanel workspace={workspace} isBusy={false} onAddOfficialSource={onAddOfficialSource} onLoadGinFixture={vi.fn()} onImportSourceFile={vi.fn()} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)

    expect(screen.getByText('https://github.com/gin-gonic/gin')).toBeTruthy()
    expect(screen.getByText('为什么需要 Web 框架')).toBeTruthy()
    const locator = screen.getByRole('textbox', { name: '官方资料地址或本地路径' })
    fireEvent.change(locator, { target: { value: 'https://gin-gonic.com/docs/' } })
    fireEvent.submit(locator.closest('form')!)

    await waitFor(() => expect(onAddOfficialSource).toHaveBeenCalledWith({ kind: 'official_web', locator: 'https://gin-gonic.com/docs/' }))
    await waitFor(() => expect((locator as HTMLInputElement).value).toBe(''))
  })

  it('creates an explicitly profiled next chapter', async () => {
    const onCreateChapter = vi.fn().mockResolvedValue(undefined)
    render(<TextbookWorkspacePanel workspace={workspace} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={vi.fn()} onImportSourceFile={vi.fn()} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={onCreateChapter} onOpenChapter={vi.fn()} />)

    const title = screen.getByRole('textbox', { name: '章节标题' })
    fireEvent.change(title, { target: { value: '路由如何选择处理函数' } })
    fireEvent.change(screen.getByRole('combobox', { name: '教学类型' }), { target: { value: 'hands_on' } })
    fireEvent.submit(title.closest('form')!)

    await waitFor(() => expect(onCreateChapter).toHaveBeenCalledWith({ title: '路由如何选择处理函数', chapterProfile: 'hands_on' }))
    await waitFor(() => expect((title as HTMLInputElement).value).toBe(''))
  })

  it('exposes the fixed Gin fixture only for the matching primary source', async () => {
	const onLoadGinFixture = vi.fn().mockResolvedValue(undefined)
	render(<TextbookWorkspacePanel workspace={workspace} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={onLoadGinFixture} onImportSourceFile={vi.fn()} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)

	expect(screen.getByText(/ServeHTTP、handleHTTPRequest、getValue/)).toBeTruthy()
	fireEvent.click(screen.getByRole('button', { name: '载入固定 Gin 样章资料' }))
	await waitFor(() => expect(onLoadGinFixture).toHaveBeenCalledOnce())
	cleanup()
	render(<TextbookWorkspacePanel workspace={{ ...workspace, sources: [{ ...workspace.sources[0], locator: 'https://github.com/other/project' }] }} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={vi.fn()} onImportSourceFile={vi.fn()} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)
	expect(screen.queryByRole('button', { name: '载入固定 Gin 样章资料' })).toBeNull()
  })

  it('queues only a file that matches a registered local source', async () => {
	const onImportSourceFile = vi.fn().mockResolvedValue(undefined)
	const localWorkspace = { ...workspace, sources: [{ ...workspace.sources[0], id: 'source-local', kind: 'markdown' as const, locator: 'file:///intro.md' }] }
	render(<TextbookWorkspacePanel workspace={localWorkspace} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={vi.fn()} onImportSourceFile={onImportSourceFile} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)

	const input = screen.getByLabelText('导入 file:///intro.md')
	const file = new File(['# 入门'], 'intro.md', { type: 'text/markdown' })
	fireEvent.change(input, { target: { files: [file] } })
	await waitFor(() => expect(onImportSourceFile).toHaveBeenCalledWith('source-local', file))
  })

  it('shows deterministic retrieval reasons without exposing source excerpts', async () => {
	const onRetrieveSourceEvidence = vi.fn().mockResolvedValue(undefined)
	render(<TextbookWorkspacePanel workspace={workspace} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={vi.fn()} onImportSourceFile={vi.fn()} onImportOfficialWeb={vi.fn()} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={{ id: 'run-1', query: '路由', input_hash: 'sha256:plan', candidates: [], selected: [{ chunk_id: 'chunk-1', document_id: 'doc-1', document_title: 'RouterGroup', snapshot_id: 'snapshot-1', source_role: 'primary', canonical_locator: 'router.go', ordinal: 2, score: 6, reasons: ['标题路径:路由', '主资料优先'] }], created_at: '2026-09-03T00:00:00Z' }} onRetrieveSourceEvidence={onRetrieveSourceEvidence} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)

	expect(screen.getByText('匹配理由：标题路径:路由；主资料优先（分数 6）')).toBeTruthy()
	const query = screen.getByRole('textbox', { name: '检索关键词' })
	fireEvent.change(query, { target: { value: '路由' } })
	fireEvent.submit(query.closest('form')!)
	await waitFor(() => expect(onRetrieveSourceEvidence).toHaveBeenCalledWith('路由'))
  })

  it('queues a bounded crawl only for a confirmed official web source', async () => {
	const onImportOfficialWeb = vi.fn().mockResolvedValue(undefined)
	const officialWorkspace = { ...workspace, sources: [{ ...workspace.sources[0], id: 'source-web', kind: 'official_web' as const, role: 'official_supporting' as const, locator: 'https://gin-gonic.com/en/docs/', official_confirmed: false }] }
	render(<TextbookWorkspacePanel workspace={officialWorkspace} isBusy={false} onAddOfficialSource={vi.fn()} onLoadGinFixture={vi.fn()} onImportSourceFile={vi.fn()} onImportOfficialWeb={onImportOfficialWeb} latestSourceImportTask={null} onSourceImportSucceeded={vi.fn()} sourceRetrieval={null} onRetrieveSourceEvidence={vi.fn()} onCreateChapter={vi.fn()} onOpenChapter={vi.fn()} />)

	expect((screen.getByRole('textbox', { name: 'https://gin-gonic.com/en/docs/ 的允许抓取路径' }) as HTMLInputElement).value).toBe('/en/docs')
	fireEvent.click(screen.getByRole('button', { name: '抓取并导入官网资料' }))
	await waitFor(() => expect(onImportOfficialWeb).toHaveBeenCalledWith('source-web', '/en/docs'))
  })
})
