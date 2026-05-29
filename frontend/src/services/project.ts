import { buildAuthHeaders } from './auth'

export interface ProjectArchiveSummary {
  total_files: number
  supported_files?: number
  kept_files: number
  duplicate_files: number
  ignored_files: number
  failed_files: number
  kept_paths?: string[]
}

export interface ParseProjectResponse {
  content?: string
  data?: {
    source_content?: string
    archive_summary?: ProjectArchiveSummary
  }
}

interface ParseProjectErrorResponse {
  error?: string
  message?: string
}

const getLocalStorage = () => {
  if (typeof window === 'undefined' && typeof globalThis.localStorage === 'undefined') {
    return null
  }
  return globalThis.localStorage ?? null
}

async function parseProjectFile(formData: FormData, signal?: AbortSignal): Promise<ParseProjectResponse> {
  const response = await fetch('/api/v1/project/parse', {
    method: 'POST',
    headers: buildAuthHeaders(),
    body: formData,
    signal,
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as
    | (ParseProjectResponse & ParseProjectErrorResponse)
    | null
  if (!response.ok) {
    throw new Error(payload?.error || payload?.message || '文件解析失败')
  }

  return payload ?? {}
}

export const projectService = {
  parseProjectFile,
}
