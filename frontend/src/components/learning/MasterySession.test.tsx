// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MasterySession } from './MasterySession'

const task = { objective_id: 'objective-1', chapter_id: 'chapter-1', title: '解释路由登记', skill: 'explain' as const, due_at: '2026-09-03T00:00:00Z', reason: '先建立因果链。' }
const workspace = { objective: { id: 'objective-1', chapter_id: 'chapter-1', title: task.title, behavior: '独立说明方法与路径怎样选择处理函数。', required_skills: ['explain' as const], rubric: ['准确解释选择过程'], key_points: ['先选择方法树'], prerequisites: [], evidence_refs: ['evidence:gin-routing'] }, attempts: [] }
const onLoad = vi.fn().mockResolvedValue(workspace)
afterEach(() => { cleanup(); sessionStorage.clear() })

describe('MasterySession', () => {
  it('keeps historical answers visible while blocking an unbound practice submission', async () => {
    const onRecord = vi.fn()
    const saved = { id: 'attempt-saved', skill: 'explain' as const, answer: '这份原作答仍应可读。', correct: true, independent: true, hint_count: 0, took_millis: 1000, confidence: 4, error_kinds: [], attempted_at: task.due_at }
    render(<MasterySession task={task} onLoad={async () => ({ ...workspace, attempts: [saved], practice_error: '服务端题目依据不匹配。' })} onRecord={onRecord} onRecorded={vi.fn()} onClose={vi.fn()} />)
    await screen.findByText(saved.answer)
    expect(screen.getByRole('alert').textContent).toContain('服务端题目依据不匹配')
    expect((screen.getByRole('button', { name: '记录本次表现' }) as HTMLButtonElement).disabled).toBe(true)
    expect(onRecord).not.toHaveBeenCalled()
  })

  it('keeps disclosed help in the saved attempt and excludes answer-assisted independence', async () => {
    const practice = { id: 'practice-1', mode: 'explain' as const, prompt: '解释两条调用链', variation: '换一条路径', expected_answer: '登记后再查找', rubric: [], hints: [{ level: 1, text: '先区分时间点' }], evidence_ids: ['source-1'], min_delay_hours: 0 }
    const onRecord = vi.fn().mockResolvedValue({ skill: 'explain', due_at: task.due_at, reason: '降低提示依赖。' })
    const session = { id: 'session-1', objective_id: task.objective_id, skill: task.skill, practice_task_id: practice.id, practice_content_hash: 'sha256:fixture', started_at: task.due_at, hints_shown: 0, answer_shown: false, hint_count: 0 }
    const onHelp = vi.fn().mockRejectedValueOnce(new Error('帮助记录保存失败')).mockResolvedValueOnce({ ...session, hints_shown: 1, hint_count: 1 }).mockResolvedValueOnce({ ...session, hints_shown: 1, answer_shown: true, hint_count: 2 })
    render(<MasterySession task={task} onLoad={async () => ({ ...workspace, objective: { ...workspace.objective, practice_content_hash: session.practice_content_hash }, practice, practice_session: session })} onHelp={onHelp} onRecord={onRecord} onRecorded={vi.fn()} onClose={vi.fn()} />)
    await screen.findByText(practice.prompt)
    fireEvent.click(screen.getByLabelText('独立完成，没有依赖完整答案'))
    fireEvent.click(screen.getByRole('button', { name: '查看第 1 层提示（记 1 次）' }))
    await screen.findByText('帮助记录保存失败')
    expect(screen.queryByText('提示 1：先区分时间点')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '查看第 1 层提示（记 1 次）' }))
    await screen.findByText('提示 1：先区分时间点')
    fireEvent.click(screen.getByRole('button', { name: '查看参考答案（本次不计独立完成）' }))
    await screen.findByText(practice.expected_answer)
    expect((screen.getByLabelText('独立完成，没有依赖完整答案') as HTMLInputElement).disabled).toBe(true)
    fireEvent.change(screen.getByLabelText('提示次数'), { target: { value: '0' } })
    fireEvent.change(screen.getByLabelText('本次作答'), { target: { value: '在提示帮助下完成解释。' } })
    fireEvent.click(screen.getByLabelText('达到'))
    fireEvent.click(screen.getByRole('button', { name: '记录本次表现' }))
    await waitFor(() => expect(onRecord).toHaveBeenCalledOnce())
    expect(onRecord.mock.calls[0][1]).toMatchObject({ independent: false, hint_count: 2, practice_session_id: session.id })
  })

  it('records structured learner performance and shows the server-selected next task', async () => {
    const onRecord = vi.fn().mockResolvedValue({ skill: 'diagnose', due_at: '2026-09-04T00:00:00Z', reason: '需要练习定位根因。' })
    const onRecorded = vi.fn()
    render(<MasterySession task={task} onLoad={onLoad} onRecord={onRecord} onRecorded={onRecorded} onClose={vi.fn()} />)
    await screen.findByText(workspace.objective.behavior)
    fireEvent.change(screen.getByLabelText('本次作答'), { target: { value: '先按方法选择路由树，再按路径查找处理函数。' } })

    fireEvent.click(screen.getByLabelText('达到'))
    fireEvent.click(screen.getByLabelText('独立完成，没有依赖完整答案'))
    fireEvent.change(screen.getByLabelText('提示次数'), { target: { value: '1' } })
    fireEvent.change(screen.getByLabelText('错误类别'), { target: { value: '漏掉因果链，术语不清' } })
    fireEvent.click(screen.getByRole('button', { name: '记录本次表现' }))

    await waitFor(() => expect(onRecord).toHaveBeenCalledOnce())
    expect(onRecord.mock.calls[0][0]).toBe('objective-1')
    expect(onRecord.mock.calls[0][1]).toMatchObject({ skill: 'explain', answer: '先按方法选择路由树，再按路径查找处理函数。', correct: true, independent: true, hint_count: 1, error_kinds: ['漏掉因果链', '术语不清'] })
    expect(onRecorded).toHaveBeenCalledOnce()
    expect(screen.getByText('作答保存时的安排：定位故障 · 需要练习定位根因。')).toBeTruthy()
  })

  it('requires an explicit outcome before recording an attempt', async () => {
    const onRecord = vi.fn()
    render(<MasterySession task={task} onLoad={onLoad} onRecord={onRecord} onRecorded={vi.fn()} onClose={vi.fn()} />)
    await screen.findByText(workspace.objective.behavior)

    fireEvent.click(screen.getByRole('button', { name: '记录本次表现' }))

    expect(screen.getByRole('alert').textContent).toContain('请先如实选择本次是否达到目标')
    expect(onRecord).not.toHaveBeenCalled()
  })

  it('keeps the answer after a failed save and counts a requested key-point hint', async () => {
    const onRecord = vi.fn().mockRejectedValue(new Error('保存失败，请重试'))
    render(<MasterySession task={task} onLoad={onLoad} onRecord={onRecord} onRecorded={vi.fn()} onClose={vi.fn()} />)
    await screen.findByText(workspace.objective.behavior)
    expect(screen.queryByText('先选择方法树')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '查看关键点提示（记 1 次提示）' }))
    expect((screen.getByLabelText('提示次数') as HTMLInputElement).value).toBe('1')
    fireEvent.click(screen.getByLabelText('达到'))
    fireEvent.change(screen.getByLabelText('本次作答'), { target: { value: '这份作答不能因网络失败而丢失。' } })
    fireEvent.click(screen.getByRole('button', { name: '记录本次表现' }))
    await screen.findByText('保存失败，请重试')
    expect((screen.getByLabelText('本次作答') as HTMLTextAreaElement).value).toBe('这份作答不能因网络失败而丢失。')
  })

  it('requires answer text and renders saved code or HTML as inert text', async () => {
    const onRecord = vi.fn()
    const answer = '<script>window.shouldNeverRun = true</script>'
    render(<MasterySession task={task} onLoad={async () => ({ ...workspace, attempts: [{ id: 'attempt-1', skill: 'explain', answer, correct: false, independent: false, hint_count: 0, took_millis: 1000, confidence: 3, error_kinds: [], attempted_at: task.due_at }] })} onRecord={onRecord} onRecorded={vi.fn()} onClose={vi.fn()} />)
    await screen.findByText(answer)
    expect(document.querySelector('script')).toBeNull()
    fireEvent.click(screen.getByLabelText('达到'))
    fireEvent.click(screen.getByRole('button', { name: '记录本次表现' }))
    expect(screen.getByRole('alert').textContent).toContain('请填写本次作答')
    expect(onRecord).not.toHaveBeenCalled()
  })
})
