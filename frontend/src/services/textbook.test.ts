import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestJson } = vi.hoisted(() => ({ requestJson: vi.fn() }));

vi.mock("./apiClient", () => ({ requestJson }));

import { textbookService } from "./textbook";

describe("textbookService", () => {
  it('keeps verification lookup read-only and sends only artifact identity for execution', async () => {
    await textbookService.getArtifactVerification('chapter 1', 'artifact/2');
    expect(requestJson).toHaveBeenLastCalledWith('/api/v1/textbook-projects/chapters/chapter%201/code-artifacts/artifact%2F2/verify', { fallbackMessage: '查询代码验证任务失败' });
    await textbookService.startArtifactVerification('chapter 1', 'artifact/2');
    expect(requestJson).toHaveBeenLastCalledWith('/api/v1/textbook-projects/chapters/chapter%201/code-artifacts/artifact%2F2/verify', { method: 'POST', json: {}, fallbackMessage: '启动隔离验证失败' });
  });
  it("encodes exact selected evidence identifiers without turning them into a search", async () => {
    await textbookService.getSourceEvidence("project", ["chunk+a", "chunk&b"]);
    expect(requestJson).toHaveBeenCalledWith("/api/v1/textbook-projects/project/source-evidence?chunk_id=chunk%2Ba&chunk_id=chunk%26b", expect.anything());
  });
  beforeEach(() => {
    requestJson.mockReset();
    requestJson.mockResolvedValue({ code: 0, data: {} });
  });

  it("binds project, source, contract and blueprint actions to textbook routes", async () => {
    await textbookService.testConfiguredProviderConnection();
    await textbookService.list();
    await textbookService.get("project 1");
    await textbookService.getWorkspace("project 1");
    await textbookService.getProgress("project 1");
    await textbookService.getSourceLibrary("project 1");
    await textbookService.getSourceEvidence("project 1");
    await textbookService.retrieveSourceEvidence("project 1", {
      query: "路由 匹配",
      limit: 8,
    });
    await textbookService.create({
      title: "Gin 教材",
      audience_level: "foundation",
      primary_source: {
        kind: "git_repository",
        locator: "https://github.com/gin-gonic/gin",
      },
    });
    await textbookService.addSource("project 1", {
      kind: "official_web",
      role: "official_supporting",
      locator: "https://go.dev/",
      official_confirmed: true,
    });
    await textbookService.loadGinFixture("project 1");
    await textbookService.createChapter("project 1", {
      sort_order: 1,
      title: "路由",
      chapter_profile: "concept",
    });
    await textbookService.createBookContract("project 1", {
      reader: {
        audience: "foundation",
        known_knowledge: [],
        forbidden_assumptions: [],
        learning_outcomes: ["解释路由"],
      },
      promise: "从问题开始",
      chapter_profiles: ["concept"],
      terminology_version: "v1",
      publication_profile: "personal_learning",
    });
    await textbookService.createStyleSheet("project 1", {
      language: "zh-CN",
      terminology_rules: ["先白话"],
      code_rules: ["标记证据"],
      visual_rules: [],
      citation_rules: ["引用来源"],
      forbidden_phrases: [],
    });
    await textbookService.approveBookContract("project 1", "book 1");
    await textbookService.approveStyleSheet("project 1", "style 1");
    await textbookService.createBlueprint("project 1", {
      volumes: [
        {
          id: "volume-1",
          title: "基础",
          sort: 1,
          chapters: [
            {
              id: "chapter-1",
              title: "路由",
              sort: 1,
              profile: "concept",
              evidence_ids: ["evidence-1"],
              critical_claims: [
                {
                  id: "claim-1",
                  label: "路由登记",
                  evidence_ids: ["evidence-1"],
                },
              ],
            },
          ],
        },
      ],
    });
    await textbookService.approveBlueprint("project 1", "blueprint 1");
    await textbookService.createBookBuild("project 1");

    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/stream/provider-connection-test",
      expect.objectContaining({ method: "POST", json: {} }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/source-retrieval",
      expect.objectContaining({
        method: "POST",
        json: { query: "路由 匹配", limit: 8 },
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects",
      expect.objectContaining({
        json: expect.objectContaining({ title: "Gin 教材" }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/sources",
      expect.objectContaining({
        method: "POST",
        json: expect.objectContaining({ official_confirmed: true }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/blueprints/blueprint%201/approve",
      expect.objectContaining({ method: "POST" }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/book-builds",
      expect.objectContaining({ method: "POST", json: {} }),
    );
  });

  it("sends revision guards, task controls and visual metadata without exposing file contents as JSON", async () => {
    await textbookService.getChapterWorkspace("chapter 1");
    await textbookService.getApprovedChapterProjections("chapter 1");
    await textbookService.acquireChapterLock("chapter 1", {
      owner_id: "local-owner",
      expected_version: 2,
      lease_seconds: 90,
    });
    await textbookService.appendDraftRevision("chapter 1", {
      expected_version: 2,
      markdown: "# 路由",
      document_json: { blocks: [] },
      content_hash: "a".repeat(64),
      lock_owner_id: "local-owner",
      lock_version: 3,
    });
    await textbookService.applyCandidate("chapter 1", "revision 2", {
      expected_version: 3,
      lock_owner_id: "local-owner",
      lock_version: 4,
      review_note: "已逐项核对样章内容与教学边界，同意应用。",
      dimension_scores: [
        "句子与段落负担",
        "标题承诺",
        "重复",
        "语气",
        "例子相关性",
        "章节节奏",
        "图示机会",
        "练习梯度",
      ].map((dimension) => ({ dimension, score: 3 })),
    });
    await textbookService.rejectCandidate("chapter 1", "revision 2", {
      expected_version: 3,
      lock_owner_id: "local-owner",
      lock_version: 4,
      reason: "示例没有证明路由查找机制",
    });
    await textbookService.prepareSampleGeneration("project 1", "chapter 1");
    await textbookService.generateSample(
      "project 1",
      "chapter 1",
      "sha256:confirmed",
    );
    await textbookService.getGenerationTask("task 1");
    await textbookService.retryGenerationTask("task 1", "sha256:confirmed");
    await textbookService.retryTask("task 2");
    await textbookService.createSourceImport("project 1", {
      source_id: "source 1",
      file: new File(["package gin"], "routergroup.go", { type: "text/plain" }),
      resolved_version: "73726dc606796a025971fe451f0aa6f1b9b847f6",
    });
    await textbookService.uploadVisualAsset("chapter 1", {
      revisionId: "revision 2",
      evidenceId: "evidence 1",
      file: new File(["image"], "route.png", { type: "image/png" }),
      altText: "路由表截图",
      source: "local IDE",
      generationMethod: "manual_capture",
      visualPurpose: "rendered_ui",
      rightsStatus: "ready",
    });

    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/chapters/chapter%201/lock",
      expect.objectContaining({
        method: "POST",
        json: expect.objectContaining({ expected_version: 2 }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/chapters/chapter%201/revisions",
      expect.objectContaining({
        method: "POST",
        json: expect.objectContaining({ kind: "draft", lock_version: 3 }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/chapters/chapter%201/revisions/revision%202/apply",
      expect.objectContaining({
        method: "POST",
        json: expect.objectContaining({
          expected_version: 3,
          review_note: "已逐项核对样章内容与教学边界，同意应用。",
          dimension_scores: expect.arrayContaining([
            { dimension: "练习梯度", score: 3 },
          ]),
        }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/chapters/chapter%201/revisions/revision%202/reject",
      expect.objectContaining({
        method: "POST",
        json: expect.objectContaining({ reason: "示例没有证明路由查找机制" }),
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/chapters/chapter%201/generation-preflight",
      expect.any(Object),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/project%201/chapters/chapter%201/generate-sample",
      expect.objectContaining({
        method: "POST",
        json: { confirmed_input_hash: "sha256:confirmed" },
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/tasks/task%201",
      expect.any(Object),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/tasks/task%201/retry",
      expect.objectContaining({
        method: "POST",
        json: { confirmed_input_hash: "sha256:confirmed" },
      }),
    );
    expect(requestJson).toHaveBeenCalledWith(
      "/api/v1/textbook-projects/tasks/task%202/retry",
      expect.objectContaining({ method: "POST", json: {} }),
    );

    const importCall = requestJson.mock.calls.find(
      ([url]) => url === "/api/v1/textbook-projects/project%201/source-imports",
    );
    const visualCall = requestJson.mock.calls.find(
      ([url]) =>
        url === "/api/v1/textbook-projects/chapters/chapter%201/visual-assets",
    );
    expect(importCall?.[1]?.body).toBeInstanceOf(FormData);
    expect((importCall?.[1]?.body as FormData).get("source_id")).toBe(
      "source 1",
    );
    expect((importCall?.[1]?.body as FormData).get("resolved_version")).toBe(
      "73726dc606796a025971fe451f0aa6f1b9b847f6",
    );
    expect(visualCall?.[1]?.body).toBeInstanceOf(FormData);
    expect((visualCall?.[1]?.body as FormData).get("visual_purpose")).toBe(
      "rendered_ui",
    );
    expect((visualCall?.[1]?.body as FormData).get("rights_status")).toBe(
      "ready",
    );
  });
});
