import { Button } from '@/components/ui/button';
import { StatusPill } from '@/components/ui/workspace';
import type { CandidateReview } from '@/services/textbook';

interface Props {
 candidateReview?: CandidateReview | null;
 isBusy: boolean;
 lockIsMine: boolean;
 rejectReason: string;
 setRejectReason: (value: string) => void;
 rejectReasonIsValid: boolean;
 rejectReasonLength: number;
 onReject: () => Promise<void>;
}

export function CandidateDecisionPanel({candidateReview, isBusy, lockIsMine, rejectReason, setRejectReason, rejectReasonIsValid, rejectReasonLength, onReject}: Props) {
 return (
        <section
          aria-label="候选稿人工决定"
          className="mt-4 rounded-md border border-border p-4"
        >
          {candidateReview?.decision === "approved" ? (
            <div role="status">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="font-medium text-emerald-700 dark:text-emerald-300">
                  {candidateReview.human_review?.reviewer_kind === "delegated_ai" ? "该候选稿已由用户授权 AI 审阅并批准" : "该候选稿已人工审阅并批准"}
                </h3>
                <StatusPill tone="success">已批准</StatusPill>
              </div>
              <p className="mt-2 whitespace-pre-wrap text-sm">
                {candidateReview.reason}
              </p>
              {candidateReview.human_review?.reviewer_kind === "delegated_ai" && <p className="mt-2 whitespace-pre-wrap text-sm">授权说明：{candidateReview.human_review.delegation_note}（AI 审阅，不属于真人试学或出版社认证。）</p>}
              <dl className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                <div>
                  <dt>决定时间</dt>
                  <dd>
                    {new Date(candidateReview.created_at).toLocaleString()}
                  </dd>
                </div>
                <div>
                  <dt>质量合同</dt>
                  <dd className="font-mono">
                    {candidateReview.quality_contract_version}
                  </dd>
                </div>
                <div>
                  <dt>审阅量表</dt>
                  <dd className="font-mono">
                    {candidateReview.human_review?.contract_version ?? "缺失"}
                  </dd>
                </div>
                <div>
                  <dt>候选稿内容哈希</dt>
                  <dd className="break-all font-mono">
                    {candidateReview.candidate_content_hash}
                  </dd>
                </div>
              </dl>
              <ul className="mt-3 grid gap-1 text-xs sm:grid-cols-2">
                {candidateReview.human_review?.dimension_scores?.map(
                  (score) => (
                    <li key={score.dimension}>
                      {score.dimension}：{score.score} / 4
                    </li>
                  ),
                )}
              </ul>
            </div>
          ) : candidateReview ? (
            <div role="status">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="font-medium text-destructive">该候选稿已驳回</h3>
                <StatusPill tone="warning">不可应用</StatusPill>
              </div>
              <p className="mt-2 whitespace-pre-wrap text-sm">
                {candidateReview.reason}
              </p>
              <dl className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                <div>
                  <dt>决定时间</dt>
                  <dd>
                    {new Date(candidateReview.created_at).toLocaleString()}
                  </dd>
                </div>
                <div>
                  <dt>质量合同</dt>
                  <dd className="font-mono">
                    {candidateReview.quality_contract_version}
                  </dd>
                </div>
                <div className="sm:col-span-2">
                  <dt>候选稿内容哈希</dt>
                  <dd className="break-all font-mono">
                    {candidateReview.candidate_content_hash}
                  </dd>
                </div>
              </dl>
              <p className="mt-3 text-xs text-muted-foreground">
                驳回记录不可改写；请生成新的候选稿继续迭代。
              </p>
            </div>
          ) : (
            <div>
              <h3 className="font-medium">驳回候选稿</h3>
              <p className="mt-1 text-xs text-muted-foreground">
                理由将绑定当前候选稿内容哈希并永久保存。驳回后该候选稿不能再应用，正文不会被删除或改写。
              </p>
              <label
                className="mt-3 block text-sm"
                htmlFor="candidate-reject-reason"
              >
                人工驳回理由
              </label>
              <textarea
                id="candidate-reject-reason"
                className="mt-1 min-h-24 w-full rounded-md border border-border bg-background p-3 text-sm"
                value={rejectReason}
                onChange={(event) => setRejectReason(event.target.value)}
                placeholder="至少 8 个字符，例如：教学实现直接导入 Gin，没有从零复现路由树机制。"
                maxLength={2000}
              />
              <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
                <span
                  className={`text-xs ${rejectReason.length > 0 && !rejectReasonIsValid ? "text-destructive" : "text-muted-foreground"}`}
                >
                  去除首尾空白后 {rejectReasonLength} / 2000 字符，至少需要 8
                  个字符。
                </span>
                <Button
                  type="button"
                  variant="outline"
                  disabled={isBusy || !lockIsMine || !rejectReasonIsValid}
                  onClick={() => void onReject()}
                >
                  驳回并保存理由
                </Button>
              </div>
            </div>
          )}
        </section>
 );
}
