// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { textbookService } from '@/services/textbook'
import { ArtifactVerificationControls } from './ArtifactVerificationControls'
import { VerificationPanel } from './VerificationPanel'

vi.mock('@/services/textbook', () => ({ textbookService: {
  getArtifactVerification: vi.fn(), getTask: vi.fn(), getChapterWorkspace: vi.fn(),
  startArtifactVerification: vi.fn(), retryTask: vi.fn(), cancelVerificationTask: vi.fn(),
} }))
const snapshot = (status: string) => ({ id: 'task', status, task_type: 'verification', task_subtype: 'textbook_teaching_artifact_verify', retry_count: 0 })
const loaded = vi.fn()
const open = () => render(<ArtifactVerificationControls chapterId="chapter" artifactId="artifact" onEvidenceLoaded={loaded} />)
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: null })
  vi.mocked(textbookService.getChapterWorkspace).mockResolvedValue({ code: 0, data: {} } as never)
})
afterEach(cleanup)

it('starts only on an explicit click and restores the same task after remount', async () => {
  const view = open()
  fireEvent.click(await screen.findByRole('button', { name: '运行隔离验证' }))
  await waitFor(() => expect(textbookService.startArtifactVerification).toHaveBeenCalledWith('chapter', 'artifact', { request_id: expect.any(String) }))
  view.unmount()
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'running' } })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('running') as never)
  open()
  await screen.findByText('正在隔离验证')
  expect(textbookService.startArtifactVerification).toHaveBeenCalledTimes(1)
  expect(screen.getByRole('button', { name: '取消验证' })).toBeTruthy()
})

it('polls a running task and refreshes evidence without asserting a passed result', async () => {
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'running' } })
  vi.mocked(textbookService.getTask).mockResolvedValueOnce(snapshot('running') as never).mockResolvedValue(snapshot('succeeded') as never)
  open()
  await screen.findByText('正在隔离验证')
  await waitFor(() => expect(loaded).toHaveBeenCalledTimes(1), { timeout: 3500 })
  expect(screen.getByText('验证任务已完成')).toBeTruthy()
  expect(screen.queryByText('已验证')).toBeNull()
  expect(textbookService.startArtifactVerification).not.toHaveBeenCalled()
})

it('offers explicit frozen retry for failure and cancel only for active work', async () => {
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'failed' } })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('failed') as never)
  open()
  fireEvent.click(await screen.findByRole('button', { name: '重试同一验证' }))
  await waitFor(() => expect(textbookService.retryTask).toHaveBeenCalledWith('task'))
  expect(screen.queryByRole('button', { name: '取消验证' })).toBeNull()
})

it('does not start or retry on lookup failure, and rejects a mismatched task type', async () => {
  vi.mocked(textbookService.getArtifactVerification).mockRejectedValueOnce(new Error('连接中断'))
  open()
  await screen.findByRole('alert')
  expect(screen.queryByRole('button', { name: '运行隔离验证' })).toBeNull()
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'running' } })
  vi.mocked(textbookService.getTask).mockResolvedValue({ ...snapshot('running'), task_subtype: 'textbook_sample_generation' } as never)
  fireEvent.click(screen.getByRole('button', { name: '刷新验证状态' }))
  await screen.findByText(/任务类型不属于教学代码验证/)
  expect(textbookService.retryTask).not.toHaveBeenCalled()
  expect(textbookService.startArtifactVerification).not.toHaveBeenCalled()
})

it('cancels an active task explicitly', async () => {
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'running' } })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('running') as never)
  open()
  fireEvent.click(await screen.findByRole('button', { name: '取消验证' }))
  await waitFor(() => expect(textbookService.cancelVerificationTask).toHaveBeenCalledWith('task'))
})

it('waits for executor exit, then creates an independent attempt after cancellation', async () => {
  const old = { id: 'task', status: 'cancelled' as const, attempt: 1, attempt_contract: 'inkwords.verification-attempt.v1', execution_stopped: false, can_start_new: false }
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: old })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('cancelled') as never)
  open()
  await screen.findByText(/正在等待执行器退出确认/)
  expect(screen.queryByRole('button', { name: '重新运行隔离验证' })).toBeNull()
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { ...old, execution_stopped: true, can_start_new: true } })
  fireEvent.click(screen.getByRole('button', { name: '刷新验证状态' }))
  fireEvent.click(await screen.findByRole('button', { name: '重新运行隔离验证' }))
  await waitFor(() => expect(textbookService.startArtifactVerification).toHaveBeenCalledWith('chapter', 'artifact', { request_id: expect.any(String), expected_previous_task_id: 'task' }))
  expect(textbookService.retryTask).not.toHaveBeenCalled()
})

it('pins an uncertain request and predecessor while showing immutable history', async () => {
  const first = { id: 'task', status: 'succeeded' as const, attempt: 1, attempt_contract: 'inkwords.verification-attempt.v1', execution_stopped: true, can_start_new: true }
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { ...first, history: [first] } })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('succeeded') as never)
  vi.mocked(textbookService.startArtifactVerification).mockRejectedValueOnce(new Error('响应中断')).mockResolvedValue({ code: 0, data: first })
  open()
  fireEvent.click(await screen.findByRole('button', { name: '重新验证' }))
  await screen.findByRole('button', { name: '重试提交本次验证' })
  const request = vi.mocked(textbookService.startArtifactVerification).mock.calls[0][2]
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { ...first, id: 'other', attempt: 2, history: [{ ...first, id: 'other', attempt: 2 }, first] } })
  vi.mocked(textbookService.getTask).mockResolvedValue({ ...snapshot('succeeded'), id: 'other' } as never)
  fireEvent.click(screen.getByRole('button', { name: '刷新验证状态' }))
  await screen.findByText('验证历史（2 次）')
  fireEvent.click(screen.getByRole('button', { name: '重试提交本次验证' }))
  await waitFor(() => expect(textbookService.startArtifactVerification).toHaveBeenCalledTimes(2))
  expect(vi.mocked(textbookService.startArtifactVerification).mock.calls[1][2]).toEqual(request)
  expect(request?.expected_previous_task_id).toBe('task')
  expect(textbookService.retryTask).not.toHaveBeenCalled()
})

it('keeps expired evidence unverified while offering an explicit new attempt', async () => {
  const artifact = { id: 'artifact', revision_id: 'revision', kind: 'teaching_implementation' as const, language: 'go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified' as const, created_at: '' }
  const expired = { id: 'old-evidence', revision_id: 'revision', code_artifact_id: 'artifact', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output' as const, status: 'verified' as const, output_truncated: false, expires_at: '2020-01-01T00:00:00Z', created_at: '2020-01-01T00:00:00Z' }
  vi.mocked(textbookService.getArtifactVerification).mockResolvedValue({ code: 0, data: { id: 'task', status: 'succeeded', attempt: 1, attempt_contract: 'inkwords.verification-attempt.v1', execution_stopped: true, can_start_new: true } })
  vi.mocked(textbookService.getTask).mockResolvedValue(snapshot('succeeded') as never)
  vi.mocked(textbookService.getChapterWorkspace).mockResolvedValue({ code: 0, data: { code_artifacts: [artifact], runtime_evidence: [expired], assets: [] } } as never)
  render(<VerificationPanel chapterId="chapter" artifacts={[artifact]} evidence={[expired]} assets={[]} />)
  const rerun = await screen.findByRole('button', { name: '重新验证' })
  expect(screen.getByText('运行证据已过期，需重新验证后才能作为当前事实使用。')).toBeTruthy()
  expect(screen.queryByText('已验证')).toBeNull()
  expect(textbookService.startArtifactVerification).not.toHaveBeenCalled()
  fireEvent.click(rerun)
  await waitFor(() => expect(textbookService.startArtifactVerification).toHaveBeenCalledWith('chapter', 'artifact', { request_id: expect.any(String), expected_previous_task_id: 'task' }))
  expect(screen.queryByText('已验证')).toBeNull()
})
