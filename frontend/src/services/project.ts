import { requestJson } from './apiClient'
import { apiRoutes } from './apiRoutes'

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

async function parseProjectFile(formData: FormData, signal?: AbortSignal): Promise<ParseProjectResponse> {
  return requestJson<ParseProjectResponse>(apiRoutes.parserService.parseProject, {
    method: 'POST',
    body: formData,
    signal,
    fallbackMessage: '文件解析失败',
  })
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

  return requestJson<CreateParseTaskResponse>(apiRoutes.coreApi.tasks.parse, {
    method: 'POST',
    json: body,
    fallbackMessage: '创建解析任务失败',
  })
}

async function getTaskSnapshot(taskID: string): Promise<TaskSnapshotResponse> {
  return requestJson<TaskSnapshotResponse>(apiRoutes.coreApi.tasks.byId(taskID), {
    method: 'GET',
    fallbackMessage: '查询解析任务失败',
  })
}

export const projectService = {
  createParseTask,
  getTaskSnapshot,
  parseProjectFile,
}
