import { useEffect, useMemo, useState } from "react";
import {
  Download,
  FileDiff,
  LockKeyhole,
  Save,
  Sparkles,
  X,
} from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Panel, SectionHeader, StatusPill } from "@/components/ui/workspace";
import {
  getChapterEditorOwnerID,
  hashChapterMarkdown,
} from "@/lib/chapterEditorIdentity";
import type {
  ChapterRevision,
  ChapterWorkspace,
  SampleGenerationPreflight,
} from "@/services/textbook";
import { apiRoutes } from "@/services/apiRoutes";
import { RevisionVideoRunbooks } from "@/features/manuscript/RevisionVideoRunbooks";
import { VerificationPanel } from "@/features/verification/VerificationPanel";
import { DependencyProjectionPanel } from "@/features/verification/DependencyProjectionPanel";
import { VisualAssetUploadPanel } from "@/features/verification/VisualAssetUploadPanel";
import { CorrectionProvenancePanel } from "@/features/generation/CorrectionProvenancePanel";
import { correctionProvenance } from "@/features/generation/correctionProvenance";
import { GenerationTaskStatusPanel } from "@/features/generation/GenerationTaskStatusPanel";
import { GenerationPreflightPanel } from "@/features/generation/GenerationPreflightPanel";
import { CandidateDecisionPanel } from "./CandidateDecisionPanel";
import { CandidateApprovalReviewPanel } from "./CandidateApprovalReviewPanel";
import { fallbackSampleHumanReviewContract } from "./sampleHumanReview";

interface ChapterEditorPanelProps {
  workspace: ChapterWorkspace;
  isBusy: boolean;
  onAcquireLock: (ownerID: string) => Promise<void>;
  onSaveDraft: (input: {
    markdown: string;
    documentJson: Record<string, unknown>;
    contentHash: string;
    ownerId: string;
  }) => Promise<void>;
  onApplyCandidate: (input: {
    candidateRevisionId: string;
    ownerId: string;
    reviewNote: string;
    reviewerKind?: "human" | "delegated_ai";
    delegationNote?: string;
    dimensionScores: Array<{ dimension: string; score: number }>;
  }) => Promise<void>;
  onRejectCandidate: (input: {
    candidateRevisionId: string;
    ownerId: string;
    reason: string;
  }) => Promise<void>;
  onPrepareSampleGeneration?: () => Promise<SampleGenerationPreflight>;
  onGenerateSample: (confirmedInputHash: string) => Promise<void>;
  onUploadVisualAsset: (input: {
    revisionId: string;
    evidenceId: string;
    file: File;
    altText: string;
    source: string;
    generationMethod: string;
    visualPurpose:
      "layout" | "memory_map" | "call_stack" | "network_flow" | "rendered_ui";
    rightsStatus: "pending";
  }) => Promise<void>;
  onStartLearning?: () => Promise<void>;
  onDependencyRegistered?: () => Promise<void>;
  latestSampleTaskId: string | null;
  onClose: () => void;
}

const lineDiff = (before: string, after: string) => {
  const previous = before.split("\n");
  const next = after.split("\n");
  const length = Math.max(previous.length, next.length);
  return Array.from({ length }, (_, index) => ({
    index: index + 1,
    before: previous[index] ?? "",
    after: next[index] ?? "",
    changed: previous[index] !== next[index],
  }));
};

const threeWayLineDiff = (base: string, current: string, candidate: string) => {
  const baseLines = base.split("\n");
  const currentLines = current.split("\n");
  const candidateLines = candidate.split("\n");
  const length = Math.max(
    baseLines.length,
    currentLines.length,
    candidateLines.length,
  );
  return Array.from({ length }, (_, index) => ({
    index: index + 1,
    base: baseLines[index] ?? "",
    current: currentLines[index] ?? "",
    candidate: candidateLines[index] ?? "",
    changed:
      baseLines[index] !== currentLines[index] ||
      baseLines[index] !== candidateLines[index],
  }));
};

const revisionLabel = (revision: ChapterRevision) =>
  `${revision.kind === "candidate" ? "候选稿" : revision.kind === "approved" ? "批准稿" : "草稿"} r${revision.revision_number}`;

const shortReference = (value?: string) =>
  value ? `${value.slice(0, 8)}…${value.slice(-4)}` : "缺失";
const sampleQualityContractVersion = "inkwords.sample-quality.v10";

function ManualQualityReview({
  report,
}: {
  report?: ChapterRevision["quality_report_json"];
}) {
  if (!report?.manual_review_required) return null;
  const dimensions = report.manual_review_dimensions ?? [];
  const advisories = report.advisories ?? [];

  return (
    <section
      aria-label="人工质量审阅提示"
      className="rounded-md border border-amber-300/70 bg-amber-50/60 p-3 text-amber-950 dark:border-amber-800 dark:bg-amber-950/20 dark:text-amber-100"
    >
      <h4 className="font-medium">人工审阅清单</h4>
      <p className="mt-1 text-xs">
        以下自动软提示只定位值得复核的位置，不计分、不阻断应用，也不能替代你的技术判断和读者试学。
      </p>
      {dimensions.length ? (
        <ul aria-label="人工审阅维度" className="mt-2 flex flex-wrap gap-1.5">
          {dimensions.map((dimension) => (
            <li
              key={dimension}
              className="rounded-full border border-current/20 px-2 py-0.5"
            >
              {dimension}
            </li>
          ))}
        </ul>
      ) : null}
      {advisories.length ? (
        <ul className="mt-3 space-y-2">
          {advisories.map((item) => (
            <li
              key={`${item.dimension}:${item.code}`}
              className="rounded border border-current/15 bg-background/60 p-2"
            >
              <p className="font-medium">
                {item.dimension}：{item.message}
              </p>
              <p className="mt-1">定位：{item.evidence}</p>
              <p className="mt-1">请判断：{item.review_question}</p>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-2">本次未命中自动软提示；这不表示人工审阅已经通过。</p>
      )}
    </section>
  );
}

export function ChapterEditorPanel({
  workspace,
  isBusy,
  onAcquireLock,
  onSaveDraft,
  onApplyCandidate,
  onRejectCandidate,
  onPrepareSampleGeneration,
  onGenerateSample,
  onUploadVisualAsset,
  onStartLearning,
  onDependencyRegistered,
  latestSampleTaskId,
  onClose,
}: ChapterEditorPanelProps) {
  const ownerID = useMemo(() => getChapterEditorOwnerID(), []);
  // Older exported workspaces did not include these later-added projections.
  // Keep the review surface usable while the client refreshes to the current contract.
  const codeArtifacts = workspace.code_artifacts ?? [];
  const runtimeEvidence = workspace.runtime_evidence ?? [];
  const manuscriptAssets = workspace.assets ?? [];
  const candidateReviews = workspace.candidate_reviews ?? [];
  const currentRevision = workspace.revisions[0];
  const [markdown, setMarkdown] = useState(currentRevision?.markdown ?? "");
  const [candidateID, setCandidateID] = useState<string | null>(
    workspace.revisions.find((revision) => revision.kind === "candidate")?.id ??
      null,
  );
  const [rejectReason, setRejectReason] = useState("");
  const [editorError, setEditorError] = useState<string | null>(null);
  const [generationPreflight, setGenerationPreflight] =
    useState<SampleGenerationPreflight | null>(null);
  const [currentTime, setCurrentTime] = useState<number | null>(null);
  const candidate =
    workspace.revisions.find((revision) => revision.id === candidateID) ?? null;
  const candidateReview = candidate
    ? (candidateReviews.find(
        (review) => review.candidate_revision_id === candidate.id,
      ) ?? null)
    : null;
  const normalizedRejectReason = rejectReason.trim();
  const rejectReasonLength = Array.from(normalizedRejectReason).length;
  const rejectReasonIsValid =
    rejectReasonLength >= 8 && rejectReasonLength <= 2000;
  const approvedRevision = workspace.chapter.approved_revision_id
    ? (workspace.revisions.find(
        (revision) => revision.id === workspace.chapter.approved_revision_id,
      ) ?? null)
    : null;
  const candidateBaseRevision = candidate?.parent_revision_id
    ? (workspace.revisions.find(
        (revision) => revision.id === candidate.parent_revision_id,
      ) ?? null)
    : candidate && currentRevision?.id === candidate.id
      ? (workspace.revisions.find((revision) => revision.id !== candidate.id) ??
        null)
      : currentRevision;
  const candidateConflictsWithCurrent = Boolean(
    candidate &&
    candidateBaseRevision &&
    currentRevision &&
    currentRevision.id !== candidate.id &&
    currentRevision.id !== candidateBaseRevision.id,
  );
  const candidateQualityIsCurrent =
    candidate?.quality_report_json?.contract_version ===
      sampleQualityContractVersion &&
    candidate.quality_report_json.passed === true;
  const generationTarget = workspace.generation_target;
  const generationTargetLabel = generationTarget
    ? `${generationTarget.provider_name} / ${generationTarget.model_name}`
    : "服务端未声明";
  const candidateGenerationTargetIsCurrent = Boolean(
    candidate &&
    generationTarget &&
    candidate.provider_name === generationTarget.provider_name &&
    candidate.model_name === generationTarget.model_name,
  );
  const lockIsMine =
    workspace.lock?.owner_id === ownerID &&
    (currentTime === null ||
      new Date(workspace.lock.lease_expires_at).getTime() > currentTime);
  const isDirty = markdown !== (currentRevision?.markdown ?? "");

  useEffect(() => {
    const refreshCurrentTime = () => setCurrentTime(Date.now());
    const initialTimer = window.setTimeout(refreshCurrentTime, 0);
    const timer = window.setInterval(refreshCurrentTime, 30_000);
    return () => {
      window.clearTimeout(initialTimer);
      window.clearInterval(timer);
    };
  }, []);

  const saveDraft = async () => {
    setEditorError(null);
    try {
      const contentHash = await hashChapterMarkdown(markdown);
      await onSaveDraft({
        markdown,
        contentHash,
        ownerId: ownerID,
        documentJson: { format: "inkwords.chapter.v1", markdown },
      });
    } catch (error) {
      setEditorError(
        error instanceof Error ? error.message : "保存章节草稿失败",
      );
    }
  };

  const applyCandidate = async (review: {
    reviewNote: string;
    reviewerKind?: "human" | "delegated_ai";
    delegationNote?: string;
    dimensionScores: Array<{ dimension: string; score: number }>;
  }) => {
    if (!candidate) return;
    setEditorError(null);
    try {
      await onApplyCandidate({
        candidateRevisionId: candidate.id,
        ownerId: ownerID,
        ...review,
      });
    } catch (error) {
      setEditorError(error instanceof Error ? error.message : "应用候选稿失败");
    }
  };

  const rejectCandidate = async () => {
    if (!candidate || !rejectReasonIsValid) return;
    setEditorError(null);
    try {
      await onRejectCandidate({
        candidateRevisionId: candidate.id,
        ownerId: ownerID,
        reason: normalizedRejectReason,
      });
    } catch (error) {
      setEditorError(error instanceof Error ? error.message : "驳回候选稿失败");
    }
  };

  const prepareGeneration = async () => {
    if (!onPrepareSampleGeneration) return;
    setEditorError(null);
    try {
      setGenerationPreflight(await onPrepareSampleGeneration());
    } catch (error) {
      setEditorError(error instanceof Error ? error.message : "生成前预检失败");
    }
  };

  const confirmGeneration = async () => {
    if (!generationPreflight) return;
    setEditorError(null);
    try {
      await onGenerateSample(generationPreflight.input_hash);
      setGenerationPreflight(null);
    } catch (error) {
      setEditorError(
        error instanceof Error ? error.message : "创建样章生成任务失败",
      );
    }
  };

  return (
    <Panel className="mt-6 p-5" aria-label="章节编辑器">
      <SectionHeader
        eyebrow="章节编辑"
        title={workspace.chapter.title}
        description="每次保存都会创建新的不可变修订；批准稿不会被静默覆盖。"
        action={
          <Button variant="ghost" size="sm" className="gap-2" onClick={onClose}>
            <X className="h-4 w-4" />
            关闭
          </Button>
        }
      />
      <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <StatusPill tone={lockIsMine ? "success" : "warning"}>
          {lockIsMine
            ? `编辑锁有效至 ${new Date(workspace.lock!.lease_expires_at).toLocaleTimeString()}`
            : workspace.lock
              ? "章节由其他会话锁定或锁已过期"
              : "尚未获取编辑锁"}
        </StatusPill>
        <span>当前服务端版本 r{workspace.chapter.revision_version}</span>
        <span>{isDirty ? "有未保存修改" : "内容已与当前修订一致"}</span>
      </div>
      {editorError ? (
        <p role="alert" className="mt-4 text-sm text-destructive">
          {editorError}。编辑内容仍保留在当前页面，可修正后重试。
        </p>
      ) : null}
      {latestSampleTaskId ? (
        <GenerationTaskStatusPanel taskID={latestSampleTaskId} />
      ) : null}
      <div className="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_300px]">
        <div>
          <label className="text-sm font-medium" htmlFor="chapter-markdown">
            Markdown 教材原稿
          </label>
          <textarea
            id="chapter-markdown"
            disabled={!lockIsMine}
            className="mt-2 min-h-80 w-full rounded-md border border-border bg-background p-3 font-mono text-sm leading-6 disabled:cursor-not-allowed disabled:opacity-60"
            value={markdown}
            onChange={(event) => setMarkdown(event.target.value)}
            placeholder="# 为什么需要这项技术\n\n先从读者会遇到的真实问题讲起。"
          />
          <div className="mt-3 flex flex-wrap gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={isBusy || !onPrepareSampleGeneration}
              onClick={() => void prepareGeneration()}
              className="gap-2"
            >
              <Sparkles className="h-4 w-4" />
              生成候选教材稿
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={isBusy || lockIsMine}
              onClick={() =>
                void onAcquireLock(ownerID).catch((error) =>
                  setEditorError(
                    error instanceof Error ? error.message : "获取编辑锁失败",
                  ),
                )
              }
              className="gap-2"
            >
              <LockKeyhole className="h-4 w-4" />
              获取 15 分钟编辑锁
            </Button>
            <Button
              type="button"
              disabled={isBusy || !lockIsMine || !isDirty}
              onClick={() => void saveDraft()}
              className="gap-2"
            >
              <Save className="h-4 w-4" />
              保存为新草稿修订
            </Button>
          </div>
          {generationPreflight ? (
            <GenerationPreflightPanel
              preflight={generationPreflight}
              disabled={isBusy}
              onConfirm={() => void confirmGeneration()}
              onCancel={() => setGenerationPreflight(null)}
            />
          ) : null}
        </div>
        <aside className="rounded-md border border-border p-4">
          <h3 className="font-medium">修订历史</h3>
          <ol className="mt-3 space-y-2">
            {workspace.revisions.map((revision) => (
              <li
                key={revision.id}
                className="rounded border border-border px-3 py-2 text-xs"
              >
                <div className="flex items-center justify-between gap-2">
                  <span>{revisionLabel(revision)}</span>
                  <span>
                    {correctionProvenance(revision) ? "自动局部修订" : revision.created_by === "generation" ? "生成" : "人工"}
                  </span>
                </div>
                <p className="mt-1 text-muted-foreground">
                  {new Date(revision.created_at).toLocaleString()}
                </p>
              </li>
            ))}
          </ol>
          {workspace.revisions.length === 0 ? (
            <p className="mt-3 text-sm text-muted-foreground">
              还没有正文。获取编辑锁后即可保存第一份草稿。
            </p>
          ) : null}
          {workspace.chapter.approved_revision_id ? (
            <div className="mt-4 border-t border-border pt-4">
              <p className="text-xs text-muted-foreground">
                仅导出服务端已批准修订；草稿和候选稿不会进入文件。
              </p>
              <div className="mt-3 flex flex-wrap gap-2">
                <a
                  className={buttonVariants({ variant: "outline", size: "sm" })}
                  href={apiRoutes.exportService.textbookChapterMarkdown(
                    workspace.chapter.id,
                  )}
                >
                  <Download className="mr-1 h-3.5 w-3.5" />
                  Markdown
                </a>
                <a
                  className={buttonVariants({ variant: "outline", size: "sm" })}
                  href={apiRoutes.exportService.textbookChapterZip(
                    workspace.chapter.id,
                  )}
                >
                  <Download className="mr-1 h-3.5 w-3.5" />
                  含谱系 ZIP
                </a>
              </div>
            </div>
          ) : (
            <p className="mt-4 border-t border-border pt-4 text-xs text-muted-foreground">
              批准候选稿后，才会显示可下载的 Markdown 和含谱系 ZIP。
            </p>
          )}
        </aside>
      </div>
      {candidate ? (
        <section className="mt-6 border-t border-border pt-5">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="flex items-center gap-2 font-medium">
                <FileDiff className="h-4 w-4" />
                候选稿差异对照
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                候选稿由生成流程创建；应用前请逐行检查，与当前版本冲突时服务端会拒绝覆盖。
              </p>
            </div>
            <div className="flex items-center gap-2">
              <label className="sr-only" htmlFor="candidate-revision">
                选择候选稿
              </label>
              <select
                id="candidate-revision"
                className="rounded-md border border-border bg-background px-3 py-2 text-sm"
                value={candidateID ?? ""}
                onChange={(event) => {
                  setCandidateID(event.target.value);
                  setRejectReason("");
                }}
              >
                {workspace.revisions
                  .filter((revision) => revision.kind === "candidate")
                  .map((revision) => (
                    <option key={revision.id} value={revision.id}>
                      {revisionLabel(revision)}
                    </option>
                  ))}
              </select>
            </div>
          </div>
          <div className="mt-4 grid gap-3 rounded-md border border-border bg-muted/30 p-3 text-xs">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium">自动质量门禁</span>
              <StatusPill
                tone={candidateQualityIsCurrent ? "success" : "warning"}
              >
                {candidateQualityIsCurrent
                  ? "通过"
                  : candidate.quality_report_json?.passed === true
                    ? "规则已过期"
                    : candidate.quality_report_json
                      ? "未通过"
                      : "未提供报告"}
              </StatusPill>
            </div>
            {!candidateQualityIsCurrent &&
            candidate.quality_report_json?.passed === true ? (
              <p role="alert" className="text-amber-700 dark:text-amber-300">
                该候选稿使用旧版或缺失版本的质量合同，不能应用；请用当前蓝图和证据重新生成。
              </p>
            ) : null}
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium">生成目标</span>
              <StatusPill
                tone={
                  candidateGenerationTargetIsCurrent ? "success" : "warning"
                }
              >
                {candidateGenerationTargetIsCurrent ? "匹配" : "不匹配"}
              </StatusPill>
            </div>
            {!candidateGenerationTargetIsCurrent ? (
              <p role="alert" className="text-amber-700 dark:text-amber-300">
                该候选稿由 {candidate.provider_name || "未知 provider"} /{" "}
                {candidate.model_name || "未知模型"} 生成，当前任务目标为{" "}
                {generationTargetLabel}，不能应用；请按当前目标重新生成。
              </p>
            ) : null}
            {candidate.quality_report_json?.failures?.length ? (
              <ul className="list-disc space-y-1 pl-5 text-destructive">
                {candidate.quality_report_json.failures.map((failure) => (
                  <li key={failure}>{failure}</li>
                ))}
              </ul>
            ) : null}
            {candidate.quality_report_json?.critical_claim_count !==
              undefined &&
            candidate.quality_report_json.covered_critical_claim_count !==
              undefined ? (
              <p className="text-muted-foreground">
                关键事实证据覆盖：
                {
                  candidate.quality_report_json.covered_critical_claim_count
                } / {candidate.quality_report_json.critical_claim_count}
              </p>
            ) : null}
            <p className="text-muted-foreground">
              自动门禁只检查结构、证据引用、反自学措辞与运行状态等可机械验证项；它不替代你对技术事实、示例可运行性和读者是否真正理解的人工审阅。
            </p>
            <ManualQualityReview report={candidate.quality_report_json} />
            <dl className="grid gap-2 sm:grid-cols-2">
              <div>
                <dt className="text-muted-foreground">BookContract</dt>
                <dd className="font-mono">
                  {shortReference(candidate.book_contract_revision_id)}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">StyleSheet</dt>
                <dd className="font-mono">
                  {shortReference(candidate.style_sheet_revision_id)}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">教学蓝图</dt>
                <dd className="font-mono">
                  {shortReference(candidate.blueprint_revision_id)}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">证据包指纹</dt>
                <dd className="break-all font-mono">
                  {candidate.evidence_pack_hash || "缺失"}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{correctionProvenance(candidate) ? "修订输入指纹" : "提示词指纹"}</dt>
                <dd className="break-all font-mono">
                  {candidate.prompt_hash || "缺失"}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{correctionProvenance(candidate) ? "原始模型" : "生成器"}</dt>
                <dd>
                  {candidate.provider_name && candidate.model_name
                    ? `${candidate.provider_name} / ${candidate.model_name}`
                    : "缺失"}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">当前任务目标</dt>
                <dd>{generationTargetLabel}</dd>
              </div>
            </dl>
          </div>
          {candidateConflictsWithCurrent ? (
            <>
              <p
                role="alert"
                className="mt-4 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive"
              >
                当前稿已在候选稿的基线之后发生人工修改。请比较三方内容并创建新的候选稿；服务端也会拒绝直接应用此候选稿。
              </p>
              <div className="mt-4 grid grid-cols-[40px_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)] rounded-t-md border border-b-0 border-border bg-muted/40 px-2 py-2 font-mono text-xs text-muted-foreground">
                <span>行</span>
                <span>基线 r{candidateBaseRevision!.revision_number}</span>
                <span>当前 r{currentRevision!.revision_number}</span>
                <span>候选 r{candidate.revision_number}</span>
              </div>
              <div className="max-h-96 overflow-auto rounded-b-md border border-border font-mono text-xs">
                {threeWayLineDiff(
                  candidateBaseRevision!.markdown,
                  currentRevision!.markdown,
                  candidate.markdown,
                ).map((line) => (
                  <div
                    key={line.index}
                    className={`grid grid-cols-[40px_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)] border-b border-border/60 ${line.changed ? "bg-amber-50 dark:bg-amber-950/20" : ""}`}
                  >
                    <span className="px-2 py-1 text-muted-foreground">
                      {line.index}
                    </span>
                    <code className="min-w-0 whitespace-pre-wrap border-l border-border px-2 py-1">
                      {line.base || " "}
                    </code>
                    <code className="min-w-0 whitespace-pre-wrap border-l border-border px-2 py-1">
                      {line.current || " "}
                    </code>
                    <code className="min-w-0 whitespace-pre-wrap border-l border-border px-2 py-1">
                      {line.candidate || " "}
                    </code>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <>
              <div className="mt-4 grid grid-cols-[40px_minmax(0,1fr)_minmax(0,1fr)] rounded-t-md border border-b-0 border-border bg-muted/40 px-2 py-2 font-mono text-xs text-muted-foreground">
                <span>行</span>
                <span>
                  基线{" "}
                  {candidateBaseRevision
                    ? `r${candidateBaseRevision.revision_number}`
                    : "（无）"}
                </span>
                <span>候选 r{candidate.revision_number}</span>
              </div>
              <div className="max-h-96 overflow-auto rounded-b-md border border-border font-mono text-xs">
                {lineDiff(
                  candidateBaseRevision?.markdown ?? "",
                  candidate.markdown,
                ).map((line) => (
                  <div
                    key={line.index}
                    className={`grid grid-cols-[40px_minmax(0,1fr)_minmax(0,1fr)] border-b border-border/60 ${line.changed ? "bg-amber-50 dark:bg-amber-950/20" : ""}`}
                  >
                    <span className="px-2 py-1 text-muted-foreground">
                      {line.index}
                    </span>
                    <code className="min-w-0 whitespace-pre-wrap border-l border-border px-2 py-1">
                      {line.before || " "}
                    </code>
                    <code className="min-w-0 whitespace-pre-wrap border-l border-border px-2 py-1">
                      {line.after || " "}
                    </code>
                  </div>
                ))}
              </div>
            </>
          )}
        </section>
      ) : null}
      {candidate && !candidateReview ? (
        <CandidateApprovalReviewPanel
          key={candidate.id}
          contract={
            workspace.human_review_contract ?? fallbackSampleHumanReviewContract
          }
          disabled={
            isBusy ||
            !lockIsMine ||
            candidateConflictsWithCurrent ||
            !candidateQualityIsCurrent ||
            !candidateGenerationTargetIsCurrent
          }
          onApprove={applyCandidate}
        />
      ) : null}
      {candidate ? <CandidateDecisionPanel candidateReview={candidateReview} isBusy={isBusy} lockIsMine={lockIsMine} rejectReason={rejectReason} setRejectReason={setRejectReason} rejectReasonIsValid={rejectReasonIsValid} rejectReasonLength={rejectReasonLength} onReject={rejectCandidate} /> : null}
      {candidate ? <CorrectionProvenancePanel revision={candidate} /> : null}
      <RevisionVideoRunbooks candidate={candidate} approved={approvedRevision} />
      {approvedRevision ? (
        <>
          {onStartLearning ? (
            <div className="mt-4">
              <Button
                type="button"
                variant="outline"
                disabled={isBusy}
                onClick={() =>
                  void onStartLearning().catch((error) =>
                    setEditorError(
                      error instanceof Error
                        ? error.message
                        : "创建学习目标失败",
                    ),
                  )
                }
              >
                从批准教材创建学习任务
              </Button>
              <p className="mt-2 text-xs text-muted-foreground">
                任务只从当前批准修订的学习投影创建；重复操作会复用同一冻结修订目标。
              </p>
            </div>
          ) : null}
        </>
      ) : null}
      {workspace.chapter.chapter_profile === 'hands_on' ? <DependencyProjectionPanel key={workspace.chapter.id} chapterId={workspace.chapter.id} revisions={workspace.revisions} disabled={isBusy} onRegistered={onDependencyRegistered} /> : null}
      <VerificationPanel
        key={[...codeArtifacts.map(item => `${item.id}:${item.status}`), ...runtimeEvidence.map(item => item.id), ...manuscriptAssets.map(item => item.id)].join('|')}
        chapterId={workspace.chapter.id}
        revisionNumbers={Object.fromEntries(workspace.revisions.map(item => [item.id, item.revision_number]))}
        currentSourceRevisionId={currentRevision?.kind === 'approved' ? currentRevision.parent_revision_id : currentRevision?.id}
        artifacts={codeArtifacts}
        evidence={runtimeEvidence}
        assets={manuscriptAssets}
      />
      <VisualAssetUploadPanel
        revisions={workspace.revisions}
        evidence={runtimeEvidence}
        disabled={isBusy}
        onUpload={onUploadVisualAsset}
      />
    </Panel>
  );
}
