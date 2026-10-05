// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { GenerationFailureDetails } from './GenerationFailureDetails'
import type { TextbookGenerationTaskSnapshot } from '@/services/textbook'

afterEach(cleanup)

const task = (diagnostics: unknown[]): TextbookGenerationTaskSnapshot => ({
  id: 'failed-sample', task_type: 'generation', task_subtype: 'textbook_sample_generate', status: 'failed', retry_count: 0,
  result: { result_version: 1, task_subtype: 'textbook_sample_generate', final_status: 'failed', candidate_persisted: false,
    markdown: 'whole rejected manuscript must stay hidden',
    quality_report_json: { passed: false, failures: ['undefined_repeated_acronym: POST'], failure_diagnostics: diagnostics } },
})
const diagnostic = { failure: 'undefined_repeated_acronym: POST', line: 47, occurrences: 3, area: 'practice_set', excerpt: '使用 POST 修改请求。' }

it('shows the bounded location without rendering a rejected manuscript', () => {
  render(<GenerationFailureDetails task={task([diagnostic])} />)
  expect(screen.getByText('练习与答案 · 原稿第 47 行 · 出现 3 次')).toBeTruthy()
  expect(screen.getByText(diagnostic.excerpt)).toBeTruthy()
  expect(screen.queryByText('whole rejected manuscript must stay hidden')).toBeNull()
})

it('rejects unrelated or malformed diagnostic payloads and supports old tasks', () => {
  render(<GenerationFailureDetails task={task([null, { ...diagnostic, failure: 'unrelated' }, { ...diagnostic, line: -1 }, { ...diagnostic, excerpt: '长'.repeat(161) }, { ...diagnostic, area: 'unknown' }])} />)
  expect(screen.queryByLabelText('失败位置与片段')).toBeNull()
  expect(screen.getByLabelText('自动检查失败项')).toBeTruthy()
})

it('renders diagnostic markup as inert text', () => {
  const markup = '<img src=x onerror=alert(1)> POST'
  const { container } = render(<GenerationFailureDetails task={task([{ ...diagnostic, excerpt: markup }])} />)
  expect(screen.getByText(markup)).toBeTruthy()
  expect(container.querySelector('img')).toBeNull()
})
