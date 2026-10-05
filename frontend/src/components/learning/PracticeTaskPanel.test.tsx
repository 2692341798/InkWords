// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PracticeTask } from '@/services/textbook'
import { PracticeTaskPanel } from './PracticeTaskPanel'

afterEach(cleanup)
const task: PracticeTask = { id: 'task-1', mode: 'explain', prompt: '说明请求方法和路径如何共同决定处理函数。', variation: '请求缺少 /api 前缀时会发生什么？', expected_answer: '先按方法选树，再按路径查找处理函数。', rubric: [{ id: 'accuracy', description: '区分登记与查找阶段。', requires_runtime: false }], hints: [{ level: 1, text: '先回忆两个查找条件。' }, { level: 2, text: '第一步选择方法树。' }, { level: 3, text: '对照树中的路径分段。' }], evidence_ids: ['source-1'], min_delay_hours: 0 }

describe('PracticeTaskPanel', () => {
  it('reveals hints one level at a time and requires an explicit answer disclosure', () => {
    const onHint = vi.fn(), onAnswer = vi.fn()
    const { rerender } = render(<PracticeTaskPanel task={task} disabled={false} hintsShown={0} answerShown={false} onHint={onHint} onAnswer={onAnswer} />)
    expect(screen.getByText(task.prompt)).toBeTruthy()
    expect(screen.queryByText(task.expected_answer)).toBeNull()
    expect(screen.queryByText('提示 2：第一步选择方法树。')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '查看第 1 层提示（记 1 次）' }))
    expect(onHint).toHaveBeenCalledOnce()
    expect(onHint).toHaveBeenCalledWith(1)
    expect(screen.queryByText('提示 1：先回忆两个查找条件。')).toBeNull()
    rerender(<PracticeTaskPanel task={task} disabled={false} hintsShown={1} answerShown={false} onHint={onHint} onAnswer={onAnswer} />)
    expect(screen.getByText('提示 1：先回忆两个查找条件。')).toBeTruthy()
    expect(screen.queryByText('提示 2：第一步选择方法树。')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '查看参考答案（本次不计独立完成）' }))
    expect(screen.queryByText(task.expected_answer)).toBeNull()
    rerender(<PracticeTaskPanel task={task} disabled={false} hintsShown={1} answerShown onHint={onHint} onAnswer={onAnswer} />)
    expect(screen.getByText(task.expected_answer)).toBeTruthy()
    expect(onAnswer).toHaveBeenCalledOnce()
  })
})
