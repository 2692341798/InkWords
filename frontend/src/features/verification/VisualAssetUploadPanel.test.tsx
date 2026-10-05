// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ChapterRevision, TextbookRuntimeEvidence } from '@/services/textbook'
import { VisualAssetUploadPanel } from './VisualAssetUploadPanel'

afterEach(cleanup)

const revision = (id: string, number: number): ChapterRevision => ({
  id, chapter_id: 'chapter', revision_number: number, kind: 'candidate', markdown: '',
  document_json: {}, content_hash: `sha256:${id}`, created_by: 'generation', created_at: '',
})
const receipt = (id: string, revisionId: string): TextbookRuntimeEvidence => ({
  id, revision_id: revisionId, code_artifact_id: `artifact-${revisionId}`, code_artifact_hash: `sha256:${revisionId}`,
  input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', output_truncated: false,
  expires_at: '2099-01-01T00:00:00Z', created_at: '',
})
const revisions = [revision('new', 3), revision('old', 1)]

function fillFile() {
  fireEvent.change(screen.getByLabelText('截图文件'), { target: { files: [new File(['png'], 'capture.png', { type: 'image/png' })] } })
  fireEvent.change(screen.getByLabelText('截图替代文本'), { target: { value: '调用栈画面' } })
}

describe('VisualAssetUploadPanel', () => {
  it('requires an explicit receipt and binds its own revision regardless of list order', async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined)
    render(<VisualAssetUploadPanel revisions={revisions} evidence={[receipt('old-run', 'old'), receipt('new-run', 'new')]} disabled={false} onUpload={onUpload} />)
    fillFile()
    expect((screen.getByRole('button', { name: '登记截图资产' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.change(screen.getByLabelText('截图关联运行证据'), { target: { value: 'old-run' } })
    fireEvent.click(screen.getByRole('button', { name: '登记截图资产' }))
    await waitFor(() => expect(onUpload).toHaveBeenCalledWith(expect.objectContaining({ revisionId: 'old', evidenceId: 'old-run', rightsStatus: 'pending' })))
  })

  it('cannot submit an orphan receipt or a selection removed by a workspace refresh', () => {
    const onUpload = vi.fn()
    const { rerender } = render(<VisualAssetUploadPanel revisions={revisions} evidence={[receipt('old-run', 'old'), receipt('orphan', 'missing')]} disabled={false} onUpload={onUpload} />)
    expect(screen.queryByRole('option', { name: /orphan/ })).toBeNull()
    fireEvent.change(screen.getByLabelText('截图关联运行证据'), { target: { value: 'old-run' } })
    fillFile()
    rerender(<VisualAssetUploadPanel revisions={revisions} evidence={[receipt('new-run', 'new')]} disabled={false} onUpload={onUpload} />)
    fireEvent.click(screen.getByRole('button', { name: '登记截图资产' }))
    expect(onUpload).not.toHaveBeenCalled()
  })

  it('labels stale evidence without presenting a historical screenshot as current verification', () => {
    render(<VisualAssetUploadPanel revisions={revisions} evidence={[{ ...receipt('old-run', 'old'), stale_reason: '合同已变化' }]} disabled={false} onUpload={vi.fn()} />)
    fireEvent.change(screen.getByLabelText('截图关联运行证据'), { target: { value: 'old-run' } })
    expect(screen.getByText(/合同已变化/)).toBeTruthy()
    expect(screen.getByText(/修订：r1/)).toBeTruthy()
  })
})
