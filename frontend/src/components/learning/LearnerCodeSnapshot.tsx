import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { masteryService, type LearnerArtifact, type LearnerVerificationJob } from '@/services/mastery'
import { learnerCodeError } from '@/lib/learnerCode'
import { useLearnerVerification } from '@/hooks/useLearnerVerification'

export function LearnerCodeSnapshot({ objectiveID, attemptID, hash }: { objectiveID: string; attemptID: string; hash: string }) {
  return <Snapshot key={JSON.stringify([objectiveID, attemptID, hash])} objectiveID={objectiveID} attemptID={attemptID} hash={hash} />
}

function Snapshot({ objectiveID, attemptID, hash }: { objectiveID: string; attemptID: string; hash: string }) {
  const [snapshot, setSnapshot] = useState<LearnerArtifact | null>(null)
  const [error, setError] = useState<string | null>(null)
  const { preview: verification, job, ready, previewError, statusError, actionError, busy, pending, start, cancel, refresh } = useLearnerVerification(objectiveID, attemptID, hash)
  const [reload, setReload] = useState(0)
  useEffect(() => {
    let active = true
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 10000)
    void masteryService.learnerArtifact(objectiveID, attemptID, controller.signal).then((saved) => {
      if (!saved || saved.snapshot_hash !== hash || saved.objective_id !== objectiveID || saved.attempt_id !== attemptID || saved.format !== 'inkwords.learner-artifact.v1' || !Array.isArray(saved.files) || !saved.files.length || learnerCodeError(saved.files)) throw new Error('代码快照与本次作答不匹配。')
      if (active) { setSnapshot(saved); setError(null) }
    }).catch((cause) => { if (active) setError(cause instanceof Error ? cause.message : '读取代码失败。') }).finally(() => clearTimeout(timer))
    return () => { active = false; controller.abort(); clearTimeout(timer) }
  }, [objectiveID, attemptID, hash, reload])
  return <section aria-label="本次代码快照" className="mt-4 space-y-2 border-t border-border pt-3 text-sm">
    <h4 className="font-medium">本次代码快照</h4>
    <p>已随原作答保存。模型评分与隔离运行是两个独立动作。</p>
    {error ? <p role="alert" className="text-destructive">{error}<Button type="button" variant="ghost" size="sm" onClick={() => setReload((value) => value + 1)}>重新读取代码</Button></p> : !snapshot ? <p>正在读取原文件…</p> : null}
    {snapshot ? <><p>{snapshot.files.length} 个文件 · {new Date(snapshot.submitted_at).toLocaleString()}</p>{snapshot.files.map((file) => <details key={file.path}><summary className="cursor-pointer break-all">{file.path}</summary><pre className="mt-2 whitespace-pre-wrap break-words rounded border border-border p-2 font-mono text-xs">{file.content}</pre></details>)}</> : null}
    <div className="rounded border border-border bg-muted/30 p-3 text-xs">
      <p className="font-medium">隔离执行范围已接受</p>
      {verification ? verification.capability.available ? <>
        <p className="mt-1">当前运行器已通过隔离预检；点击后才会执行固定的离线 Go 测试。本次代码不会获得网络、宿主路径、环境变量或任意命令权限。</p>
        {verification.derived_files?.length ? <p className="mt-1 text-muted-foreground">运行时会加入受信任的最小 go.mod；原代码快照不会改写。</p> : null}
      </> : <p className="mt-1 text-muted-foreground">当前无法安全执行新的验证。{verification.capability.reason ? `原因：${verification.capability.reason}` : ''}已有运行记录保留。</p> : previewError ? <p role="alert" className="mt-1 text-muted-foreground">{previewError}</p> : <p className="mt-1 text-muted-foreground">正在读取隔离预检…</p>}
      {job ? <VerificationResult job={job} /> : ready ? <p className="mt-1 text-muted-foreground">本次记录尚未运行。</p> : !statusError ? <p className="mt-1 text-muted-foreground">正在读取已有验证记录…</p> : null}
      {statusError ? <p role="alert" className="mt-2 text-destructive">{statusError}</p> : null}
      {actionError ? <p role="alert" className="mt-2 text-destructive">{actionError}</p> : null}
      <div className="mt-2 flex flex-wrap gap-2">
        {job?.status === 'queued' || job?.status === 'running' ? <Button type="button" size="sm" variant="outline" disabled={busy} onClick={() => void cancel()}>取消验证</Button> : pending && snapshot ? <Button type="button" size="sm" disabled={busy} onClick={() => void start()}>重试启动请求</Button> : ready && snapshot && verification?.capability.available ? <Button type="button" size="sm" disabled={busy} onClick={() => void start()}>{job ? '明确重新验证' : '明确开始验证'}</Button> : null}
        <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={refresh}>重新读取验证状态</Button>
      </div>
    </div>
  </section>
}

function VerificationResult({ job }: { job: LearnerVerificationJob }) {
  const labels: Record<LearnerVerificationJob['status'], string> = { queued: '等待运行', running: '正在隔离运行', passed: '学习者检查通过', failed: '学习者检查失败', timed_out: '运行超时', cancelled: '已取消', unavailable: '运行器不可用', runner_error: '隔离运行错误', interrupted: '进程中断' }
  return <div className="mt-2 space-y-1" role="status"><p>状态：{labels[job.status]}</p>{job.report?.execution_started ? <p className="text-muted-foreground">已开始执行；退出码：{job.report.exit_code ?? '未知'}；执行树：{job.report.execution_tree_hash?.slice(0, 19) ?? '未记录'}…</p> : <p className="text-muted-foreground">没有可证明的执行开始记录。</p>}{job.report?.reason ? <p className="text-muted-foreground">{job.report.reason}</p> : null}{job.report?.output ? <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded border border-border bg-background p-2 font-mono">{job.report.output}{job.report.output_truncated ? '\n[输出已截断]' : ''}</pre> : null}<p className="text-muted-foreground">学习者自带测试只记为学习者检查，不等同于题目独立验收。</p></div>
}
