import { Clock3 } from 'lucide-react'
import { SectionHeader, StatusPill } from '@/components/ui/workspace'
import type { DueMasteryTask, MasterySkill } from '@/services/mastery'

const skillLabels: Record<MasterySkill, string> = {
  explain: '独立复述',
  complete: '补全代码',
  reproduce: '从零实现',
  transfer: '修改需求',
  diagnose: '定位故障',
  retain: '延迟复习',
}

interface TodayQueueProps {
  tasks: DueMasteryTask[]
  loading: boolean
  onStartTask?: (task: DueMasteryTask) => void
}

// Why: 任务队列只呈现服务端已经到期的投影；它没有计时器、轮询或推送能力。
export function TodayQueue({ tasks, loading, onStartTask }: TodayQueueProps) {
  return (
    <section className="summary-row" aria-live="polite">
      <SectionHeader
        eyebrow="到期巩固"
        title="本次打开的学习任务"
        description="仅在打开首页时检查，不会后台提醒。"
        action={<StatusPill tone="success">{tasks.length} 项</StatusPill>}
      />
      <div className="mt-4 space-y-3">
        {loading ? <p className="text-sm text-muted-foreground">正在检查到期任务...</p> : null}
        {!loading && tasks.length === 0 ? <p className="text-sm leading-6 text-muted-foreground">当前没有到期任务，下一次主动打开时会重新检查。</p> : null}
        {tasks.map((task) => (
          <article key={`${task.objective_id}-${task.skill}`} className="surface-inset px-3 py-3">
            <div className="flex items-start gap-2">
              <Clock3 className="mt-0.5 h-4 w-4 shrink-0 text-[var(--success)]" />
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-foreground">{task.title}</p>
                <p className="mt-1 text-xs text-muted-foreground">{skillLabels[task.skill]} · {task.reason}</p>
                {onStartTask ? <button type="button" className="mt-2 text-xs font-medium text-[var(--brand)] underline-offset-4 hover:underline" onClick={() => onStartTask(task)}>开始本次练习</button> : null}
              </div>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}
