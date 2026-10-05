import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import type { DueMasteryTask, MasteryAttemptInput, MasteryObjectiveWorkspace, MasterySkill, NextMasteryTask, PracticeHelpInput, PracticeSession } from '@/services/mastery'
import { PracticeTaskPanel } from './PracticeTaskPanel'
import { SavedMasteryAnswer } from './SavedMasteryAnswer'
import { LearnerCodeInput } from './LearnerCodeInput'
import { learnerCodeError } from '@/lib/learnerCode'
import type { LearnerCodeFile } from '@/services/mastery'

const skillLabels: Record<MasterySkill, string> = {
  explain: '独立复述', complete: '补全代码', reproduce: '从零实现', transfer: '修改需求', diagnose: '定位故障', retain: '延迟复习',
}

function pendingSession(key: string): string | undefined {
  try { return sessionStorage.getItem(key) || undefined } catch { return undefined }
}

function rememberSession(key: string, id?: string) {
  try { if (id) sessionStorage.setItem(key, id); else sessionStorage.removeItem(key) } catch { /* Server still resumes open sessions when browser storage is unavailable. */ }
}

interface MasterySessionProps {
  task: DueMasteryTask
  onRecord: (objectiveID: string, input: MasteryAttemptInput) => Promise<NextMasteryTask>
  onLoad: (objectiveID: string, skill: MasterySkill, sessionID?: string) => Promise<MasteryObjectiveWorkspace>
  onHelp?: (objectiveID: string, sessionID: string, input: PracticeHelpInput) => Promise<PracticeSession>
  onRecorded: () => void
  onClose: () => void
}

// This records self-reported performance only. It deliberately does not claim
// that a free-text answer or an uploaded file was automatically scored.
export function MasterySession({ task, onRecord, onLoad, onHelp, onRecorded, onClose }: MasterySessionProps) {
  const startedAt = useMemo(() => Date.now(), [])
  const [answer, setAnswer] = useState('')
  const [codeFiles, setCodeFiles] = useState<LearnerCodeFile[]>([])
  const [codeBusy, setCodeBusy] = useState(false)
  const [workspace, setWorkspace] = useState<MasteryObjectiveWorkspace | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reload, setReload] = useState(0)
  const [showHint, setShowHint] = useState(false)
  const [usedAnswer, setUsedAnswer] = useState(false)
  const [correct, setCorrect] = useState<boolean | null>(null)
  const [independent, setIndependent] = useState(false)
  const [hintCount, setHintCount] = useState(0)
  const [revealedHelpCount, setRevealedHelpCount] = useState(0)
  const [confidence, setConfidence] = useState(3)
  const [errorKinds, setErrorKinds] = useState('')
  const [result, setResult] = useState<NextMasteryTask | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [helpBusy, setHelpBusy] = useState(false)
  const sessionKey = `inkwords:practice-session:${task.objective_id}:${task.skill}`
  const sessionID = useRef<string | undefined>(pendingSession(sessionKey))

  const restoreHelp = (session: PracticeSession) => {
    setRevealedHelpCount(session.hint_count)
    setHintCount((value) => Math.max(value, session.hint_count))
    setUsedAnswer(session.answer_shown)
    if (session.answer_shown) setIndependent(false)
    if (session.result) setResult(session.result)
  }

  useEffect(() => {
    let active = true
    setLoadError(null)
    onLoad(task.objective_id, task.skill, sessionID.current).then((data) => {
      if (active) {
        setWorkspace(data); setLoadError(data.practice_error || null)
        if (data.practice_session) {
          sessionID.current = data.practice_session.id
          rememberSession(sessionKey, data.practice_session.result ? undefined : data.practice_session.id)
          restoreHelp(data.practice_session)
          const saved = data.practice_session.result && data.attempts.find((attempt) => attempt.practice_session_id === data.practice_session?.id)
          if (saved) {
            setAnswer(saved.answer || '')
            setCorrect(saved.correct)
            setIndependent(saved.independent && !data.practice_session.answer_shown)
            setHintCount(Math.max(saved.hint_count, data.practice_session.hint_count))
            setConfidence(saved.confidence)
            setErrorKinds((saved.error_kinds ?? []).join('，'))
          }
        }
      }
    }).catch((cause) => {
      if (active) setLoadError(cause instanceof Error ? cause.message : '读取学习目标失败')
    })
    return () => { active = false }
  }, [onLoad, task.objective_id, task.skill, reload, sessionKey])

  const revealHelp = async (input: PracticeHelpInput) => {
    if (!onHelp || !sessionID.current || helpBusy) return
    setHelpBusy(true)
    setError(null)
    try {
      const session = await onHelp(task.objective_id, sessionID.current, input)
      if (session.id !== sessionID.current || session.objective_id !== task.objective_id || session.practice_content_hash !== workspace?.objective.practice_content_hash) throw new Error('帮助记录与当前练习不匹配，请重新读取。')
      setWorkspace((current) => current ? { ...current, practice_session: session } : current)
      restoreHelp(session)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '保存帮助记录失败，内容尚未展开，请重试。')
    } finally { setHelpBusy(false) }
  }

  const submit = async () => {
    if (codeBusy) return
    const codeError = learnerCodeError(codeFiles)
    if (codeError) { setError(codeError); return }
    if (correct === null) {
      setError('请先如实选择本次是否达到目标。')
      return
    }
    if (!answer.trim() || Array.from(answer).length > 20000) {
      setError('请填写本次作答，最多 20,000 字符。')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const next = await onRecord(task.objective_id, {
        skill: task.skill, answer, code_files: codeFiles.length ? codeFiles : undefined, practice_task_id: workspace?.practice?.id, practice_content_hash: workspace?.objective.practice_content_hash, practice_session_id: sessionID.current, correct, independent: independent && !usedAnswer, hint_count: Math.max(hintCount, revealedHelpCount),
        took_millis: Math.max(0, Date.now() - startedAt), confidence,
        error_kinds: errorKinds.split(/[，,\n]/).map((item) => item.trim()).filter(Boolean), attempted_at: new Date().toISOString(),
      })
      setResult(next)
      rememberSession(sessionKey)
      setReload((value) => value + 1)
      onRecorded()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '记录学习表现失败')
    } finally {
      setSubmitting(false)
    }
  }

  return <Panel className="mt-4 p-5" aria-label="掌握练习会话">
    <SectionHeader eyebrow="当前练习" title={task.title} description={`${skillLabels[task.skill]}：${task.reason}`} action={<Button type="button" variant="ghost" size="sm" onClick={onClose}>关闭</Button>} />
    <p className="mt-4 text-sm text-muted-foreground">先独立作答，再对照评分依据自评。展开已保存作答可发起模型逐项评分；自评与自动反馈分别记录，代码不会在此执行。</p>
    {loadError ? <div role="alert" className="mt-3 text-sm text-destructive">{loadError}<Button type="button" variant="ghost" onClick={() => setReload((value) => value + 1)}>重新读取</Button></div> : null}
    {!workspace && !loadError ? <p className="mt-3 text-sm">正在读取学习目标…</p> : null}
    {workspace?.practice ? <PracticeTaskPanel key={workspace.practice.id} task={workspace.practice} hintsShown={workspace.practice_session?.hints_shown ?? 0} answerShown={workspace.practice_session?.answer_shown ?? false} disabled={submitting || helpBusy || !!result || !onHelp || !workspace.practice_session} onHint={(level) => void revealHelp({ kind: 'hint', level })} onAnswer={() => void revealHelp({ kind: 'answer', level: 0 })} /> : workspace ? <section aria-label="本次学习目标" className="mt-4 space-y-3 rounded-md border border-border p-4 text-sm">
      <p>{workspace.objective.behavior}</p>
      <div><h3 className="font-medium">评分依据</h3><ul className="mt-1 list-disc pl-5">{workspace.objective.rubric.map((item, index) => <li key={index}>{item}</li>)}</ul></div>
      {workspace.objective.prerequisites?.length ? <p>前置要求：{workspace.objective.prerequisites.join('；')}</p> : null}
      {!showHint ? <Button type="button" variant="outline" size="sm" disabled={submitting || !!result} onClick={() => { setShowHint(true); setRevealedHelpCount((value) => value + 1); setHintCount((value) => value + 1) }}>查看关键点提示（记 1 次提示）</Button> : <ul aria-label="关键点提示" className="list-disc pl-5">{workspace.objective.key_points.map((item, index) => <li key={index}>{item}</li>)}</ul>}
      <details><summary className="cursor-pointer">来源依据</summary><ul className="mt-1 break-all">{workspace.objective.evidence_refs.map((item, index) => <li key={index}>{item}</li>)}</ul></details>
    </section> : null}
    <label className="mt-4 grid gap-2 text-sm">本次作答<textarea aria-label="本次作答" value={answer} onChange={(event) => setAnswer(event.target.value)} disabled={submitting || !!result} rows={8} placeholder="写下你的解释、代码或排错过程，并注明观察到的结果。" className="w-full rounded-md border border-border bg-background p-3 font-mono focus-visible:outline-2 focus-visible:outline-ring" /><span className="text-xs text-muted-foreground">{Array.from(answer).length.toLocaleString()} / 20,000 字符 · 提交失败时保留此处文本</span></label>
    <fieldset disabled={submitting || !!result} className="mt-4 grid gap-3 rounded-md border border-border p-4"><legend className="px-1 text-sm font-medium">本次是否达到目标？</legend><div className="flex flex-wrap gap-4 text-sm"><label><input type="radio" name="mastery-correct" checked={correct === true} onChange={() => setCorrect(true)} /> 达到</label><label><input type="radio" name="mastery-correct" checked={correct === false} onChange={() => setCorrect(false)} /> 未达到</label></div><label className="text-sm"><input type="checkbox" disabled={usedAnswer} checked={independent} onChange={(event) => setIndependent(event.target.checked)} /> 独立完成，没有依赖完整答案</label><label className="grid gap-1 text-sm">提示次数<input aria-label="提示次数" type="number" min={revealedHelpCount} value={hintCount} onChange={(event) => setHintCount(Math.max(revealedHelpCount, Number(event.target.value) || 0))} className="w-28 rounded-md border border-border bg-background px-2 py-1" /></label><label className="grid gap-1 text-sm">信心（1 最低，5 最高）<select aria-label="信心" value={confidence} onChange={(event) => setConfidence(Number(event.target.value))} className="w-40 rounded-md border border-border bg-background px-2 py-1">{[1, 2, 3, 4, 5].map((value) => <option key={value} value={value}>{value}</option>)}</select></label><label className="grid gap-1 text-sm">错误类别（可选，用逗号分隔）<input aria-label="错误类别" value={errorKinds} onChange={(event) => setErrorKinds(event.target.value)} placeholder="例如：漏掉因果链" className="rounded-md border border-border bg-background px-2 py-1" /></label></fieldset>
    {workspace?.practice && !result ? <LearnerCodeInput files={codeFiles} onChange={setCodeFiles} onBusy={setCodeBusy} disabled={submitting || helpBusy} /> : null}
    {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
    {result ? <div className="mt-4 rounded-md border border-[var(--brand)]/30 bg-[var(--brand)]/5 p-3 text-sm"><StatusPill tone="brand">已记录</StatusPill><p className="mt-2">作答保存时的安排：{skillLabels[result.skill]} · {result.reason}</p><p className="mt-1 text-xs text-muted-foreground">建议时间：{new Date(result.due_at).toLocaleString()}。后续应用评分可更新安排；只有六维要求和延迟保持证据齐全才可判为掌握。</p></div> : <Button type="button" className="mt-4" disabled={submitting || helpBusy || codeBusy || !workspace || !!loadError || (!!workspace.practice && !workspace.practice_session)} onClick={() => void submit()}>{submitting ? '正在记录…' : '记录本次表现'}</Button>}
    {workspace?.attempts.length ? <section aria-label="已保存作答" className="mt-5 space-y-3 text-sm"><h3 className="font-medium">最近 20 次作答与自评</h3>{[...workspace.attempts].reverse().map((attempt) => <SavedMasteryAnswer key={attempt.id} objectiveID={task.objective_id} attempt={attempt} label={skillLabels[attempt.skill]} onApplied={onRecorded} />)}</section> : null}
  </Panel>
}
