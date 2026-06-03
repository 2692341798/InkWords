import { authTokenStore } from '@/lib/authTokenStore'
import { buildAuthHeader } from './sse'

export interface CreateGenerationTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

export const buildGenerationTaskRequest = (
  kind: string,
  payload: Record<string, unknown>,
) => ({
  kind,
  payload,
})

/**
 * Why: 任务式生成需要先创建后台任务，再拿 `stream_url` 建立真正的 SSE 订阅；
 * 把这一步抽到服务层后，Hook 只负责编排流程，不直接感知鉴权头与请求细节。
 */
export async function createGenerationTask(
  body: ReturnType<typeof buildGenerationTaskRequest>,
): Promise<CreateGenerationTaskResponse> {
  const authHeader = buildAuthHeader(authTokenStore.getSnapshot())
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (authHeader.Authorization) {
    headers.Authorization = authHeader.Authorization
  }

  const response = await fetch('/api/v1/tasks/generation', {
    method: 'POST',
    headers,
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
