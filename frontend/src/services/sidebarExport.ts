import type { BlogNode } from '@/store/blogStore'
import {
  createExportTask as createExportTaskRequest,
  downloadTaskArtifact as downloadTaskArtifactRequest,
  waitForTaskCompletion as waitForTaskCompletionRequest,
} from './exportTasks'

interface SidebarExportDependencies {
  fetchImpl?: typeof fetch
  getToken?: () => string | null
  downloadBlob?: (blob: Blob, filename: string) => void
  createExportTask?: (blogID: string) => Promise<{ task_id: string; stream_url: string }>
  waitForTaskCompletion?: (
    streamURL: string,
  ) => Promise<void | { status?: string; error_message?: string }>
  downloadTaskArtifact?: (taskID: string, filename: string) => Promise<void>
}

interface SidebarPdfExportFailure {
  id: string
  title: string
  message: string
}

interface SidebarPdfExportResult {
  succeededCount: number
  failed: SidebarPdfExportFailure[]
}

function getAuthHeaders(token: string | null): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function sanitizeDownloadFilename(name: string) {
  return (name || 'series').replaceAll('/', '-').replaceAll('\\', '-').replaceAll(':', '：').trim()
}

/**
 * Why: Sidebar 只需要关心交互状态与 toast，不应该内联维护多个导出接口细节；
 * 将请求与下载副作用收口到 service 后，组件才能稳定做 TDD 和后续复用。
 */
export async function syncSeriesToObsidian(seriesRoots: BlogNode[], dependencies: SidebarExportDependencies = {}) {
  const fetchImpl = dependencies.fetchImpl ?? fetch
  const getToken = dependencies.getToken ?? (() => localStorage.getItem('token'))
  const token = getToken()

  for (const series of seriesRoots) {
    const response = await fetchImpl(`/api/v1/blogs/${series.id}/export/obsidian/series`, {
      method: 'POST',
      headers: getAuthHeaders(token),
    })

    const data = await response.json().catch(() => null)
    if (!response.ok || (typeof data === 'object' && data !== null && 'code' in data && data.code !== 200)) {
      const message =
        typeof data === 'object' && data !== null && 'message' in data && typeof data.message === 'string'
          ? data.message
          : '同步系列失败'
      throw new Error(message)
    }
  }

  return seriesRoots.length
}

export async function exportSeriesPdfs(
  seriesRoots: BlogNode[],
  dependencies: SidebarExportDependencies = {},
): Promise<SidebarPdfExportResult> {
  const createExportTask = dependencies.createExportTask ?? ((blogID: string) => createExportTaskRequest(blogID, dependencies))
  const waitForTaskCompletion = dependencies.waitForTaskCompletion ?? waitForTaskCompletionRequest
  const downloadTaskArtifact = dependencies.downloadTaskArtifact ?? ((taskID: string, filename: string) =>
    downloadTaskArtifactRequest(taskID, filename, dependencies))
  const failed: SidebarPdfExportFailure[] = []
  let succeededCount = 0

  for (const series of seriesRoots) {
    try {
      const task = await createExportTask(series.id)
      const snapshot = await waitForTaskCompletion(task.stream_url)
      if (snapshot && snapshot.status && snapshot.status !== 'succeeded') {
        throw new Error(snapshot.error_message || '导出失败')
      }
      await downloadTaskArtifact(task.task_id, `${sanitizeDownloadFilename(series.title)}.pdf`)
      succeededCount += 1
    } catch (error: unknown) {
      failed.push({
        id: series.id,
        title: series.title || '未命名系列',
        message: error instanceof Error ? error.message : '导出失败',
      })
    }
  }

  return {
    succeededCount,
    failed,
  }
}
