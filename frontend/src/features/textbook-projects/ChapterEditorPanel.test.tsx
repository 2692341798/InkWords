// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ChapterEditorPanel } from "./ChapterEditorPanel";
import { sampleManualReviewDimensions } from "./sampleHumanReview";

const videoRunbook = {
  format: "inkwords.video-runbook.v1",
  stack: "go",
  observation_goal: "调用链",
  recommendation: {
    primary: "GoLand",
    alternative: "VS Code + Go 扩展",
    reason: "GoLand 适合展示调试变量。",
    manual_capture_required: true,
  },
  steps: [
    {
      tool: "GoLand",
      tool_version: "2024.1 或更新版本",
      start_state: "已关闭旧项目。",
      action: "打开教材示例目录。",
      shortcut_or_menu: "File → Open",
      input: "选择示例目录",
      expected_view: "项目树可见。",
      narration: "固定示例。",
      capture_point: "项目树",
      recovery: "重新选择示例目录。",
      completion_signal: "目标文件已打开。",
    },
  ],
  capture_checklist: ["记录实际工具名称、完整版本和操作系统。"],
  manual_capture_pending: true,
  verification_status: "unverified" as const,
};

const workspace = {
  chapter: {
    id: "chapter-1",
    project_id: "project-1",
    sort_order: 1,
    title: "请求生命周期",
    chapter_profile: "concept" as const,
    status: "candidate",
    revision_version: 2,
    updated_at: "2026-09-02T00:00:00Z",
  },
  generation_target: {
    provider_name: "fixture",
    model_name: "deterministic-gin-v1.12.0",
  },
  human_review_contract: {
    contract_version: "inkwords.sample-human-review.v1",
    dimensions: [...sampleManualReviewDimensions],
    minimum_score: 3,
    maximum_score: 4,
    note_min_runes: 8,
    note_max_runes: 2000,
  },
  revisions: [
    {
      id: "candidate-1",
      chapter_id: "chapter-1",
      revision_number: 2,
      kind: "candidate" as const,
      markdown: "# 新稿\n候选解释",
      document_json: { video_runbook: videoRunbook },
      content_hash: "b".repeat(64),
      created_by: "generation" as const,
      parent_revision_id: "draft-1",
      book_contract_revision_id: "book-contract-1234",
      style_sheet_revision_id: "style-sheet-1234",
      blueprint_revision_id: "blueprint-1234",
      evidence_pack_hash: "sha256:evidence",
      prompt_hash: "sha256:prompt",
      provider_name: "fixture",
      model_name: "deterministic-gin-v1.12.0",
      provider_usage_json: {
        known: false,
        provider_call_count: 0,
        local_cache_hit: false,
        estimated_cost_known: false,
      },
      quality_report_json: {
        contract_version: "inkwords.sample-quality.v10",
        passed: true,
        critical_claim_count: 3,
        covered_critical_claim_count: 3,
      },
      created_at: "2026-09-02T00:00:00Z",
    },
    {
      id: "draft-1",
      chapter_id: "chapter-1",
      revision_number: 1,
      kind: "draft" as const,
      markdown: "# 旧稿\n原始解释",
      document_json: {},
      content_hash: "a".repeat(64),
      created_by: "manual" as const,
      created_at: "2026-09-01T00:00:00Z",
    },
  ],
  code_artifacts: [],
  runtime_evidence: [],
  assets: [],
  candidate_reviews: [],
};

const approvedWorkspace = {
  ...workspace,
  chapter: { ...workspace.chapter, approved_revision_id: "approved-1" },
  revisions: [
    ...workspace.revisions,
    {
      id: "approved-1",
      chapter_id: "chapter-1",
      revision_number: 0,
      kind: "approved" as const,
      markdown: "# 批准稿\n已经人工审阅",
      document_json: {
        video_runbook: {
          ...videoRunbook,
          steps: [
            {
              ...videoRunbook.steps[0],
              action: "从批准稿打开教材示例目录。",
              shortcut_or_menu: "批准稿菜单",
            },
          ],
        },
      },
      content_hash: "c".repeat(64),
      created_by: "manual" as const,
      created_at: "2026-09-01T00:00:00Z",
    },
  ],
};

const generationPreflight = {
  task_version: 3,
  prompt_schema_version: "inkwords.sample-prompt.v9",
  quality_contract_version: "inkwords.sample-quality.v10",
  generation_target: workspace.generation_target,
  input_hash: `sha256:${"d".repeat(64)}`,
  evidence_reference_count: 6,
  estimated_input_tokens: 12800,
  allowed_input_tokens: 25000,
  reserved_output_tokens: 12000,
  within_budget: true,
  estimate_method: "conservative_utf8_payload_plus_prompt_reserve_v1",
  cache_status: "unknown_before_worker" as const,
  estimated_cost_known: false as const,
  requires_confirmation: true as const,
};

describe("ChapterEditorPanel", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("crypto", {
      randomUUID: () => "00000000-0000-4000-8000-000000000001",
      subtle: globalThis.crypto.subtle,
    });
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("shows immutable history and requires a short lease before editing", async () => {
    const onAcquireLock = vi.fn().mockResolvedValue(undefined);
    const onPrepareSampleGeneration = vi
      .fn()
      .mockResolvedValue(generationPreflight);
    const onGenerateSample = vi.fn().mockResolvedValue(undefined);
    const onStartLearning = vi.fn().mockResolvedValue(undefined);
    render(
      <ChapterEditorPanel
        workspace={approvedWorkspace}
        isBusy={false}
        onAcquireLock={onAcquireLock}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onPrepareSampleGeneration={onPrepareSampleGeneration}
        onGenerateSample={onGenerateSample}
        onUploadVisualAsset={vi.fn()}
        onStartLearning={onStartLearning}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getAllByText("候选稿 r2")).toHaveLength(2);
    expect(screen.getByText("草稿 r1")).toBeTruthy();
    expect(screen.getByText("候选稿差异对照")).toBeTruthy();
    expect(screen.getByText("自动质量门禁")).toBeTruthy();
    expect(screen.getByText("通过")).toBeTruthy();
    expect(screen.getByText("关键事实证据覆盖：3 / 3")).toBeTruthy();
    expect(screen.getByText("证据包指纹")).toBeTruthy();
    expect(
      screen.getAllByText("fixture / deterministic-gin-v1.12.0"),
    ).toHaveLength(2);
    expect(screen.getByText("匹配")).toBeTruthy();
    expect(screen.getByLabelText("生成使用量").textContent).toContain(
      "供应商未返回 Token 使用量",
    );
    const candidateRunbook = within(screen.getByRole("region", { name: "所选候选稿 r2 视频教案" }));
    const approvedRunbook = within(screen.getByRole("region", { name: "批准稿 r0 视频教案" }));
    expect(candidateRunbook.getByText("File → Open")).toBeTruthy();
    expect(candidateRunbook.queryByText("批准稿菜单")).toBeNull();
    expect(approvedRunbook.getByText("批准稿菜单")).toBeTruthy();
    expect(approvedRunbook.queryByText("File → Open")).toBeNull();
    expect(candidateRunbook.getByText("待人工采集证据")).toBeTruthy();
    expect(
      (screen.getByLabelText("Markdown 教材原稿") as HTMLTextAreaElement)
        .disabled,
    ).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "生成候选教材稿" }));
    await waitFor(() =>
      expect(onPrepareSampleGeneration).toHaveBeenCalledOnce(),
    );
    expect(onGenerateSample).not.toHaveBeenCalled();
    expect(screen.getByLabelText("样章生成前预检").textContent).toContain(
      "约 12,800 / 25,000",
    );
    expect(screen.getByLabelText("样章生成前预检").textContent).toContain(
      "工作器执行前未知",
    );
    fireEvent.click(screen.getByRole("button", { name: "确认并创建任务" }));
    await waitFor(() =>
      expect(onGenerateSample).toHaveBeenCalledWith(
        generationPreflight.input_hash,
      ),
    );
    fireEvent.click(screen.getByRole("button", { name: "获取 15 分钟编辑锁" }));

    await waitFor(() =>
      expect(onAcquireLock).toHaveBeenCalledWith(
        "00000000-0000-4000-8000-000000000001",
      ),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "从批准教材创建学习任务" }),
    );
    await waitFor(() => expect(onStartLearning).toHaveBeenCalledOnce());
  });

  it("only exposes downloads once the server has approved a revision", () => {
    const approvedWorkspace = {
      ...workspace,
      chapter: { ...workspace.chapter, approved_revision_id: "approved-1" },
      revisions: [
        {
          ...workspace.revisions[0],
          id: "approved-1",
          kind: "approved" as const,
        },
        ...workspace.revisions.slice(1),
      ],
    };
    const { rerender } = render(
      <ChapterEditorPanel
        workspace={workspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );
    expect(screen.queryByRole("link", { name: "Markdown" })).toBeNull();
    rerender(
      <ChapterEditorPanel
        workspace={approvedWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );
    expect(
      (
        screen.getByRole("link", { name: "Markdown" }) as HTMLAnchorElement
      ).getAttribute("href"),
    ).toBe("/api/v1/textbook-projects/chapters/chapter-1/export/markdown");
    expect(
      (
        screen.getByRole("link", { name: "含谱系 ZIP" }) as HTMLAnchorElement
      ).getAttribute("href"),
    ).toBe("/api/v1/textbook-projects/chapters/chapter-1/export/zip");
  });

  it("keeps a candidate review usable when an older workspace omits verification projections", () => {
    const olderWorkspace = {
      chapter: workspace.chapter,
      revisions: workspace.revisions,
    };

    render(
      <ChapterEditorPanel
        workspace={olderWorkspace as typeof workspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("候选稿差异对照")).toBeTruthy();
    expect(screen.getByText("关键事实证据覆盖：3 / 3")).toBeTruthy();
  });

  it("shows bounded soft-review evidence without blocking a current candidate", async () => {
    const onApplyCandidate = vi.fn().mockResolvedValue(undefined);
    const advisoryWorkspace = {
      ...workspace,
      lock: {
        chapter_id: "chapter-1",
        owner_id: "00000000-0000-4000-8000-000000000001",
        version: 1,
        lease_expires_at: "2099-01-01T00:00:00Z",
      },
      revisions: workspace.revisions.map((revision) =>
        revision.kind === "candidate"
          ? {
              ...revision,
              quality_report_json: {
                ...revision.quality_report_json,
                manual_review_required: true,
                manual_review_dimensions: [
                  "句子与段落负担",
                  "标题承诺",
                  "重复",
                  "语气",
                  "例子相关性",
                  "章节节奏",
                  "图示机会",
                  "练习梯度",
                ],
                advisories: [
                  {
                    dimension: "图示机会",
                    code: "flow_visual_opportunity",
                    message: "本节包含多步调用或数据流，但没有配套流程图。",
                    evidence: "先调用，再查找，最后返回。",
                    review_question: "流程图是否能降低定位负担？",
                  },
                ],
              },
            }
          : revision,
      ),
    };

    render(
      <ChapterEditorPanel
        workspace={advisoryWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={onApplyCandidate}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("人工质量审阅提示").textContent).toContain(
      "不计分、不阻断应用",
    );
    expect(screen.getByLabelText("人工审阅维度").textContent).toContain(
      "练习梯度",
    );
    expect(
      screen.getByText(
        "图示机会：本节包含多步调用或数据流，但没有配套流程图。",
      ),
    ).toBeTruthy();
    const approveButton = screen.getByRole("button", {
      name: "我已人工审阅并批准",
    }) as HTMLButtonElement;
    expect(approveButton.disabled).toBe(true);
    for (const dimension of sampleManualReviewDimensions) {
      fireEvent.change(screen.getByLabelText(`${dimension}人工评分`), {
        target: { value: "3" },
      });
    }
    fireEvent.change(screen.getByRole("textbox", { name: "人工审阅说明" }), {
      target: { value: "已逐项核对技术事实、示例边界与练习梯度，同意应用。" },
    });
    expect(approveButton.disabled).toBe(false);
    fireEvent.click(approveButton);
    await waitFor(() =>
      expect(onApplyCandidate).toHaveBeenCalledWith({
        candidateRevisionId: "candidate-1",
        ownerId: "00000000-0000-4000-8000-000000000001",
        reviewNote: "已逐项核对技术事实、示例边界与练习梯度，同意应用。",
        dimensionScores: sampleManualReviewDimensions.map((dimension) => ({
          dimension,
          score: 3,
        })),
      }),
    );
  });

  it("uses the server-owned human review threshold instead of a client-only rule", () => {
    const lockedWorkspace = {
      ...workspace,
      human_review_contract: {
        ...workspace.human_review_contract,
        minimum_score: 4,
      },
      lock: {
        chapter_id: "chapter-1",
        owner_id: "00000000-0000-4000-8000-000000000001",
        version: 1,
        lease_expires_at: "2099-01-01T00:00:00Z",
      },
    };

    render(
      <ChapterEditorPanel
        workspace={lockedWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    const approveButton = screen.getByRole("button", {
      name: "我已人工审阅并批准",
    }) as HTMLButtonElement;
    for (const dimension of sampleManualReviewDimensions) {
      fireEvent.change(screen.getByLabelText(`${dimension}人工评分`), {
        target: { value: "3" },
      });
    }
    fireEvent.change(screen.getByRole("textbox", { name: "人工审阅说明" }), {
      target: { value: "已经逐项核对并接受当前候选稿边界。" },
    });
    expect(approveButton.disabled).toBe(true);
    expect(screen.getByLabelText("候选稿人工批准量表").textContent).toContain(
      "每项至少 4/4",
    );
    for (const dimension of sampleManualReviewDimensions) {
      fireEvent.change(screen.getByLabelText(`${dimension}人工评分`), {
        target: { value: "4" },
      });
    }
    expect(approveButton.disabled).toBe(false);
  });

  it("requires a current lock and a bounded human reason before rejecting a candidate", async () => {
    const onRejectCandidate = vi.fn().mockResolvedValue(undefined);
    const lockedWorkspace = {
      ...workspace,
      lock: {
        chapter_id: "chapter-1",
        owner_id: "00000000-0000-4000-8000-000000000001",
        version: 1,
        lease_expires_at: "2099-01-01T00:00:00Z",
      },
    };

    render(
      <ChapterEditorPanel
        workspace={lockedWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={onRejectCandidate}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    const reason = screen.getByRole("textbox", { name: "人工驳回理由" });
    const rejectButton = screen.getByRole("button", {
      name: "驳回并保存理由",
    }) as HTMLButtonElement;
    expect(rejectButton.disabled).toBe(true);
    fireEvent.change(reason, { target: { value: "太短" } });
    expect(rejectButton.disabled).toBe(true);
    fireEvent.change(reason, {
      target: { value: "  教学实现直接导入 Gin，没有复现路由树机制。  " },
    });
    expect(rejectButton.disabled).toBe(false);
    fireEvent.click(rejectButton);

    await waitFor(() =>
      expect(onRejectCandidate).toHaveBeenCalledWith({
        candidateRevisionId: "candidate-1",
        ownerId: "00000000-0000-4000-8000-000000000001",
        reason: "教学实现直接导入 Gin，没有复现路由树机制。",
      }),
    );
  });

  it("renders an immutable rejection record and prevents later application", () => {
    const rejectedWorkspace = {
      ...workspace,
      lock: {
        chapter_id: "chapter-1",
        owner_id: "00000000-0000-4000-8000-000000000001",
        version: 1,
        lease_expires_at: "2099-01-01T00:00:00Z",
      },
      candidate_reviews: [
        {
          id: "review-1",
          project_id: "project-1",
          chapter_id: "chapter-1",
          candidate_revision_id: "candidate-1",
          reviewer_workspace_id: "00000000-0000-4000-8000-000000000001",
          decision: "rejected" as const,
          reason: "教学实现直接导入 Gin，没有复现路由树机制。",
          candidate_content_hash: "b".repeat(64),
          quality_contract_version: "inkwords.sample-quality.v7",
          created_at: "2026-09-04T00:00:00Z",
        },
      ],
    };

    render(
      <ChapterEditorPanel
        workspace={rejectedWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByRole("status").textContent).toContain("该候选稿已驳回");
    expect(screen.getByRole("status").textContent).toContain(
      "教学实现直接导入 Gin",
    );
    expect(screen.getByRole("status").textContent).toContain(
      "inkwords.sample-quality.v7",
    );
    expect(screen.queryByRole("textbox", { name: "人工驳回理由" })).toBeNull();
    expect(
      screen.queryByRole("button", { name: "我已人工审阅并批准" }),
    ).toBeNull();
  });

  it.each([undefined, "inkwords.sample-quality.v9"])("blocks a passing report using stale contract %s", (contractVersion) => {
    const staleQualityWorkspace = {
      ...workspace,
      revisions: workspace.revisions.map((revision) =>
        revision.kind === "candidate"
          ? {
              ...revision,
              quality_report_json: {
                contract_version: contractVersion,
                passed: true,
                critical_claim_count: 3,
                covered_critical_claim_count: 3,
              },
            }
          : revision,
      ),
    };

    render(
      <ChapterEditorPanel
        workspace={staleQualityWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("规则已过期")).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain(
      "旧版或缺失版本的质量合同",
    );
    expect(
      (
        screen.getByRole("button", {
          name: "我已人工审阅并批准",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
  });

  it("blocks a passing candidate when its frozen provider and model no longer match the current generation target", () => {
    const wrongTargetWorkspace = {
      ...workspace,
      generation_target: {
        provider_name: "deepseek",
        model_name: "deepseek-v4-flash",
      },
      lock: {
        chapter_id: "chapter-1",
        owner_id: "00000000-0000-4000-8000-000000000001",
        version: 1,
        lease_expires_at: "2099-01-01T00:00:00Z",
      },
    };

    render(
      <ChapterEditorPanel
        workspace={wrongTargetWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("不匹配")).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain(
      "当前任务目标为 deepseek / deepseek-v4-flash",
    );
    expect(
      (
        screen.getByRole("button", {
          name: "我已人工审阅并批准",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
  });

  it("shows a three-way review and disables applying a candidate when manual work moved past its baseline", () => {
    const staleWorkspace = {
      ...workspace,
      chapter: { ...workspace.chapter, status: "draft", revision_version: 3 },
      revisions: [
        {
          ...workspace.revisions[1],
          id: "draft-2",
          revision_number: 3,
          markdown: "# 当前稿\n人工补充解释",
          created_at: "2026-09-03T00:00:00Z",
        },
        ...workspace.revisions,
      ],
    };
    render(
      <ChapterEditorPanel
        workspace={staleWorkspace}
        isBusy={false}
        onAcquireLock={vi.fn()}
        onSaveDraft={vi.fn()}
        onApplyCandidate={vi.fn()}
        onRejectCandidate={vi.fn()}
        onGenerateSample={vi.fn()}
        onUploadVisualAsset={vi.fn()}
        latestSampleTaskId={null}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert").textContent).toContain(
      "当前稿已在候选稿的基线之后发生人工修改",
    );
    expect(screen.getByText("当前 r3")).toBeTruthy();
    expect(screen.getByText("候选 r2")).toBeTruthy();
    expect(
      (
        screen.getByRole("button", {
          name: "我已人工审阅并批准",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
  });
});
