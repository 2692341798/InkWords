import { authTokenStore } from '@/lib/authTokenStore'
import type { ScenarioMode } from '@/lib/scenarioMode'
import { buildAuthHeader } from './sse'

export interface CreateGenerationTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

export interface CancelGenerationTaskResponse {
  task_id: string
  status: string
}

export interface SeriesChapter {
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

const buildJSONRequestHeaders = () => {
  const authHeader = buildAuthHeader(authTokenStore.getSnapshot())
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (authHeader.Authorization) {
    headers.Authorization = authHeader.Authorization
  }
  return headers
}

/**
 * Why: 任务式生成需要先创建后台任务，再拿 `stream_url` 建立真正的 SSE 订阅；
 * 把这一步抽到服务层后，Hook 只负责编排流程，不直接感知鉴权头与请求细节。
 */
export async function createGenerationTask(
  body: ReturnType<typeof buildGenerationTaskRequest>,
): Promise<CreateGenerationTaskResponse> {
  const response = await fetch('/api/v1/tasks/generation', {
    method: 'POST',
    headers: buildJSONRequestHeaders(),
    body: JSON.stringify(body),
  })

  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error('登录已过期，请重新登录')
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as
      | { error?: string; message?: string }
      | null
    throw new Error(payload?.message || payload?.error || '创建生成任务失败')
  }

  return (await response.json()) as CreateGenerationTaskResponse
}

export async function cancelGenerationTask(taskID: string): Promise<CancelGenerationTaskResponse> {
  const response = await fetch(`/api/v1/tasks/${taskID}/cancel`, {
    method: 'POST',
    headers: buildJSONRequestHeaders(),
  })

  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error('登录已过期，请重新登录')
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as
      | { error?: string; message?: string }
      | null
    throw new Error(payload?.message || payload?.error || '取消生成任务失败')
  }

  return (await response.json()) as CancelGenerationTaskResponse
}
