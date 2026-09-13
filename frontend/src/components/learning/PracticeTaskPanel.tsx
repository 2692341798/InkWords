import { Button } from '@/components/ui/button'
import type { PracticeTask } from '@/services/textbook'

interface Props { task: PracticeTask; disabled: boolean; hintsShown: number; answerShown: boolean; onHint: (level: number) => void; onAnswer: () => void }

// Task text is read from the matching approved revision, not edited in a
// separate learning copy. Hints and the answer key require explicit disclosure.
export function PracticeTaskPanel({ task, disabled, hintsShown, answerShown, onHint, onAnswer }: Props) {
  return <section aria-label="本次具体练习" className="mt-4 space-y-4 rounded-md border border-border p-4 text-sm">
    <div><h3 className="font-medium">本次题目</h3><p className="mt-2 whitespace-pre-wrap">{task.prompt}</p></div>
    <p className="text-muted-foreground">变式：{task.variation}</p>
    {task.min_delay_hours > 0 ? <p>延迟要求：至少间隔 {task.min_delay_hours} 小时，不看原文再完成。</p> : null}
    <div><h3 className="font-medium">本题评分依据</h3><ul className="mt-2 list-disc space-y-1 pl-5">{task.rubric.map((criterion) => <li key={criterion.id}>{criterion.description}{criterion.requires_runtime ? <span className="text-muted-foreground">（需要本次工件的运行证据）</span> : null}</li>)}</ul></div>
    {task.hints.slice(0, hintsShown).map((hint) => <p key={hint.level}>提示 {hint.level}：{hint.text}</p>)}
    {hintsShown < task.hints.length ? <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={() => onHint(hintsShown + 1)}>查看第 {hintsShown + 1} 层提示（记 1 次）</Button> : null}
    {answerShown ? <div><h3 className="font-medium">参考答案</h3><p className="mt-2 whitespace-pre-wrap">{task.expected_answer}</p><p className="mt-2 text-xs text-muted-foreground">参考答案是教材内容，不是代码运行或掌握证明。</p></div> : <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={onAnswer}>查看参考答案（本次不计独立完成）</Button>}
    <details><summary className="cursor-pointer">本题来源</summary><ul className="mt-2 break-all">{task.evidence_ids.map((id) => <li key={id}>{id}</li>)}</ul></details>
  </section>
}
