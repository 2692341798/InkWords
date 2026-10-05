import type { AssessmentDecision, AssessmentTaskReference } from '@/services/masteryAssessment'

const statusLabels: Record<AssessmentDecision['status'], string> = {
  satisfied: '满足本项要求', unsatisfied: '尚未满足本项要求', unknown: '无法判断',
}

export function AssessmentDecisionDetails({ decisions, reference }: { decisions?: AssessmentDecision[]; reference?: AssessmentTaskReference }) {
  return <>
    {decisions?.length ? <details>
      <summary>原始模型对评分要求的判断</summary>
      <p className="mt-2 text-xs text-muted-foreground">这里保留模型最初的判断，供核对与纠正。当前分数和纠正记录见上方评分；结构检查通过仍需核对判断是否准确。</p>
      {decisions.map((decision) => <div key={decision.criterion_id} className="mt-3 space-y-1 rounded-md border border-border p-3">
        <p className="font-medium">{statusLabels[decision.status]}</p>
        <p className="whitespace-pre-wrap">本项冻结要求：{decision.requirement}</p>
        {decision.gaps.map((gap, index) => <div key={index} className="space-y-1 border-l-2 border-border pl-3">
          <blockquote className="whitespace-pre-wrap text-sm">要求原文：{gap.requirement_quote}</blockquote>
          <p className="whitespace-pre-wrap text-sm">模型指出的缺口：{gap.reason}</p>
        </div>)}
      </div>)}
    </details> : null}
    {reference ? <details>
      <summary>本次冻结参考答案</summary>
      <p className="mt-2 text-xs text-muted-foreground">参考答案用于核对任务意图；满足约束的其他正确解法也应认可。运行结论仍须有本次作答的实际验证记录。</p>
      <p className="mt-2 whitespace-pre-wrap">{reference.expected_answer}</p>
    </details> : null}
  </>
}
