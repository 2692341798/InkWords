// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { AssessmentDecisionDetails } from './AssessmentDecisionDetails'

afterEach(cleanup)

describe('AssessmentDecisionDetails', () => {
  it('keeps frozen requirements, gaps and authored answer distinguishable from current scores', () => {
    render(<AssessmentDecisionDetails decisions={[{ criterion_id: 'causality', requirement: '解释方法隔离的原因', status: 'unsatisfied', gaps: [{ requirement_quote: '原因', reason: '原文只有调用顺序' }] }]} reference={{ task_id: 'explain', practice_content_hash: 'sha256:fixture', expected_answer: '<script>这是参考答案数据</script>' }} />)
    expect(screen.getByText('原始模型对评分要求的判断')).toBeTruthy()
    expect(screen.getByText('尚未满足本项要求')).toBeTruthy()
    expect(screen.getByText('本项冻结要求：解释方法隔离的原因')).toBeTruthy()
    expect(screen.getByText('要求原文：原因')).toBeTruthy()
    expect(screen.getByText('模型指出的缺口：原文只有调用顺序')).toBeTruthy()
    expect(screen.getByText('<script>这是参考答案数据</script>')).toBeTruthy()
    expect(document.querySelector('script')).toBeNull()
  })

  it('does not invent judgments or reference answers for saved legacy feedback', () => {
    const { container } = render(<AssessmentDecisionDetails />)
    expect(container.innerHTML).toBe('')
  })
})
