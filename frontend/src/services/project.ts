import { buildAuthHeaders } from './auth'

interface ProjectArchiveSummary {
  total_files: number
  supported_files?: number
  kept_files: number
  duplicate_files: number
  ignored_files: number
  failed_files: number
  kept_paths?: string[]
}

interface ParseProjectResponse {
  content?: string
  data?: {
    source_content?: string
    archive_summary?: ProjectArchiveSummary
  }
}

interface CreateParseTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

interface TaskSnapshotResponse {
  id: string
  status: string
  result?: {
    source_content?: string
    archive_summary?: ProjectArchiveSummary
  }
  error_message?: string
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

const encodeFileToBase64 = async (file: File): Promise<string> => {
  const buffer = await file.arrayBuffer()
  let binary = ''
  const bytes = new Uint8Array(buffer)
  const chunkSize = 0x8000
  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(index, index + chunkSize))
  }
  return btoa(binary)
}

const buildParseTaskKind = (filename: string) =>
  filename.toLowerCase().endsWith('.zip') ? 'parse_archive' : 'parse_file'

async function createParseTask(file: File): Promise<CreateParseTaskResponse> {
  const body = {
    kind: buildParseTaskKind(file.name),
    payload: {
      filename: file.name,
      content_base64: await encodeFileToBase64(file),
    },
    idempotency_key: `parse:${file.name}:${file.size}:${file.lastModified}`,
  }

  const response = await fetch('/api/v1/tasks/parse', {
    method: 'POST',
    headers: buildAuthHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(body),
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as
    | (CreateParseTaskResponse & ParseProjectErrorResponse)
    | null
  if (!response.ok) {
    throw new Error(payload?.error || payload?.message || '创建解析任务失败')
  }

  return payload as CreateParseTaskResponse
}

async function getTaskSnapshot(taskID: string): Promise<TaskSnapshotResponse> {
  const response = await fetch(`/api/v1/tasks/${taskID}`, {
    method: 'GET',
    headers: buildAuthHeaders(),
  })

  if (response.status === 401) {
    getLocalStorage()?.removeItem('token')
    throw new Error('登录已过期，请重新登录')
  }

  const payload = (await response.json().catch(() => null)) as
    | (TaskSnapshotResponse & ParseProjectErrorResponse)
    | null
  if (!response.ok) {
    throw new Error(payload?.error || payload?.message || '查询解析任务失败')
  }

  return payload as TaskSnapshotResponse
}

export const projectService = {
  createParseTask,
  getTaskSnapshot,
  parseProjectFile,
}
