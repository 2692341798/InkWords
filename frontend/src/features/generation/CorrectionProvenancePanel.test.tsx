// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import type { ChapterRevision } from '@/services/textbook'
import { CorrectionProvenancePanel } from './CorrectionProvenancePanel'
afterEach(cleanup)
it('separates the source model usage from the zero-call local correction', () => {
  render(<CorrectionProvenancePanel revision={{ provider_name: 'deepseek', model_name: 'deepseek-v4-flash', document_json: { correction: { origin: 'automated_local_correction', original_task_id: 'source-task', reason: '修正路由查询的副作用', original_usage_json: { known: true, input_tokens: 11663, output_tokens: 8100, provider_call_count: 1 } } } } as unknown as ChapterRevision} />)
  const panel = screen.getByRole('region', { name: '局部修订来源' })
  expect(panel.textContent).toContain('本次修订供应商调用：0 次')
  expect(panel.textContent).toContain('原模型调用的用量（不是本次修订用量）')
  expect(panel.textContent).toContain('输入 Token11663')
  expect(panel.textContent).toContain('不代表人工评审或代码运行验证')
})

it('shows both frozen identities for an explicit contract migration', () => {
  render(<CorrectionProvenancePanel revision={{ document_json: { correction: { origin: 'automated_local_correction', original_input_hash: 'sha256:old-source', target_input_hash: 'sha256:current-approved' } } } as unknown as ChapterRevision} />)
  const panel = screen.getByRole('region', { name: '局部修订来源' })
  expect(panel.textContent).toContain('历史合同迁移')
  expect(panel.textContent).toContain('原任务输入指纹：sha256:old-source')
  expect(panel.textContent).toContain('当前批准输入指纹：sha256:current-approved')
})
