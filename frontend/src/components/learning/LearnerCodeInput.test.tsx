// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { LearnerCodeInput } from './LearnerCodeInput'
import { LearnerCodeSnapshot } from './LearnerCodeSnapshot'
import { masteryService } from '@/services/mastery'
import { readLearnerCodeFiles } from '@/lib/learnerCode'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

function sourceFile(name: string, content: string) {
  const file = new File([content], name)
  Object.defineProperty(file, 'arrayBuffer', { value: async () => new TextEncoder().encode(content).buffer })
  return file
}

it('preserves original UTF-8 bytes including BOM and CRLF during selection', async () => {
  const content = '\uFEFFpackage main\r\n// 原作答\r\n'
  const files = await readLearnerCodeFiles([sourceFile('answer.go', content)])
  expect(files).toEqual([{ path: 'answer.go', content }])
})

it('keeps the prior selection when a new file set is invalid', async () => {
  const files = [{ path: 'answer.go', content: 'package main' }]
  const onChange = vi.fn()
  render(<LearnerCodeInput files={files} onChange={onChange} onBusy={vi.fn()} disabled={false} />)
  fireEvent.click(screen.getByText('随作答保存代码（可选） · 1 个文件'))
  fireEvent.change(screen.getByLabelText('选择本次代码文件'), { target: { files: [sourceFile('run.sh', 'echo unexpected')] } })
  await screen.findByText('文件名需为相对路径的 .go 文件或 go.mod，不接受目录、压缩包或脚本。')
  expect(onChange).not.toHaveBeenCalled()
  expect(screen.getByText('answer.go', { selector: 'summary' })).toBeTruthy()
})

it('reads only the requested saved snapshot and renders code as text', async () => {
  const load = vi.spyOn(masteryService, 'learnerArtifact').mockResolvedValue({ format: 'inkwords.learner-artifact.v1', workspace_id: 'local', objective_id: 'objective', attempt_id: 'attempt', session_id: 'session', revision_id: 'revision', task_id: 'task', skill: 'reproduce', practice_content_hash: 'sha256:practice', submitted_at: '2026-09-05T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'answer.go', content: 'package main\n// <script>alert(1)</script>' }] })
  vi.spyOn(masteryService, 'learnerVerificationLatest').mockResolvedValue(null)
  vi.spyOn(masteryService, 'learnerVerificationPreview').mockResolvedValue({ capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: false, profile: 'inkwords.learner-go-test-offline.v2', reason: 'namespace preflight failed' }, requires_explicit_start: true })
  render(<LearnerCodeSnapshot objectiveID="objective" attemptID="attempt" hash="sha256:code" />)
  await screen.findByText('answer.go')
  expect(load).toHaveBeenCalledWith('objective', 'attempt', expect.any(AbortSignal))
  expect(screen.getByText('已随原作答保存。模型评分与隔离运行是两个独立动作。')).toBeTruthy()
  expect(await screen.findByText(/当前无法安全执行新的验证/)).toBeTruthy()
  expect(document.querySelector('script')).toBeNull()
})

it('requires an explicit click and sends only the frozen preview hash', async () => {
  vi.spyOn(masteryService, 'learnerArtifact').mockResolvedValue({ format: 'inkwords.learner-artifact.v1', workspace_id: 'local', objective_id: 'objective', attempt_id: 'attempt', session_id: 'session', revision_id: 'revision', task_id: 'task', skill: 'reproduce', practice_content_hash: 'sha256:practice', submitted_at: '2026-09-05T00:00:00Z', snapshot_hash: 'sha256:code', files: [{ path: 'answer.go', content: 'package main' }] })
  vi.spyOn(masteryService, 'learnerVerificationPreview').mockResolvedValue({ capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: true, profile: 'inkwords.learner-go-test-offline.v2', runner: { image_digest: 'sha256:runner', toolchain_version: 'go1.25.4' } }, input_hash: 'sha256:input', snapshot_hash: 'sha256:code', files_hash: 'sha256:files', execution_files_hash: 'sha256:tree', requires_explicit_start: true })
  vi.spyOn(masteryService, 'learnerVerificationLatest').mockResolvedValue(null)
  const start = vi.spyOn(masteryService, 'startLearnerVerification').mockResolvedValue({ id: 'run', objective_id: 'objective', attempt_id: 'attempt', request_id: 'request', status: 'queued', preview: {} as never, created_at: '2026-09-05T00:00:00Z' })
  vi.spyOn(crypto, 'randomUUID').mockReturnValue('11111111-1111-4111-8111-111111111111')
  render(<LearnerCodeSnapshot objectiveID="objective" attemptID="attempt" hash="sha256:code" />)
  const button = await screen.findByRole('button', { name: '明确开始验证' })
  expect(start).not.toHaveBeenCalled()
  fireEvent.click(button)
  await waitFor(() => expect(start).toHaveBeenCalledWith('objective', 'attempt', { request_id: '11111111-1111-4111-8111-111111111111', expected_input_hash: 'sha256:input' }, expect.any(AbortSignal)))
})

it.each(['inkwords.learner-go-test-offline.v1', 'inkwords.learner-go-test-offline.v999'])('rejects unsupported execution profile %s without starting code', async (profile) => {
  vi.spyOn(masteryService, 'learnerArtifact').mockResolvedValue(null)
  vi.spyOn(masteryService, 'learnerVerificationLatest').mockResolvedValue(null)
  vi.spyOn(masteryService, 'learnerVerificationPreview').mockResolvedValue({ capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: true, profile }, requires_explicit_start: true, input_hash: 'sha256:input' } as never)
  const start = vi.spyOn(masteryService, 'startLearnerVerification')
  render(<LearnerCodeSnapshot objectiveID="objective" attemptID="attempt" hash="sha256:code" />)
  await screen.findByText('代码隔离预检合同不匹配。')
  expect(screen.queryByRole('button', { name: '明确开始验证' })).toBeNull()
  expect(start).not.toHaveBeenCalled()
})

it('rejects a snapshot from another answer instead of displaying its files', async () => {
  vi.spyOn(masteryService, 'learnerVerificationLatest').mockResolvedValue(null)
  vi.spyOn(masteryService, 'learnerArtifact').mockResolvedValue({ format: 'inkwords.learner-artifact.v1', objective_id: 'other', attempt_id: 'attempt', snapshot_hash: 'sha256:code' } as never)
  vi.spyOn(masteryService, 'learnerVerificationPreview').mockResolvedValue({ capability: { format: 'inkwords.learner-verification-capability.v1', accepted: true, available: false, profile: 'inkwords.learner-go-test-offline.v2', reason: 'disabled' }, requires_explicit_start: true })
  render(<LearnerCodeSnapshot objectiveID="objective" attemptID="attempt" hash="sha256:code" />)
  await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('代码快照与本次作答不匹配'))
})
