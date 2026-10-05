import { expect, test } from "./fixtures/app";

const projectID = "11111111-1111-1111-1111-111111111111";
const chapterID = "22222222-2222-2222-2222-222222222222";
const candidateID = "33333333-3333-3333-3333-333333333333";
const approvedRevisionID = "44444444-4444-4444-4444-444444444444";
const manualReviewDimensions = [
  "句子与段落负担",
  "标题承诺",
  "重复",
  "语气",
  "例子相关性",
  "章节节奏",
  "图示机会",
  "练习梯度",
];

const envelope = (data: unknown) => ({ code: 0, data });

test("@core @textbook-sample keeps export unavailable until an explicit candidate approval", async ({
  appPage: page,
}) => {
  let approved = false;
  let locked = false;
  let lockOwnerID = "";
  let generationTasks = 0;

  const chapter = () => ({
    id: chapterID,
    project_id: projectID,
    sort_order: 1,
    title: "请求生命周期",
    chapter_profile: "concept",
    status: approved ? "approved" : "candidate",
    revision_version: approved ? 3 : 2,
    current_revision_id: approved ? approvedRevisionID : candidateID,
    ...(approved ? { approved_revision_id: approvedRevisionID } : {}),
    updated_at: "2026-09-03T00:00:00Z",
  });
  const workspace = () => ({
    project: {
      id: projectID,
      title: "Gin 从零自学教材",
      audience_level: "foundation",
      status: approved ? "approved" : "candidate",
      revision_version: 1,
      updated_at: "2026-09-03T00:00:00Z",
    },
    sources: [
      {
        id: "source-gin",
        project_id: projectID,
        kind: "git_repository",
        role: "primary",
        locator: "https://github.com/gin-gonic/gin",
        official_confirmed: false,
        license_status: "confirmed",
        created_at: "2026-09-03T00:00:00Z",
      },
    ],
    chapters: [chapter()],
  });
  const revisions = () => [
    ...(approved
      ? [
          {
            id: approvedRevisionID,
            chapter_id: chapterID,
            revision_number: 3,
            kind: "approved",
            markdown: "# 批准的 Gin 样章",
            document_json: {},
            content_hash: "c".repeat(64),
            created_by: "manual",
            parent_revision_id: candidateID,
            created_at: "2026-09-03T00:00:00Z",
          },
        ]
      : []),
    {
      id: candidateID,
      chapter_id: chapterID,
      revision_number: 2,
      kind: "candidate",
      markdown: "# Gin 路由登记\n\n候选教材稿",
      document_json: {},
      content_hash: "b".repeat(64),
      created_by: "generation",
      parent_revision_id: "draft-1",
      book_contract_revision_id: "book-contract-1",
      style_sheet_revision_id: "style-sheet-1",
      blueprint_revision_id: "blueprint-1",
      evidence_pack_hash: "sha256:evidence",
      prompt_hash: "sha256:prompt",
      provider_name: "fixture",
      model_name: "deterministic-gin-v1.12.0",
      quality_report_json: {
        contract_version: "inkwords.sample-quality.v9",
        passed: true,
        critical_claim_count: 3,
        covered_critical_claim_count: 3,
        manual_review_required: true,
        manual_review_dimensions: manualReviewDimensions,
        advisories: [],
      },
      created_at: "2026-09-03T00:00:00Z",
    },
    {
      id: "draft-1",
      chapter_id: chapterID,
      revision_number: 1,
      kind: "draft",
      markdown: "# 初稿",
      document_json: {},
      content_hash: "a".repeat(64),
      created_by: "manual",
      created_at: "2026-09-02T00:00:00Z",
    },
  ];

  // Register after the generic fixture route so this test owns the complete
  // textbook flow while the surrounding application keeps its normal stubs.
  await page.route("**/api/v1/textbook-projects**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (request.method() === "GET" && path === "/api/v1/textbook-projects")
      return route.fulfill({ json: envelope([workspace().project]) });
    if (
      request.method() === "GET" &&
      path === `/api/v1/textbook-projects/${projectID}/workspace`
    )
      return route.fulfill({ json: envelope(workspace()) });
    if (
      request.method() === "GET" &&
      path === `/api/v1/textbook-projects/${projectID}/source-library`
    )
      return route.fulfill({ json: envelope([]) });
    if (
      request.method() === "GET" &&
      path === `/api/v1/textbook-projects/${projectID}/source-evidence`
    )
      return route.fulfill({ json: envelope([]) });
    if (
      request.method() === "GET" &&
      path === `/api/v1/textbook-projects/${projectID}/progress`
    )
      return route.fulfill({
        json: envelope({
          stages: [
            {
              key: "sources",
              status: "approved",
              reason: "固定 Gin 资料已就绪。",
            },
            { key: "blueprint", status: "approved", reason: "蓝图已批准。" },
            {
              key: "sample",
              status: approved ? "approved" : "ready",
              reason: approved ? "样章已人工批准。" : "候选稿等待人工审阅。",
            },
            {
              key: "chapters",
              status: approved ? "approved" : "in_progress",
              reason: "样章审阅中。",
            },
            { key: "verification", status: "unavailable", reason: "未接入。" },
            { key: "learning", status: "unavailable", reason: "未接入。" },
            { key: "publication", status: "unavailable", reason: "未接入。" },
          ],
        }),
      });
    if (
      request.method() === "GET" &&
      path === `/api/v1/textbook-projects/chapters/${chapterID}/workspace`
    )
      return route.fulfill({
        json: envelope({
          chapter: chapter(),
          generation_target: {
            provider_name: "fixture",
            model_name: "deterministic-gin-v1.12.0",
          },
          human_review_contract: {
            contract_version: "inkwords.sample-human-review.v1",
            dimensions: manualReviewDimensions,
            minimum_score: 3,
            maximum_score: 4,
            note_min_runes: 8,
            note_max_runes: 2000,
          },
          revisions: revisions(),
          code_artifacts: [],
          runtime_evidence: [],
          assets: [],
          candidate_reviews: approved
            ? [
                {
                  id: "review-1",
                  project_id: projectID,
                  chapter_id: chapterID,
                  candidate_revision_id: candidateID,
                  reviewer_workspace_id: lockOwnerID,
                  decision: "approved",
                  reason: "已逐项核对技术事实、示例边界与练习梯度，同意应用。",
                  candidate_content_hash: "b".repeat(64),
                  quality_contract_version: "inkwords.sample-quality.v9",
                  human_review: {
                    contract_version: "inkwords.sample-human-review.v1",
                    dimension_scores: manualReviewDimensions.map(
                      (dimension) => ({ dimension, score: 3 }),
                    ),
                  },
                  created_at: "2026-09-03T00:00:00Z",
                },
              ]
            : [],
          ...(locked
            ? {
                lock: {
                  chapter_id: chapterID,
                  owner_id: lockOwnerID,
                  version: 1,
                  lease_expires_at: "2099-01-01T00:00:00Z",
                },
              }
            : {}),
        }),
      });
    if (
      request.method() === "GET" &&
      path ===
        `/api/v1/textbook-projects/${projectID}/chapters/${chapterID}/generation-preflight`
    )
      return route.fulfill({
        json: envelope({
          task_version: 3,
          prompt_schema_version: "inkwords.sample-prompt.v9",
          quality_contract_version: "inkwords.sample-quality.v9",
          generation_target: {
            provider_name: "fixture",
            model_name: "deterministic-gin-v1.12.0",
          },
          input_hash: `sha256:${"d".repeat(64)}`,
          evidence_reference_count: 6,
          estimated_input_tokens: 12800,
          allowed_input_tokens: 25000,
          reserved_output_tokens: 12000,
          within_budget: true,
          estimate_method: "conservative_utf8_payload_plus_prompt_reserve_v1",
          cache_status: "unknown_before_worker",
          estimated_cost_known: false,
          requires_confirmation: true,
        }),
      });
    if (
      request.method() === "POST" &&
      path ===
        `/api/v1/textbook-projects/${projectID}/chapters/${chapterID}/generate-sample`
    ) {
      generationTasks += 1;
      return route.fulfill({
        status: 202,
        json: envelope({ task_id: "task-1", status: "queued" }),
      });
    }
    if (
      request.method() === "POST" &&
      path === `/api/v1/textbook-projects/chapters/${chapterID}/lock`
    ) {
      locked = true;
      lockOwnerID = request.postDataJSON().owner_id;
      return route.fulfill({
        json: envelope({
          chapter_id: chapterID,
          owner_id: lockOwnerID,
          version: 1,
          lease_expires_at: "2099-01-01T00:00:00Z",
        }),
      });
    }
    if (
      request.method() === "POST" &&
      path ===
        `/api/v1/textbook-projects/chapters/${chapterID}/revisions/${candidateID}/apply`
    ) {
      const body = request.postDataJSON();
      expect(body.review_note).toBe(
        "已逐项核对技术事实、示例边界与练习梯度，同意应用。",
      );
      expect(body.dimension_scores).toEqual(
        manualReviewDimensions.map((dimension) => ({ dimension, score: 3 })),
      );
      approved = true;
      return route.fulfill({ json: envelope(revisions()[0]) });
    }
    return route.fulfill({
      status: 404,
      json: {
        code: 404,
        message: `Unhandled textbook route: ${request.method()} ${path}`,
      },
    });
  });

  await page.reload();
  await page.getByRole("button", { name: "打开工作台" }).click();
  await page.getByRole("button", { name: "编辑" }).click();

  await expect(
    page.getByRole("heading", { name: "请求生命周期" }),
  ).toBeVisible();
  await expect(page.getByText("关键事实证据覆盖：3 / 3")).toBeVisible();
  await expect(page.getByRole("link", { name: "Markdown" })).toHaveCount(0);
  await expect(
    page.getByText("批准候选稿后，才会显示可下载的 Markdown 和含谱系 ZIP。"),
  ).toBeVisible();

  await page.getByRole("button", { name: "生成候选教材稿" }).click();
  await expect(
    page.getByRole("heading", { name: "请确认本次样章生成" }),
  ).toBeVisible();
  await expect(page.getByText("约 12,800 / 25,000")).toBeVisible();
  expect(generationTasks).toBe(0);
  await page.getByRole("button", { name: "取消" }).click();
  expect(generationTasks).toBe(0);

  await page.getByRole("button", { name: "获取 15 分钟编辑锁" }).click();
  await expect(page.getByText(/编辑锁有效至/)).toBeVisible();
  for (const dimension of manualReviewDimensions) {
    await page.getByLabel(`${dimension}人工评分`).selectOption("3");
  }
  await page
    .getByRole("textbox", { name: "人工审阅说明" })
    .fill("已逐项核对技术事实、示例边界与练习梯度，同意应用。");
  await page.getByRole("button", { name: "我已人工审阅并批准" }).click();

  await expect(page.getByRole("link", { name: "Markdown" })).toHaveAttribute(
    "href",
    `/api/v1/textbook-projects/chapters/${chapterID}/export/markdown`,
  );
  await expect(page.getByRole("link", { name: "含谱系 ZIP" })).toHaveAttribute(
    "href",
    `/api/v1/textbook-projects/chapters/${chapterID}/export/zip`,
  );
  await expect(page.getByRole("status")).toContainText(
    "该候选稿已人工审阅并批准",
  );
});
