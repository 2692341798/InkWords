import type { BlogNode } from '@/store/blogStore'

interface SidebarExportDependencies {
  fetchImpl?: typeof fetch
  getToken?: () => string | null
  downloadBlob?: (blob: Blob, filename: string) => void
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

async function readErrorMessage(response: Response | { json?: () => Promise<unknown> }, fallback: string) {
  const data = await response.json?.().catch(() => null)
  if (typeof data === 'object' && data !== null && 'message' in data && typeof data.message === 'string') {
    return data.message
  }

  return fallback
}

function sanitizeDownloadFilename(name: string) {
  return (name || 'series').replaceAll('/', '-').replaceAll('\\', '-').replaceAll(':', '：').trim()
}

function downloadBlobWithBrowser(blob: Blob, filename: string) {
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
  const fetchImpl = dependencies.fetchImpl ?? fetch
  const getToken = dependencies.getToken ?? (() => localStorage.getItem('token'))
  const downloadBlob = dependencies.downloadBlob ?? downloadBlobWithBrowser
  const token = getToken()
  const failed: SidebarPdfExportFailure[] = []
  let succeededCount = 0

  for (const series of seriesRoots) {
    try {
      const response = await fetchImpl(`/api/v1/blogs/${series.id}/export/pdf`, {
        headers: getAuthHeaders(token),
      })

      if (!response.ok) {
        throw new Error(await readErrorMessage(response, '导出失败'))
      }

      const blob = await response.blob()
      downloadBlob(blob, `${sanitizeDownloadFilename(series.title)}.pdf`)
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
