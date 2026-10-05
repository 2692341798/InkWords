import { Button } from '@/components/ui/button'
import type { SampleGenerationPreflight } from '@/services/textbook'

interface GenerationPreflightPanelProps {
	preflight: SampleGenerationPreflight
	disabled: boolean
	onConfirm: () => void
	onCancel: () => void
}

const number = new Intl.NumberFormat('zh-CN')

export function GenerationPreflightPanel({ preflight, disabled, onConfirm, onCancel }: GenerationPreflightPanelProps) {
	const target = preflight.generation_target
	return (
		<section aria-label="样章生成前预检" className="mt-3 rounded-md border border-border bg-muted/20 p-4">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<h3 className="font-medium">请确认本次样章生成</h3>
				<span className={preflight.within_budget ? 'text-sm text-emerald-700 dark:text-emerald-300' : 'text-sm text-destructive'}>{preflight.within_budget ? '预估在预算内' : '预估超出预算'}</span>
			</div>
			<dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
				<div><dt className="text-muted-foreground">生成目标</dt><dd className="font-mono">{target.provider_name} / {target.model_name}</dd></div>
				<div><dt className="text-muted-foreground">引用证据</dt><dd>{number.format(preflight.evidence_reference_count)} 段</dd></div>
				<div><dt className="text-muted-foreground">估算输入 Token</dt><dd>约 {number.format(preflight.estimated_input_tokens)} / {number.format(preflight.allowed_input_tokens)}</dd></div>
				<div><dt className="text-muted-foreground">预留输出 Token</dt><dd>{number.format(preflight.reserved_output_tokens)}</dd></div>
				<div><dt className="text-muted-foreground">缓存</dt><dd>工作器执行前未知</dd></div>
				<div><dt className="text-muted-foreground">预估费用</dt><dd>未配置经版本化核对的价格表</dd></div>
			</dl>
			<p className="mt-3 break-all text-xs text-muted-foreground">冻结输入：{preflight.input_hash}</p>
			<p className="mt-2 text-xs text-muted-foreground">这是服务端保守估算，不是供应商账单。确认后只能为这一份冻结输入入队；如果资料或合同已变化，服务端会拒绝旧确认。</p>
			<div className="mt-3 flex flex-wrap gap-2">
				<Button type="button" disabled={disabled || !preflight.within_budget} onClick={onConfirm}>确认并创建任务</Button>
				<Button type="button" variant="outline" disabled={disabled} onClick={onCancel}>取消</Button>
			</div>
		</section>
	)
}
