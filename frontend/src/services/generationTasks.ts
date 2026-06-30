import type { ScenarioMode } from '@/lib/scenarioMode'
import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'

export interface CreateGenerationTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

export interface CancelGenerationTaskResponse {
  task_id: string
  status: string
}

interface SeriesChapter {
  id?: string
  title: string
  summary: string
  sort: number
  files?: string[]
  action?: 'new' | 'regenerate' | 'skip' | string
}

export interface BuildSeriesGenerationPayloadInput {
  sourceType: 'git' | 'file' | null
  gitUrl: string
  sourceContent: string
  seriesTitle: string
  outline: SeriesChapter[] | null
  parentBlogId: string | null
  scenarioMode: ScenarioMode
  promptProfileKey?: string
  documentKind?: string
}

export const buildGenerationTaskRequest = (
  kind: string,
  payload: Record<string, unknown>,
) => ({
  kind,
  payload,
})

export const buildSeriesGenerationPayload = (input: BuildSeriesGenerationPayloadInput) => ({
  source_type: input.sourceType,
  git_url: input.gitUrl,
  source_content: input.sourceContent,
  series_title: input.seriesTitle,
  outline: input.outline,
  parent_id: input.parentBlogId,
  scenario_mode: input.scenarioMode,
  prompt_profile_key: input.promptProfileKey,
  document_kind: input.documentKind,
})

export const buildSingleGenerationPayload = (
  content: string,
  scenarioMode: ScenarioMode,
  promptProfileKey?: string,
  documentKind?: string,
) => ({
  source_type: 'file' as const,
  source_content: content,
  outline: [],
  scenario_mode: scenarioMode,
  prompt_profile_key: promptProfileKey,
  document_kind: documentKind,
})

export const buildContinueTaskPayload = (blogId: string) => ({
  blog_id: blogId,
})

export const buildPolishTaskPayload = (blogId: string, title: string, content: string) => ({
  blog_id: blogId,
  title,
  content,
})

export function extractTaskChunkContent(rawData: string) {
  const trimmed = rawData.trim()
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) {
    return rawData
  }

  try {
    const parsed = JSON.parse(rawData) as { content?: unknown }
    if (typeof parsed.content === 'string') {
      return parsed.content
    }
  } catch {
    return rawData
  }

  return rawData
}

/**
 * Why: 任务式生成需要先创建后台任务，再拿 `stream_url` 建立真正的 SSE 订阅；
 * 把这一步抽到服务层后，Hook 只负责编排流程，不直接感知鉴权头与请求细节。
 */
export async function createGenerationTask(
  body: ReturnType<typeof buildGenerationTaskRequest>,
): Promise<CreateGenerationTaskResponse> {
  return requestJson<CreateGenerationTaskResponse>(apiRoutes.coreApi.tasks.generation, {
    method: 'POST',
    json: body,
    fallbackMessage: '创建生成任务失败',
  })
}

export async function cancelGenerationTask(taskID: string): Promise<CancelGenerationTaskResponse> {
  return requestJson<CancelGenerationTaskResponse>(apiRoutes.coreApi.tasks.cancel(taskID), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    fallbackMessage: '取消生成任务失败',
  })
}
