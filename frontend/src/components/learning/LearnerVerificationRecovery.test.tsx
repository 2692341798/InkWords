// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { LearnerCodeSnapshot } from './LearnerCodeSnapshot'
import { masteryService, type LearnerArtifact, type LearnerVerificationJob, type LearnerVerificationPreview } from '@/services/mastery'

const preview: LearnerVerificationPreview = { capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: true, profile: 'inkwords.learner-go-test-offline.v2' }, requires_explicit_start: true, snapshot_hash: 'sha256:code', input_hash: 'sha256:input' }
const artifact: LearnerArtifact = { format: 'inkwords.learner-artifact.v1', workspace_id: 'local', objective_id: 'objective', attempt_id: 'attempt', session_id: 'session', revision_id: 'revision', task_id: 'task', skill: 'reproduce', practice_content_hash: 'sha256:practice', submitted_at: '2026-09-10T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'answer.go', content: 'package answer' }] }
const job: LearnerVerificationJob = { id: 'run', objective_id: 'objective', attempt_id: 'attempt', request_id: 'request', status: 'passed', preview, created_at: '2026-09-10T00:00:00Z' }
const mount = () => render(<LearnerCodeSnapshot objectiveID="objective" attemptID="attempt" hash="sha256:code" />)

beforeEach(() => {
  vi.spyOn(masteryService, 'learnerArtifact').mockResolvedValue(artifact)
  vi.spyOn(masteryService, 'learnerVerificationPreview').mockResolvedValue(preview)
  vi.spyOn(masteryService, 'learnerVerificationLatest').mockResolvedValue(null)
})
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.useRealTimers() })

it('does not mistake an unread history for no run and recovers with reads only', async () => {
  vi.mocked(masteryService.learnerVerificationLatest).mockRejectedValueOnce(new Error('连接中断')).mockResolvedValue(job)
  const start = vi.spyOn(masteryService, 'startLearnerVerification')
  mount()
  await screen.findByText(/无法确认最新验证状态/)
  expect(screen.queryByText('本次记录尚未运行。')).toBeNull()
  expect(screen.queryByRole('button', { name: '明确开始验证' })).toBeNull()
  fireEvent.click(screen.getByRole('button', { name: '重新读取验证状态' }))
  await screen.findByText('状态：学习者检查通过')
  expect(start).not.toHaveBeenCalled()
})

it('keeps a historical result visible when the runner becomes unavailable', async () => {
  vi.mocked(masteryService.learnerVerificationLatest).mockResolvedValue(job)
  vi.mocked(masteryService.learnerVerificationPreview).mockResolvedValue({ ...preview, capability: { ...preview.capability, available: false, reason: '运行器离线' } })
  mount()
  await screen.findByText('状态：学习者检查通过')
  expect(screen.getByText(/当前无法安全执行/)).toBeTruthy()
  expect(screen.queryByRole('button', { name: '明确重新验证' })).toBeNull()
})

it('bounds an independent hung preview even after the code read finishes', async () => {
  vi.useFakeTimers()
  vi.mocked(masteryService.learnerVerificationPreview).mockImplementation((_o, _a, signal) => new Promise((_resolve, reject) => signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))))
  mount()
  await act(async () => { await vi.advanceTimersByTimeAsync(10001) })
  expect(screen.getByText(/隔离预检读取超时/)).toBeTruthy()
  expect(screen.queryByText('正在读取隔离预检…')).toBeNull()
})

it('keeps uncertain starts idempotent when the response is lost', async () => {
  const start = vi.spyOn(masteryService, 'startLearnerVerification').mockRejectedValueOnce(new Error('响应丢失')).mockResolvedValue(job)
  mount()
  fireEvent.click(await screen.findByRole('button', { name: '明确开始验证' }))
  fireEvent.click(await screen.findByRole('button', { name: '重试启动请求' }))
  await screen.findByText('状态：学习者检查通过')
  expect(start).toHaveBeenCalledTimes(2)
  expect(start.mock.calls[1].slice(0, 3)).toEqual(start.mock.calls[0].slice(0, 3))
})

it('ends a hung start wait without inventing a cancelled or failed run', async () => {
  vi.useFakeTimers()
  const start = vi.spyOn(masteryService, 'startLearnerVerification').mockImplementation((_o, _a, _request, signal) => new Promise((_resolve, reject) => signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))))
  mount()
  await act(async () => { await Promise.resolve() })
  fireEvent.click(screen.getByRole('button', { name: '明确开始验证' }))
  await act(async () => { await vi.advanceTimersByTimeAsync(10001) })
  expect(screen.getByText(/启动响应超时/)).toBeTruthy()
  expect(screen.getByRole('button', { name: '重试启动请求' }).hasAttribute('disabled')).toBe(false)
  expect(screen.queryByText('状态：已取消')).toBeNull()
  expect(start).toHaveBeenCalledTimes(1)
})

it('does not retain a previous attempt snapshot or action during navigation', async () => {
  const view = mount()
  await screen.findByText('answer.go')
  vi.mocked(masteryService.learnerArtifact).mockResolvedValue(null)
  view.rerender(<LearnerCodeSnapshot objectiveID="objective" attemptID="other" hash="sha256:other" />)
  await waitFor(() => expect(screen.queryByText('answer.go')).toBeNull())
  expect(screen.queryByRole('button', { name: '明确开始验证' })).toBeNull()
})

it('serializes status reads and ignores an old running response after cancellation', async () => {
  vi.useFakeTimers()
  vi.mocked(masteryService.learnerVerificationLatest).mockResolvedValue({ ...job, status: 'running' })
  let release: (value: LearnerVerificationJob) => void = () => undefined
  const read = vi.spyOn(masteryService, 'readLearnerVerification').mockImplementation(() => new Promise((resolve) => { release = resolve }))
  vi.spyOn(masteryService, 'cancelLearnerVerification').mockResolvedValue({ ...job, status: 'cancelled' })
  mount()
  await act(async () => { await Promise.resolve() })
  await act(async () => { await vi.advanceTimersByTimeAsync(5000) })
  expect(read).toHaveBeenCalledTimes(1)
  await act(async () => { fireEvent.click(screen.getByRole('button', { name: '取消验证' })) })
  expect(screen.getByText('状态：已取消')).toBeTruthy()
  await act(async () => { release({ ...job, status: 'running' }); await vi.advanceTimersByTimeAsync(5000) })
  expect(screen.getByText('状态：已取消')).toBeTruthy()
  expect(read).toHaveBeenCalledTimes(1)
})

it('shows a poll failure without claiming completion and resumes reading the same run', async () => {
  vi.useFakeTimers()
  vi.mocked(masteryService.learnerVerificationLatest).mockResolvedValue({ ...job, status: 'running' })
  const read = vi.spyOn(masteryService, 'readLearnerVerification').mockRejectedValueOnce(new Error('offline')).mockResolvedValue(job)
  const start = vi.spyOn(masteryService, 'startLearnerVerification')
  mount()
  await act(async () => { await Promise.resolve() })
  await act(async () => { await vi.advanceTimersByTimeAsync(1000) })
  expect(screen.getByText(/验证状态暂时无法更新/)).toBeTruthy()
  expect(screen.getByText('状态：正在隔离运行')).toBeTruthy()
  await act(async () => { await vi.advanceTimersByTimeAsync(2000) })
  expect(screen.getByText('状态：学习者检查通过')).toBeTruthy()
  expect(screen.queryByText(/验证状态暂时无法更新/)).toBeNull()
  expect(read.mock.calls.map((args) => args.slice(0, 2))).toEqual([['objective', 'run'], ['objective', 'run']])
  expect(start).not.toHaveBeenCalled()
})
