// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { BookBuildPanel } from './BookBuildPanel'

describe('BookBuildPanel', () => {
  afterEach(cleanup)

  it('keeps notice text through failed freezing and refuses to drop unadded input', async () => {
    const create = vi.fn().mockRejectedValueOnce(new Error('暂时不可用')).mockResolvedValue(undefined)
    render(<BookBuildPanel build={null} disabled={false} onCreate={create} chapters={[{id:'chapter',project_id:'project',sort_order:1,title:'路由',chapter_profile:'concept',status:'approved',revision_version:1,approved_revision_id:'revision-1',updated_at:''}]} />)
    fireEvent.click(screen.getByText('随稿分发声明（0 项）'))
    for (const [label, value] of [['声明标题','MIT'], ['声明来源链接','https://example.org/LICENSE'], ['声明整理者','委托 AI'], ['完整声明文字','Copyright fixture\nPermission fixture.\n']]) fireEvent.change(screen.getByLabelText(label), {target:{value}})
    expect(screen.getByRole('button',{name:'冻结待审构建'}).matches(':disabled')).toBe(true)
    fireEvent.click(screen.getByLabelText('章节：路由'))
    fireEvent.submit(screen.getByRole('button',{name:'添加到本次分发声明'}).closest('form')!)
    fireEvent.click(screen.getByRole('button',{name:'冻结待审构建'}))
    await waitFor(() => expect(create).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button',{name:'冻结待审构建'}))
    await waitFor(() => expect(create).toHaveBeenCalledTimes(2))
    expect(create.mock.calls[0][0]).toEqual(create.mock.calls[1][0])
    expect(create.mock.calls[1][0][0]).toMatchObject({text:'Copyright fixture\nPermission fixture.\n',subject_refs:['chapter-revision:revision-1']})
  })

  it('shows a frozen build as review-only and preserves publication blockers', () => {
    render(<BookBuildPanel disabled={false} onCreate={vi.fn().mockResolvedValue(undefined)} build={{
      id: 'build-1', project_id: 'project-1', book_contract_revision_id: 'contract-1', style_sheet_revision_id: 'style-1', approved_revision_ids: ['revision-1'], manifest_json: {}, manifest_hash: 'sha256:manifest', status: 'ready_for_review', blockers_json: ['尚未完成技术审校。'], created_at: '2026-09-03T00:00:00Z',
    }} />)
    expect(screen.getByText('待人工审校')).toBeTruthy()
    expect(screen.getByText('出版候选仍被阻断')).toBeTruthy()
    expect(screen.getByText('尚未完成技术审校。')).toBeTruthy()
		expect((screen.getByRole('link', { name: '下载 Markdown' }) as HTMLAnchorElement).getAttribute('href')).toBe('/api/v1/textbook-projects/book-builds/build-1/export/markdown')
		expect((screen.getByRole('link', { name: '生成 DOCX' }) as HTMLAnchorElement).getAttribute('href')).toBe('/api/v1/textbook-projects/book-builds/build-1/export/docx')
		expect((screen.getByRole('link', { name: '生成 PDF' }) as HTMLAnchorElement).getAttribute('href')).toBe('/api/v1/textbook-projects/book-builds/build-1/export/pdf')
		expect((screen.getByRole('link', { name: '下载审校包 ZIP' }) as HTMLAnchorElement).getAttribute('href')).toBe('/api/v1/textbook-projects/book-builds/build-1/export/review-bundle')
  })

  it('does not present a system publication preflight as publisher approval', () => {
    render(<BookBuildPanel disabled={false} onCreate={vi.fn().mockResolvedValue(undefined)} build={{
      id: 'build-2', project_id: 'project-1', book_contract_revision_id: 'contract-1', style_sheet_revision_id: 'style-1', approved_revision_ids: ['revision-1'], manifest_json: {}, manifest_hash: 'sha256:manifest', status: 'publication_candidate', blockers_json: [], tool_versions_json: { pandoc: 'pandoc 3.8' }, created_at: '2026-09-03T00:00:00Z',
    }} />)
    expect(screen.getByText('系统出版预检通过')).toBeTruthy()
    expect(screen.getByText(/不代表出版社批准/)).toBeTruthy()
    expect(screen.getByText('pandoc: pandoc 3.8')).toBeTruthy()
  })
})
