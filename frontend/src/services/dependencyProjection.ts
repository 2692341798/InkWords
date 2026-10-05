import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'
import type { TextbookCodeArtifact } from './textbook'

interface DependencySelection {
  workspace_id: string; project_id: string; chapter_id: string
  module: string; version: string; toolchain: string; dependency_manifest_hash: string
  snapshot: { id: string; locator: string; resolved_version: string; content_hash: string }
  browser_observation?: { kind: string; browser_path?: string; expected_text?: string }
}
export interface DependencyOption { id: string; title: string; selection_hash: string; selection: DependencySelection }
export interface DependencyProjectionRequest { option_id: string; task_id: string; revision_id: string; content_hash: string }
export interface DependencyProjectionPreview {
  contract: 'inkwords.dependency-projection.v1'; request: DependencyProjectionRequest
  artifact_id: string; selection: DependencySelection; selection_hash: string
  book_contract_hash: string; style_sheet_hash: string
  commands: Array<{kind: string; browser_path?: string; expected_text?: string}>
  file_count: number; total_bytes: number; files_hash: string; confirmation_hash: string
}
const base = apiRoutes.coreApi.textbookProjects.dependencyProjection
export const dependencyProjectionService = {
  options: (chapterId: string) => requestJson<{code:number;data:DependencyOption[]}>(`${base(chapterId)}/options`),
  preview: (chapterId: string, input: DependencyProjectionRequest) => requestJson<{code:number;data:DependencyProjectionPreview}>(`${base(chapterId)}/preview`, {method:'POST',json:input}),
  apply: (chapterId: string, preview: DependencyProjectionPreview) => requestJson<{code:number;data:TextbookCodeArtifact}>(`${base(chapterId)}/apply`, {method:'POST',json:{...preview.request,confirmed_preview_hash:preview.confirmation_hash}}),
}
