import type { ChapterRevision } from '@/services/textbook'
import { UsagePanel } from './UsagePanel'
import { correctionProvenance } from './correctionProvenance'

export function CorrectionProvenancePanel({ revision }: { revision: ChapterRevision }) {
  const correction = correctionProvenance(revision)
  if (!correction) return <UsagePanel usage={revision.provider_usage_json} />
  const usage = correction.original_usage_json
  return <section aria-label="局部修订来源" className="mt-3 rounded-md border border-border p-3 text-xs">
    <h3 className="text-sm font-medium">自动局部修订 · 待审候选</h3>
    <p className="mt-2">本次修订供应商调用：0 次。原始模型：{revision.provider_name} / {revision.model_name}。局部复核不代表人工评审或代码运行验证。</p>
    <p className="mt-2">修订理由：{String(correction.reason ?? '未记录')}</p>
    <p className="mt-2 break-all">原失败任务：{String(correction.original_task_id ?? '未记录')}</p>
    <p className="mt-2 break-all">原稿收据：{String(correction.original_receipt_hash ?? '未记录')}</p>
    {typeof correction.target_input_hash === 'string' && correction.target_input_hash && <>
      <p className="mt-2">历史合同迁移：保留原任务来源，按当前合同复核并生成待审候选。</p>
      <p className="mt-2 break-all">原任务输入指纹：{String(correction.original_input_hash ?? '未记录')}</p>
      <p className="mt-2 break-all">当前批准输入指纹：{correction.target_input_hash}</p>
    </>}
    <p className="mt-3 font-medium">原模型调用的用量（不是本次修订用量）</p>
    <UsagePanel usage={usage && typeof usage === 'object' && !Array.isArray(usage) ? usage as Record<string, unknown> : undefined} />
  </section>
}
