// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import type { TextbookRuntimeEvidence } from '@/services/textbook'
import { RuntimeEvidenceDetails } from './RuntimeEvidenceDetails'

afterEach(cleanup)
const evidence: TextbookRuntimeEvidence = {
  id: 'receipt-1', revision_id: 'revision-1', code_artifact_id: 'artifact-1',
  code_artifact_hash: 'sha256:code', command_manifest_hash: 'sha256:manifest',
  input_hash: 'sha256:input', kind: 'terminal_output', status: 'unverified',
  output_truncated: false, created_at: '',
}

it('exposes the persisted exit code, literal output and full receipt identities without executing markup', () => {
  const raw = JSON.stringify({ format: 'inkwords.runtime-observation.v1', observations: { command: { kind: 'go_test' }, exit_code: 1, output: 'FAIL\n<img src=x onerror=alert(1)>\n' }, interpretations: null })
  const { container } = render(<RuntimeEvidenceDetails evidence={{ ...evidence, structured_output: raw }} />)
  fireEvent.click(screen.getByText('查看运行详情与原始记录'))
  expect(screen.getByText('退出码：1')).toBeTruthy()
  expect(screen.getByLabelText('原始工具输出').textContent).toBe('FAIL\n<img src=x onerror=alert(1)>\n')
  expect(screen.getByLabelText('原始运行记录').textContent).toBe(raw)
  expect(screen.getByText('receipt-1')).toBeTruthy()
  expect(screen.getByText('sha256:manifest')).toBeTruthy()
  expect(container.querySelector('img')).toBeNull()
  expect(screen.queryByText('已验证')).toBeNull()
})

it('keeps missing or malformed observations unknown instead of inventing a zero exit code', () => {
  render(<RuntimeEvidenceDetails evidence={{ ...evidence, structured_output: '{invalid', output_truncated: true }} />)
  expect(screen.getByText('无法解析为当前运行观察合同，请保留原始记录并核对来源。')).toBeTruthy()
  expect(screen.getByText('输出已截断，不能据此判断未显示的内容。')).toBeTruthy()
  expect(screen.getByLabelText('原始运行记录').textContent).toBe('{invalid')
  expect(screen.queryByText('退出码：0')).toBeNull()
})

it('does not synthesize an original record when none was saved', () => {
  render(<RuntimeEvidenceDetails evidence={evidence} />)
  expect(screen.getByText('未保存结构化原始记录。')).toBeTruthy()
  expect(screen.queryByLabelText('原始运行记录')).toBeNull()
})
