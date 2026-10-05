import { requestJson } from "./apiClient";
import type { RightsLedger } from './rightsAmendments';
import { apiRoutes } from "./apiRoutes";

export type AudienceLevel = "foundation" | "programming" | "stack_familiar";
export type SourceKind =
  | "git_repository"
  | "official_web"
  | "pdf"
  | "docx"
  | "markdown"
  | "text"
  | "zip";
export type ChapterProfile =
  | "concept"
  | "hands_on"
  | "source_walkthrough"
  | "project_iteration"
  | "troubleshooting"
  | "integration_review"
  | "reference";

export interface TextbookProject {
  id: string;
  title: string;
  audience_level: AudienceLevel;
  status:
    | "draft"
    | "candidate"
    | "approved"
    | "needs_evidence"
    | "verification_failed"
    | "archived";
  revision_version: number;
  primary_source_id?: string;
  approved_book_contract_revision_id?: string;
  approved_style_sheet_revision_id?: string;
  approved_blueprint_revision_id?: string;
  updated_at: string;
}

export interface TextbookSource {
  id: string;
  project_id: string;
  kind: SourceKind;
  role: "primary" | "official_supporting";
  locator: string;
  official_confirmed: boolean;
  license_status: string;
  created_at: string;
}

export interface TextbookSourceSnapshot {
  id: string;
  source_id: string;
  resolved_version: string;
  content_hash: string;
  captured_at: string;
  status: string;
}

export interface TextbookChapter {
  id: string;
  project_id: string;
  sort_order: number;
  title: string;
  chapter_profile: ChapterProfile;
  status: string;
  revision_version: number;
  current_revision_id?: string;
  approved_revision_id?: string;
  updated_at: string;
}

interface SampleGenerationTarget {
  provider_name: string;
  model_name: string;
}

interface SoftQualityAdvisory {
  dimension: string;
  code: string;
  message: string;
  evidence: string;
  review_question: string;
}

export interface TextbookWorkspace {
  project: TextbookProject;
  sources: TextbookSource[];
  chapters: TextbookChapter[];
  book_contract?: GenerationContractRevision;
  style_sheet?: GenerationContractRevision;
  blueprint?: GenerationContractRevision;
  latest_book_build?: TextbookBookBuild;
}

export type RightsWorkType =
  "prose" | "code" | "image" | "screenshot" | "font" | "trademark" | "data";
export type RightsStatus = "pending" | "ready" | "blocked";
export type PublicationReviewStage =
  | "developmental"
  | "technical"
  | "self_study"
  | "consistency"
  | "copy_editing"
  | "layout"
  | "rights"
  | "reader_trial";
export interface PublicationRightsItem {
  id: string;
  build_id: string;
  project_id: string;
  subject_ref: string;
  work_type: RightsWorkType;
  rights_basis: string;
  allowed_use: string;
  attribution: string;
  publication_status: RightsStatus;
  created_at: string;
}
interface RequiredRightsSubject {
  subject_ref: string;
  work_type: RightsWorkType;
}
export interface HumanPublicationReview {
  contract_version?: 'inkwords.human-publication-review.v2';
  manifest_hash?: string;
  revision?: number;
  reviewer_kind?: 'human';
  verdict?: 'pass' | 'needs_revision' | 'not_assessed';
  score?: number;
  scope?: string;
  evidence_refs?: string[];
  hard_failures?: string[];
  id: string;
  build_id: string;
  stage: PublicationReviewStage;
  reviewer: string;
  notes: string;
  automated: false;
  completed_at: string;
  created_at: string;
}
interface AutomatedPublicationCheck {
  id: string;
  detector: string;
  status: "pass" | "soft_fail" | "hard_fail";
}
export interface HumanPublicationReviewInput {
  id: string;
  manifest_hash: string;
  expected_revision: number;
  stage: PublicationReviewStage;
  reviewer: string;
  verdict: 'pass' | 'needs_revision' | 'not_assessed';
  score: number;
  scope: string;
  notes: string;
  evidence_refs: string[];
  hard_failures: string[];
}
export interface DelegatedPublicationReviewInput {
  id: string;
  manifest_hash: string;
  expected_revision: number;
  stage: PublicationReviewStage;
  reviewer: string;
  delegation_note: string;
  verdict: 'pass' | 'needs_revision' | 'not_assessed';
  score: number;
  scope: string;
  notes: string;
  evidence_refs: string[];
  hard_failures: string[];
}
export interface DelegatedPublicationReview extends Omit<DelegatedPublicationReviewInput, 'expected_revision'> {
  contract_version: 'inkwords.delegated-publication-review.v1';
  build_id: string;
  revision: number;
  reviewer_kind: 'delegated_ai';
  completed_at: string;
}
interface PublicationPreflight {
  passed: boolean;
  blockers: string[];
}
export interface EditorialWorkspace {
  build: TextbookBookBuild;
  required_rights_subjects: RequiredRightsSubject[];
  rights_items: PublicationRightsItem[];
  rights_ledger?: RightsLedger;
  human_reviews: HumanPublicationReview[];
  delegated_review_contract?: string;
  delegated_reviews?: DelegatedPublicationReview[];
  automated_checks: AutomatedPublicationCheck[];
  preflight: PublicationPreflight;
}
type TextbookStageStatus =
  "blocked" | "ready" | "in_progress" | "approved" | "unavailable";
export interface TextbookStageState {
  key:
    | "sources"
    | "blueprint"
    | "sample"
    | "chapters"
    | "verification"
    | "learning"
    | "publication";
  status: TextbookStageStatus;
  reason: string;
}
export interface TextbookProjectProgress {
  stages: TextbookStageState[];
}
export interface TextbookBookBuild {
  id: string;
  project_id: string;
  book_contract_revision_id: string;
  style_sheet_revision_id: string;
  approved_revision_ids: string[];
  tool_versions_json?: Record<string, string>;
  manifest_json: Record<string, unknown>;
  manifest_hash: string;
  status: "draft" | "ready_for_review" | "publication_candidate" | "blocked";
  blockers_json: string[];
  created_at: string;
}
export interface GenerationContractRevision {
  id: string;
  project_id: string;
  revision_number: number;
  document_json: Record<string, unknown>;
  content_hash: string;
  status: "draft" | "approved";
  created_at: string;
}
export interface SourceLibraryDocument {
  id: string;
  source_id: string;
  snapshot_id: string;
  canonical_locator: string;
  title: string;
  media_type: string;
  content_hash: string;
  artifact_path?: string;
  chunk_count: number;
}
export interface SourceLibraryEvidence {
  locator?: { path?: string; symbol?: string; start_line?: number; end_line?: number };
  id: string;
  document_id: string;
  document_title: string;
  canonical_locator: string;
  ordinal: number;
}
interface SourceRetrievalCandidate {
	locator?: { path?: string; symbol?: string; start_line?: number; end_line?: number };
  chunk_id: string;
  document_id: string;
  document_title: string;
  snapshot_id: string;
  source_role: "primary" | "official_supporting";
  canonical_locator: string;
  artifact_path?: string;
  ordinal: number;
  score: number;
  reasons: string[];
}
export interface SourceRetrievalPlan {
  id: string;
  query: string;
  input_hash: string;
  candidates: SourceRetrievalCandidate[];
  selected: SourceRetrievalCandidate[];
  created_at: string;
}
interface ClaimRequirementInput {
  id: string;
  label: string;
  evidence_ids: string[];
}
interface BlueprintChapterInput {
  id: string;
  title: string;
  sort: number;
  profile: ChapterProfile;
  evidence_ids: string[];
  critical_claims: ClaimRequirementInput[];
}
export interface BlueprintVolumeInput {
  id: string;
  title: string;
  sort: number;
  chapters: BlueprintChapterInput[];
}

export interface ChapterRevision {
  id: string;
  chapter_id: string;
  revision_number: number;
  kind: "draft" | "candidate" | "approved";
  markdown: string;
  document_json: Record<string, unknown>;
  content_hash: string;
  created_by: "manual" | "generation";
  generation_task_id?: string;
  parent_revision_id?: string;
  book_contract_revision_id?: string;
  style_sheet_revision_id?: string;
  blueprint_revision_id?: string;
  evidence_pack_hash?: string;
  prompt_hash?: string;
  provider_name?: string;
  model_name?: string;
  provider_usage_json?: Record<string, unknown>;
  quality_report_json?: {
    contract_version?: string;
    passed?: boolean;
    failures?: string[];
    critical_claim_count?: number;
    covered_critical_claim_count?: number;
    evidence_reference_count?: number;
    manual_review_required?: boolean;
    manual_review_dimensions?: string[];
    advisories?: SoftQualityAdvisory[];
  };
  created_at: string;
}

export interface CandidateReview {
  id: string;
  project_id: string;
  chapter_id: string;
  candidate_revision_id: string;
  reviewer_workspace_id: string;
  decision: "approved" | "rejected";
  reason: string;
  candidate_content_hash: string;
  quality_contract_version: string;
  human_review?: {
    reviewer_kind?: "human" | "delegated_ai";
    delegation_note?: string;
    contract_version: string;
    dimension_scores: Array<{ dimension: string; score: number }>;
  };
  created_at: string;
}

interface DemonstrationToolRecommendation {
  primary: string;
  alternative: string;
  reason: string;
  manual_capture_required: boolean;
}

export interface VideoRunbookStep {
  tool: string;
  tool_version: string;
  start_state: string;
  action: string;
  shortcut_or_menu: string;
  input: string;
  expected_view: string;
  narration: string;
  capture_point: string;
  recovery: string;
  completion_signal: string;
}

export interface VideoRunbookProjection {
  format: "inkwords.video-runbook.v1";
  stack: string;
  observation_goal: string;
  recommendation: DemonstrationToolRecommendation;
  steps: VideoRunbookStep[];
  capture_checklist: string[];
  manual_capture_pending: boolean;
  verification_status: "draft" | "unverified" | "verified" | "blocked";
}

interface BlogProjection {
  format: "inkwords.blog-projection.v1";
  revision_id: string;
  chapter_id: string;
  title: string;
  markdown: string;
  content_hash: string;
}

export interface PracticeTask {
  id: string;
  mode: "explain" | "complete" | "reproduce" | "transfer" | "diagnose" | "retain";
  prompt: string;
  variation: string;
  expected_answer: string;
  rubric: Array<{ id: string; description: string; requires_runtime: boolean }>;
  hints: Array<{ level: number; text: string }>;
  evidence_ids: string[];
  min_delay_hours: number;
}

export interface LearningProjection {
  format: "inkwords.learning-projection.v1" | "inkwords.learning-projection.v2";
  practice_set?: { version: "inkwords.practice-set.v1"; tasks: PracticeTask[] };
  evidence_ids?: string[];
  revision_id: string;
  chapter_id: string;
  content_hash: string;
  learning_arc: {
    stages: Array<{
      stage: string;
      objective: string;
      prerequisites?: string[];
      allowed_hint_level: number;
      success_evidence: string[];
      recovery_route: string;
    }>;
  };
  objectives: Array<{
    id: string;
    chapter_id: string;
    text: string;
    required_modes: Array<
      "explain" | "complete" | "reproduce" | "transfer" | "diagnose" | "retain"
    >;
  }>;
}

export interface ApprovedChapterProjections {
  blog: BlogProjection;
  video_runbook: VideoRunbookProjection;
  learning: LearningProjection;
}

export interface ChapterLock {
  chapter_id: string;
  owner_id: string;
  version: number;
  lease_expires_at: string;
}

export type ArtifactStatus = "draft" | "unverified" | "verified" | "blocked";

export interface TextbookCodeArtifact {
  id: string;
  revision_id: string;
  kind:
    "teaching_implementation" | "integration_example" | "upstream_walkthrough";
  language: string;
  entrypoint?: string;
  manifest_hash: string;
  artifact_hash: string;
  limitations_json: string[];
  status: ArtifactStatus;
  created_at: string;
}

export interface TextbookRuntimeEvidence {
  id: string;
  revision_id: string;
  code_artifact_id: string;
  code_artifact_hash: string;
  input_hash: string;
  kind: "terminal_output" | "browser_page" | "ide_capture" | "image";
  status: ArtifactStatus;
  command_manifest_hash?: string;
  runner_image_digest?: string;
  toolchain_version?: string;
  tool_name?: string;
  tool_version?: string;
  structured_output?: string;
  raw_evidence_ref?: string;
  output_truncated: boolean;
  captured_at?: string;
  expires_at?: string;
  stale_reason?: string;
  created_at: string;
}

export interface TextbookManuscriptAsset {
  id: string;
  revision_id: string;
  evidence_id: string;
  stable_ref: string;
  kind: "screenshot" | "diagram" | "recording";
  content_hash: string;
  alt_text: string;
  source: string;
  generation_method: string;
  visual_purpose:
    | "layout"
    | "memory_map"
    | "call_stack"
    | "network_flow"
    | "rendered_ui"
    | "legacy_unclassified";
  rights_status: "pending" | "ready" | "blocked";
  status: ArtifactStatus;
  created_at: string;
}

export interface ChapterWorkspace {
  chapter: TextbookChapter;
  generation_target: SampleGenerationTarget;
  human_review_contract?: SampleHumanReviewContract;
  revisions: ChapterRevision[];
  code_artifacts: TextbookCodeArtifact[];
  runtime_evidence: TextbookRuntimeEvidence[];
  assets: TextbookManuscriptAsset[];
  candidate_reviews: CandidateReview[];
  latest_sample_task?: {
    task_id: string;
    status: TextbookTaskStatus;
    created_at: string;
  };
  lock?: ChapterLock;
}

export interface SampleHumanReviewContract {
  reviewer_kinds?: Array<"human" | "delegated_ai">;
  contract_version: string;
  dimensions: string[];
  minimum_score: number;
  maximum_score: number;
  note_min_runes: number;
  note_max_runes: number;
}

type TextbookTaskStatus =
  | "pending"
  | "queued"
  | "running"
  | "streaming"
  | "succeeded"
  | "failed"
  | "cancelled";
export interface TextbookGenerationTask {
  task_id: string;
  status: TextbookTaskStatus;
}
export interface SampleGenerationPreflight {
  task_version: number;
  prompt_schema_version: string;
  quality_contract_version: string;
  generation_target: SampleGenerationTarget;
  input_hash: string;
  evidence_reference_count: number;
  estimated_input_tokens: number;
  allowed_input_tokens: number;
  reserved_output_tokens: number;
  within_budget: boolean;
  estimate_method: string;
  cache_status: "unknown_before_worker";
  estimated_cost_known: false;
  requires_confirmation: true;
}
export interface ProviderConnectionResult {
  provider: string;
  model: string;
  request_id?: string;
  usage_known: boolean;
}
export interface VerificationAttemptInput {
  request_id: string;
  expected_previous_task_id?: string;
}
export interface TextbookVerificationAttempt {
  id: string;
  status: TextbookTaskStatus;
  attempt?: number;
  previous_task_id?: string;
  execution_stopped?: boolean;
  can_start_new?: boolean;
  attempt_contract?: string;
  history?: TextbookVerificationAttempt[];
}
export interface TextbookGenerationTaskSnapshot {
  execution_mode?: "automated_local_correction";
  id: string;
  task_type: string;
  task_subtype: string;
  status: TextbookTaskStatus;
  error_message?: string;
  retry_count: number;
  result?: unknown;
  retry_confirmation?: {
    execution_mode?: "automated_local_correction";
    generation_target: SampleGenerationTarget;
    input_hash: string;
    prompt_schema_version: string;
    quality_contract_version: string;
    estimated_input_tokens: number;
    allowed_input_tokens: number;
    reserved_output_tokens: number;
    estimated_cost_known: false;
    requires_confirmation: true;
  };
}
export interface TextbookSourceImportTask {
  id: string;
  status: TextbookTaskStatus;
}

interface Envelope<T> {
  code: number | string;
  data: T;
}

export const textbookService = {
  testConfiguredProviderConnection: () =>
    requestJson<Envelope<ProviderConnectionResult>>(
      apiRoutes.llmStream.providerConnectionTest,
      { method: "POST", json: {}, fallbackMessage: "模型连接测试失败" },
    ),
  list: () =>
    requestJson<Envelope<TextbookProject[]>>(
      apiRoutes.coreApi.textbookProjects.collection,
      { fallbackMessage: "加载教材项目失败" },
    ),
  get: (projectId: string) =>
    requestJson<Envelope<TextbookProject>>(
      apiRoutes.coreApi.textbookProjects.byId(projectId),
      { fallbackMessage: "加载教材项目失败" },
    ),
  getWorkspace: (projectId: string) =>
    requestJson<Envelope<TextbookWorkspace>>(
      apiRoutes.coreApi.textbookProjects.workspace(projectId),
      { fallbackMessage: "加载教材工作台失败" },
    ),
  getProgress: (projectId: string) =>
    requestJson<Envelope<TextbookProjectProgress>>(
      apiRoutes.coreApi.textbookProjects.progress(projectId),
      { fallbackMessage: "加载教材生产阶段失败" },
    ),
  getSourceLibrary: (projectId: string) =>
    requestJson<Envelope<SourceLibraryDocument[]>>(
      apiRoutes.coreApi.textbookProjects.sourceLibrary(projectId),
      { fallbackMessage: "加载资料库失败" },
    ),
  getSourceEvidence: (projectId: string, chunkIDs: string[] = []) =>
    requestJson<Envelope<SourceLibraryEvidence[]>>(
      apiRoutes.coreApi.textbookProjects.sourceEvidence(projectId) + (chunkIDs.length ? `?${new URLSearchParams(chunkIDs.map((id) => ['chunk_id', id]))}` : ''),
      { fallbackMessage: "加载可引用资料片段失败" },
    ),
  retrieveSourceEvidence: (
    projectId: string,
    input: { query: string; limit: number },
  ) =>
    requestJson<Envelope<SourceRetrievalPlan>>(
      apiRoutes.coreApi.textbookProjects.sourceRetrieval(projectId),
      { method: "POST", json: input, fallbackMessage: "检索资料失败" },
    ),
  create: (input: {
    title: string;
    audience_level: AudienceLevel;
    primary_source: { kind: SourceKind; locator: string };
  }) =>
    requestJson<Envelope<TextbookProject>>(
      apiRoutes.coreApi.textbookProjects.collection,
      { method: "POST", json: input, fallbackMessage: "创建教材项目失败" },
    ),
  addSource: (
    projectId: string,
    input: {
      kind: SourceKind;
      role: "official_supporting";
      locator: string;
      official_confirmed: true;
      license_status?: string;
    },
  ) =>
    requestJson<Envelope<TextbookSource>>(
      apiRoutes.coreApi.textbookProjects.sources(projectId),
      { method: "POST", json: input, fallbackMessage: "添加官方资料失败" },
    ),
  loadGinFixture: (projectId: string) =>
    requestJson<Envelope<TextbookSourceSnapshot>>(
      apiRoutes.coreApi.textbookProjects.loadGinFixture(projectId),
      {
        method: "POST",
        json: {},
        fallbackMessage: "载入固定 Gin 样章资料失败",
      },
    ),
  createSourceImport: (
    projectId: string,
    input: { source_id: string; file: File; resolved_version?: string },
  ) => {
    const form = new FormData();
    form.append("source_id", input.source_id);
    if (input.resolved_version)
      form.append("resolved_version", input.resolved_version);
    form.append("file", input.file, input.file.name);
    return requestJson<Envelope<TextbookSourceImportTask>>(
      apiRoutes.coreApi.textbookProjects.sourceImports(projectId),
      { method: "POST", body: form, fallbackMessage: "创建资料导入任务失败" },
    );
  },
  createOfficialWebImport: (
    projectId: string,
    input: { source_id: string; allowed_path_prefixes: string[] },
  ) =>
    requestJson<Envelope<TextbookSourceImportTask>>(
      apiRoutes.coreApi.textbookProjects.officialWebImports(projectId),
      {
        method: "POST",
        json: input,
        fallbackMessage: "创建官网资料导入任务失败",
      },
    ),
  createChapter: (
    projectId: string,
    input: {
      sort_order: number;
      title: string;
      chapter_profile: ChapterProfile;
    },
  ) =>
    requestJson<Envelope<TextbookChapter>>(
      apiRoutes.coreApi.textbookProjects.chapters(projectId),
      { method: "POST", json: input, fallbackMessage: "创建章节失败" },
    ),
  createBookContract: (
    projectId: string,
    input: {
      reader: {
        audience: AudienceLevel;
        known_knowledge: string[];
        forbidden_assumptions: string[];
        learning_outcomes: string[];
      };
      promise: string;
      chapter_profiles: ChapterProfile[];
      terminology_version: string;
      publication_profile: string;
    },
  ) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.bookContracts(projectId),
      {
        method: "POST",
        json: input,
        fallbackMessage: "保存 BookContract 草稿失败",
      },
    ),
  createStyleSheet: (
    projectId: string,
    input: {
      language: string;
      terminology_rules: string[];
      code_rules: string[];
      visual_rules: string[];
      citation_rules: string[];
      forbidden_phrases: string[];
    },
  ) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.styleSheets(projectId),
      {
        method: "POST",
        json: input,
        fallbackMessage: "保存 StyleSheet 草稿失败",
      },
    ),
  approveBookContract: (projectId: string, revisionId: string) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.approveBookContract(
        projectId,
        revisionId,
      ),
      { method: "POST", json: {}, fallbackMessage: "批准 BookContract 失败" },
    ),
  approveStyleSheet: (projectId: string, revisionId: string) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.approveStyleSheet(
        projectId,
        revisionId,
      ),
      { method: "POST", json: {}, fallbackMessage: "批准 StyleSheet 失败" },
    ),
  createBlueprint: (
    projectId: string,
    input: { volumes: BlueprintVolumeInput[] },
  ) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.blueprints(projectId),
      { method: "POST", json: input, fallbackMessage: "保存教学蓝图草稿失败" },
    ),
  approveBlueprint: (projectId: string, revisionId: string) =>
    requestJson<Envelope<GenerationContractRevision>>(
      apiRoutes.coreApi.textbookProjects.approveBlueprint(
        projectId,
        revisionId,
      ),
      { method: "POST", json: {}, fallbackMessage: "批准教学蓝图失败" },
    ),
  createBookBuild: (projectId: string, notices: import('./publicationNotices').PublicationNoticeDraft[] = []) =>
    requestJson<Envelope<TextbookBookBuild>>(
      apiRoutes.coreApi.textbookProjects.bookBuilds(projectId),
      { method: "POST", json: notices.length ? { publication_notices: notices } : {}, fallbackMessage: "冻结整书构建失败" },
    ),
  getEditorialWorkspace: (buildId: string) =>
    requestJson<Envelope<EditorialWorkspace>>(
      apiRoutes.coreApi.textbookProjects.editorialWorkspace(buildId),
      { fallbackMessage: "加载出版审校工作台失败" },
    ),
  addPublicationRight: (
    buildId: string,
    input: {
      subject_ref: string;
      work_type: RightsWorkType;
      rights_basis: string;
      allowed_use: string;
      attribution: string;
      publication_status: RightsStatus;
    },
  ) =>
    requestJson<Envelope<PublicationRightsItem>>(
      apiRoutes.coreApi.textbookProjects.publicationRights(buildId),
      { method: "POST", json: input, fallbackMessage: "登记权利项失败" },
    ),
  completePublicationReview: (
    buildId: string,
    input: HumanPublicationReviewInput,
  ) =>
    requestJson<Envelope<HumanPublicationReview>>(
      apiRoutes.coreApi.textbookProjects.publicationReviews(buildId),
      { method: "POST", json: input, fallbackMessage: "记录人工审校失败" },
    ),
  recordDelegatedPublicationReview: (buildId: string, input: DelegatedPublicationReviewInput) =>
    requestJson<Envelope<DelegatedPublicationReview>>(
      apiRoutes.coreApi.textbookProjects.delegatedPublicationReviews(buildId),
      { method: 'POST', json: input, fallbackMessage: '记录委托 AI 审阅失败' },
    ),
  promoteBookBuild: (buildId: string) =>
    requestJson<Envelope<TextbookBookBuild>>(
      apiRoutes.coreApi.textbookProjects.publicationCandidate(buildId),
      { method: "POST", json: {}, fallbackMessage: "标记出版候选失败" },
    ),
  getChapterWorkspace: (chapterId: string) =>
    requestJson<Envelope<ChapterWorkspace>>(
      apiRoutes.coreApi.textbookProjects.chapterWorkspace(chapterId),
      { fallbackMessage: "加载章节编辑器失败" },
    ),
  getArtifactVerification: (chapterId: string, artifactId: string) =>
    requestJson<Envelope<TextbookVerificationAttempt | null>>(
      apiRoutes.coreApi.textbookProjects.artifactVerification(chapterId, artifactId),
      { fallbackMessage: '查询代码验证任务失败' },
    ),
  startArtifactVerification: (chapterId: string, artifactId: string, input?: VerificationAttemptInput) =>
    requestJson<Envelope<TextbookVerificationAttempt>>(
      apiRoutes.coreApi.textbookProjects.artifactVerification(chapterId, artifactId),
      { method: 'POST', json: input ?? {}, fallbackMessage: '启动隔离验证失败' },
    ),
  cancelVerificationTask: (taskId: string) =>
    requestJson<unknown>(apiRoutes.coreApi.tasks.cancel(taskId),
      { method: 'POST', json: {}, fallbackMessage: '取消代码验证失败' }),
  getApprovedChapterProjections: (chapterId: string) =>
    requestJson<Envelope<ApprovedChapterProjections>>(
      apiRoutes.coreApi.textbookProjects.chapterProjections(chapterId),
      { fallbackMessage: "加载批准稿派生视图失败" },
    ),
  acquireChapterLock: (
    chapterId: string,
    input: {
      owner_id: string;
      expected_version: number;
      lease_seconds: number;
    },
  ) =>
    requestJson<Envelope<ChapterLock>>(
      apiRoutes.coreApi.textbookProjects.chapterLock(chapterId),
      { method: "POST", json: input, fallbackMessage: "获取编辑锁失败" },
    ),
  appendDraftRevision: (
    chapterId: string,
    input: {
      expected_version: number;
      markdown: string;
      document_json: Record<string, unknown>;
      content_hash: string;
      lock_owner_id: string;
      lock_version: number;
    },
  ) =>
    requestJson<Envelope<ChapterRevision>>(
      apiRoutes.coreApi.textbookProjects.revisions(chapterId),
      {
        method: "POST",
        json: { ...input, kind: "draft" },
        fallbackMessage: "保存章节草稿失败",
      },
    ),
  applyCandidate: (
    chapterId: string,
    revisionId: string,
    input: {
      expected_version: number;
      lock_owner_id: string;
      lock_version: number;
      review_note: string;
      reviewer_kind?: "human" | "delegated_ai";
      delegation_note?: string;
      dimension_scores: Array<{ dimension: string; score: number }>;
    },
  ) =>
    requestJson<Envelope<ChapterRevision>>(
      apiRoutes.coreApi.textbookProjects.applyCandidate(chapterId, revisionId),
      { method: "POST", json: input, fallbackMessage: "应用候选稿失败" },
    ),
  rejectCandidate: (
    chapterId: string,
    revisionId: string,
    input: {
      expected_version: number;
      lock_owner_id: string;
      lock_version: number;
      reason: string;
    },
  ) =>
    requestJson<Envelope<CandidateReview>>(
      apiRoutes.coreApi.textbookProjects.rejectCandidate(chapterId, revisionId),
      { method: "POST", json: input, fallbackMessage: "驳回候选稿失败" },
    ),
  prepareSampleGeneration: (projectId: string, chapterId: string) =>
    requestJson<Envelope<SampleGenerationPreflight>>(
      apiRoutes.coreApi.textbookProjects.generationPreflight(
        projectId,
        chapterId,
      ),
      { fallbackMessage: "生成前预检失败" },
    ),
  generateSample: (
    projectId: string,
    chapterId: string,
    confirmedInputHash: string,
  ) =>
    requestJson<Envelope<TextbookGenerationTask>>(
      apiRoutes.coreApi.textbookProjects.generateSample(projectId, chapterId),
      {
        method: "POST",
        json: { confirmed_input_hash: confirmedInputHash },
        fallbackMessage: "创建样章生成任务失败",
      },
    ),
  correctSample: (projectId: string, chapterId: string, originalTaskId: string, edit: Record<string, unknown>) =>
    requestJson<Envelope<TextbookGenerationTask>>(
      apiRoutes.coreApi.textbookProjects.correctSample(projectId, chapterId),
      { method: "POST", json: { original_task_id: originalTaskId, edit }, fallbackMessage: "提交局部修订失败" },
    ),
  getTask: (taskId: string) =>
    requestJson<TextbookGenerationTaskSnapshot>(
      apiRoutes.coreApi.textbookProjects.task(taskId),
      { fallbackMessage: "查询任务失败" },
    ),
  getGenerationTask: (taskId: string) =>
    requestJson<TextbookGenerationTaskSnapshot>(
      apiRoutes.coreApi.textbookProjects.task(taskId),
      { fallbackMessage: "查询候选教材稿任务失败" },
    ),
  retryGenerationTask: (taskId: string, confirmedInputHash: string) =>
    requestJson<TextbookGenerationTaskSnapshot>(
      apiRoutes.coreApi.textbookProjects.retryTask(taskId),
      { method: "POST", json: { confirmed_input_hash: confirmedInputHash }, fallbackMessage: "重试候选教材稿任务失败" },
    ),
  retryTask: (taskId: string) =>
    requestJson<TextbookGenerationTaskSnapshot>(
      apiRoutes.coreApi.textbookProjects.retryTask(taskId),
      { method: "POST", json: {}, fallbackMessage: "重试冻结任务失败" },
    ),
  uploadVisualAsset: (
    chapterId: string,
    input: {
      revisionId: string;
      evidenceId: string;
      file: File;
      altText: string;
      source: string;
      generationMethod: string;
      visualPurpose:
        "layout" | "memory_map" | "call_stack" | "network_flow" | "rendered_ui";
      rightsStatus: "pending" | "ready" | "blocked";
    },
  ) => {
    const form = new FormData();
    form.append("revision_id", input.revisionId);
    form.append("evidence_id", input.evidenceId);
    form.append("file", input.file, input.file.name);
    form.append("alt_text", input.altText);
    form.append("source", input.source);
    form.append("generation_method", input.generationMethod);
    form.append("visual_purpose", input.visualPurpose);
    form.append("rights_status", input.rightsStatus);
    return requestJson<Envelope<TextbookManuscriptAsset>>(
      apiRoutes.coreApi.textbookProjects.uploadVisualAsset(chapterId),
      { method: "POST", body: form, fallbackMessage: "登记截图资产失败" },
    );
  },
};
