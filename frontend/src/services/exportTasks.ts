import { authTokenStore } from '@/lib/authTokenStore'
import { buildAuthHeader, fetchEventSourceWithAuth } from './sse'

export interface ExportTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

interface ExportTaskDependencies {
  fetchImpl?: typeof fetch
  getToken?: () => string | null
  downloadBlob?: (blob: Blob, filename: string) => void
}

const downloadBlobWithBrowser = (blob: Blob, filename: string) => {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  URL.revokeObjectURL(url)
  document.body.removeChild(anchor)
}

const readErrorMessage = async (response: Response, fallback: string) => {
  const payload = (await response.json().catch(() => null)) as { error?: string; message?: string } | null
  return payload?.message || payload?.error || fallback
}

const buildHeaders = (token: string | null, contentType?: string) => {
  const headers: Record<string, string> = {}
  const authHeader = buildAuthHeader(token)
  if (authHeader.Authorization) {
    headers.Authorization = authHeader.Authorization
  }
  if (contentType) {
    headers['Content-Type'] = contentType
  }
  return headers
}

/**
 * Why: PDF 导出改成任务流后，Sidebar 只需要关心“创建任务”这一件事，
 * 具体鉴权头和接口契约由 service 统一维护，避免组件里散落重复实现。
 */
export async function createExportTask(
  blogID: string,
  dependencies: Pick<ExportTaskDependencies, 'fetchImpl' | 'getToken'> = {},
): Promise<ExportTaskResponse> {
  const fetchImpl = dependencies.fetchImpl ?? fetch
  const token = dependencies.getToken?.() ?? authTokenStore.getSnapshot()
  const response = await fetchImpl('/api/v1/tasks/export', {
    method: 'POST',
    headers: buildHeaders(token, 'application/json'),
    body: JSON.stringify({
      kind: 'export_pdf',
      payload: { blog_id: blogID },
      idempotency_key: `export-pdf:${blogID}`,
    }),
  })

  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error('登录已过期，请重新登录')
  }
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, '创建导出任务失败'))
  }
  return (await response.json()) as ExportTaskResponse
}

export async function waitForTaskCompletion(streamURL: string): Promise<void> {
  await fetchEventSourceWithAuth(streamURL, {
    method: 'GET',
    openWhenHidden: true,
    onmessage(message) {
      if (message.event !== 'error') {
        return
      }

      try {
        const payload = JSON.parse(message.data) as { message?: string; error?: string }
        throw new Error(payload.message || payload.error || 'PDF 导出失败')
      } catch (error) {
        throw error instanceof Error ? error : new Error(message.data || 'PDF 导出失败')
      }
    },
  })
}

export async function downloadTaskArtifact(
  taskID: string,
  filename: string,
  dependencies: ExportTaskDependencies = {},
): Promise<void> {
  const fetchImpl = dependencies.fetchImpl ?? fetch
  const token = dependencies.getToken?.() ?? authTokenStore.getSnapshot()
  const response = await fetchImpl(`/api/v1/tasks/${taskID}/download`, {
    method: 'GET',
    headers: buildHeaders(token),
  })

  if (response.status === 401) {
    authTokenStore.clearToken()
    throw new Error('登录已过期，请重新登录')
  }
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, 'PDF 下载失败'))
  }

  const blob = await response.blob()
  const downloadBlob = dependencies.downloadBlob ?? downloadBlobWithBrowser
  downloadBlob(blob, filename)
}
