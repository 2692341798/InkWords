import { requestBlob, requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'
import { fetchEventSourceLocal } from './sse'

export interface ExportTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

interface ExportTaskDependencies {
  fetchImpl?: typeof fetch
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

/**
 * Why: PDF 导出改成任务流后，Sidebar 只需要关心“创建任务”这一件事，
 * 具体本地 workspace 接口契约由 service 统一维护，避免组件里散落重复实现。
 */
export async function createExportTask(
  blogID: string,
  dependencies: Pick<ExportTaskDependencies, 'fetchImpl'> = {},
): Promise<ExportTaskResponse> {
  const fetchImpl = dependencies.fetchImpl ?? fetch
  return requestJson<ExportTaskResponse>(apiRoutes.coreApi.tasks.export, {
    method: 'POST',
    json: {
      kind: 'export_pdf',
      payload: { blog_id: blogID },
      idempotency_key: `export-pdf:${blogID}`,
    },
    fetchImpl,
    fallbackMessage: '创建导出任务失败',
  })
}

export async function waitForTaskCompletion(streamURL: string): Promise<void> {
  await fetchEventSourceLocal(streamURL, {
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
  const blob = await requestBlob(apiRoutes.coreApi.tasks.download(taskID), {
    method: 'GET',
    fetchImpl,
    fallbackMessage: 'PDF 下载失败',
  })
  const downloadBlob = dependencies.downloadBlob ?? downloadBlobWithBrowser
  downloadBlob(blob, filename)
}
