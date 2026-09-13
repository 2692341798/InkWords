import { useMemo, useState } from "react";
import { CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { SampleHumanReviewContract } from "@/services/textbook";

interface CandidateApprovalReviewPanelProps {
  disabled: boolean;
  contract: SampleHumanReviewContract;
  onApprove: (input: {
    reviewNote: string;
    reviewerKind?: "human" | "delegated_ai";
    delegationNote?: string;
    dimensionScores: Array<{ dimension: string; score: number }>;
  }) => Promise<void>;
}

export function CandidateApprovalReviewPanel({
  disabled,
  contract,
  onApprove,
}: CandidateApprovalReviewPanelProps) {
  const [scores, setScores] = useState<Record<string, number>>({});
  const [reviewNote, setReviewNote] = useState("");
  const [reviewerKind, setReviewerKind] = useState<"human" | "delegated_ai">("human");
  const [delegationNote, setDelegationNote] = useState("");
  const delegated = reviewerKind === "delegated_ai";
  const normalizedNote = reviewNote.trim();
  const noteLength = Array.from(normalizedNote).length;
  const dimensionScores = useMemo(
    () =>
      contract.dimensions.map((dimension) => ({
        dimension,
        score: scores[dimension] ?? 0,
      })),
    [contract.dimensions, scores],
  );
  const allPassing = dimensionScores.every(
    ({ score }) => score >= contract.minimum_score,
  );
  const canApprove =
    !disabled &&
    (!delegated || (contract.reviewer_kinds?.includes("delegated_ai") && Array.from(delegationNote.trim()).length >= 8 && Array.from(delegationNote.trim()).length <= 1000)) &&
    allPassing &&
    noteLength >= contract.note_min_runes &&
    noteLength <= contract.note_max_runes;

  return (
    <section
      aria-label={delegated ? "候选稿委托 AI 批准量表" : "候选稿人工批准量表"}
      className="mt-4 rounded-md border border-border p-4"
    >
      <h3 className="font-medium">{delegated ? "用户授权 AI 审阅并批准" : "人工审阅并批准"}</h3>
      <p className="mt-1 text-xs text-muted-foreground">
        {`${delegated ? "记录用户授权的 AI 决定，不属于真人试学、人工同行评审或出版社认证。" : "这是你的人工结论，不是自动门禁的重复。"}${contract.dimensions.length} 个维度必须逐项至少 ${contract.minimum_score}/${contract.maximum_score}，并留下可追溯的审阅说明。`}
      </p>
      {contract.reviewer_kinds?.includes("delegated_ai") && (
        <label className="mt-3 grid gap-1 text-sm">
          审阅来源
          <select aria-label="审阅来源" value={reviewerKind}
            onChange={event => setReviewerKind(event.target.value as "human" | "delegated_ai")}
            className="rounded-md border border-border bg-background p-2">
            <option value="human">本人审阅</option>
            <option value="delegated_ai">用户授权 AI 审阅</option>
          </select>
        </label>
      )}
      {delegated && (
        <label className="mt-3 grid gap-1 text-sm">
          授权说明
          <textarea aria-label="授权说明" value={delegationNote}
            onChange={event => setDelegationNote(event.target.value)} maxLength={1000}
            placeholder="记录用户的明确委托、审阅执行者与范围，至少 8 个字。"
            className="rounded-md border border-border bg-background p-2" />
        </label>
      )}
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        {contract.dimensions.map((dimension) => (
          <label key={dimension} className="grid gap-1 text-sm">
            <span>{dimension}</span>
            <select
              aria-label={`${dimension}${delegated ? "委托 AI" : "人工"}评分`}
              className="rounded-md border border-border bg-background px-3 py-2"
              value={scores[dimension] ?? 0}
              onChange={(event) =>
                setScores((current) => ({
                  ...current,
                  [dimension]: Number(event.target.value),
                }))
              }
            >
              <option value={0}>未评分</option>
              {Array.from(
                { length: contract.maximum_score },
                (_, index) => index + 1,
              ).map((score) => (
                <option key={score} value={score}>
                  {score} / {contract.maximum_score}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>
      <label className="mt-3 block text-sm" htmlFor="candidate-approval-note">
        {delegated ? "委托 AI 审阅说明" : "人工审阅说明"}
      </label>
      <textarea
        id="candidate-approval-note"
        className="mt-1 min-h-24 w-full rounded-md border border-border bg-background p-3 text-sm"
        value={reviewNote}
        onChange={(event) => setReviewNote(event.target.value)}
        placeholder={`至少 ${contract.note_min_runes} 个字，记录你核对过的内容、仍接受的边界与批准理由。`}
        maxLength={contract.note_max_runes}
      />
      <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
        <span className="text-xs text-muted-foreground">
          {allPassing
            ? `${contract.dimensions.length} 维评分已达到门槛`
            : `请完成 ${contract.dimensions.length} 维评分，每项至少 ${contract.minimum_score}/${contract.maximum_score}`}
          ；说明 {noteLength} / {contract.note_max_runes} 字符。
        </span>
        <Button
          type="button"
          disabled={!canApprove}
          onClick={() =>
            void onApprove({ reviewNote: normalizedNote, dimensionScores, ...(delegated ? { reviewerKind, delegationNote: delegationNote.trim() } : {}) })
          }
          className="gap-2"
        >
          <CheckCircle2 className="h-4 w-4" />
          {delegated ? "按用户授权批准（AI 审阅）" : "我已人工审阅并批准"}
        </Button>
      </div>
    </section>
  );
}
