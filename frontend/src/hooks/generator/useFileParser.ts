import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import type { Chapter, ResolvedPromptProfile } from '@/store/streamStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
import { projectService } from '@/services/project'
import { toast } from 'sonner'
import {
  type ArchiveSummary,
  buildLargeFileAnalysisHint,
  extractArchiveSummary,
  extractParsedFileContent,
  formatArchiveSummaryMessage,
  type ParseFileResponse,
} from './fileParserUtils'
import { buildCurrentFileAnalyzeRequest } from './fileAnalyzeRequest'

class StopStreamError extends Error {}

const ASYNC_PARSE_FILE_SIZE_THRESHOLD_BYTES = 50 * 1024 * 1024

interface ParseUploadedFileStore {
  abortController: AbortController | null
  setAnalyzing: (status: boolean) => void
  setAnalysisStep: (step: number) => void
  clearAnalysisHistory: () => void
  setAnalysisMessage: (message: string) => void
  appendAnalysisHistory: (item: { message: string; status?: string }) => void
  setSource: (type: 'git' | 'file', content: string, gitUrl?: string) => void
  setOutline: (outline: Chapter[] | null) => void
  setAbortController: (ctrl: AbortController | null) => void
  setSourceContent: (content: string) => void
  setResolvedPromptProfile: (
    profile: ResolvedPromptProfile | null,
    status?: 'idle' | 'classifying' | 'resolved' | 'fallback',
  ) => void
}

interface ParseUploadedFileInput {
  file: File
  store: ParseUploadedFileStore
  parseProjectFile: (formData: FormData, signal?: AbortSignal) => Promise<ParseFileResponse>
  createParseTask?: (file: File) => Promise<{ task_id: string; stream_url: string }>
  getTaskSnapshot?: (taskID: string) => Promise<{
    status: string
    result?: {
      source_content?: string
      archive_summary?: ArchiveSummary
    }
    error_message?: string
  }>
}

/**
 * Why: 文件上传成功后，界面需要先停留在“配置解析”阶段，让用户选择创作场景，
 * 不能立即串联到生成大纲的流式分析，否则会跳过 step 2。
 */
export async function parseUploadedFile({
  file,
  store,
  parseProjectFile,
  createParseTask,
  getTaskSnapshot,
}: ParseUploadedFileInput) {
  store.setAnalyzing(true)
  store.setAnalysisStep(0)
  store.clearAnalysisHistory()
  store.setAnalysisMessage('正在上传并解析文件...')
  store.appendAnalysisHistory({ message: '正在上传并解析文件...' })
  store.setSource('file', '')
  store.setOutline([])
  store.setResolvedPromptProfile(null)

  if (store.abortController) {
    store.abortController.abort()
  }
  const ctrl = new AbortController()
  store.setAbortController(ctrl)

  try {
    const data =
      shouldUseParseTask(file) && createParseTask && getTaskSnapshot
        ? await parseFileViaTask({
            file,
            signal: ctrl.signal,
            createParseTask,
            getTaskSnapshot,
          })
        : await parseFileSynchronously({
            file,
            signal: ctrl.signal,
            parseProjectFile,
          })
    const content = extractParsedFileContent(data)
    const archiveSummary = extractArchiveSummary(data)
    if (!content) {
      throw new Error('文件解析成功但未返回可用内容')
    }

    store.setSourceContent(content)

    if (archiveSummary) {
      store.appendAnalysisHistory({
        message: formatArchiveSummaryMessage(archiveSummary),
        status: 'parsed',
      })
    }

    const largeFileHint = buildLargeFileAnalysisHint(content)
    if (largeFileHint) {
      store.appendAnalysisHistory({
        message: largeFileHint,
        status: 'parsed',
      })
    }

    store.setAnalysisStep(1)
    store.setAnalysisMessage('文件解析完成，请选择创作场景')
    store.setAnalyzing(false)

    return content
  } catch (err) {
    store.setAnalyzing(false)
    const errMsg = err instanceof Error ? err.message : '文件解析失败'
    if (errMsg !== 'AbortError' && errMsg !== 'The user aborted a request.') {
      toast.error(errMsg)
    }
    throw err
  }
}

const shouldUseParseTask = (file: File) =>
  file.name.toLowerCase().endsWith('.zip') || file.size > ASYNC_PARSE_FILE_SIZE_THRESHOLD_BYTES

async function parseFileSynchronously({
  file,
  signal,
  parseProjectFile,
}: {
  file: File
  signal: AbortSignal
  parseProjectFile: (formData: FormData, signal?: AbortSignal) => Promise<ParseFileResponse>
}) {
  const formData = new FormData()
  formData.append('file', file)
  return parseProjectFile(formData, signal)
}

async function parseFileViaTask({
  file,
  signal,
  createParseTask,
  getTaskSnapshot,
}: {
  file: File
  signal: AbortSignal
  createParseTask: (file: File) => Promise<{ task_id: string; stream_url: string }>
  getTaskSnapshot: (taskID: string) => Promise<{
    status: string
    result?: {
      source_content?: string
      archive_summary?: ArchiveSummary
    }
    error_message?: string
  }>
}) {
  const task = await createParseTask(file)
  await fetchEventSourceWithAuth(task.stream_url, {
    method: 'GET',
    signal,
    openWhenHidden: true,
    async onopen(response) {
      if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
        return
      }
      if (response.headers.get('content-type')?.includes('application/json')) {
        const data = await response.json()
        throw new StopStreamError(data.message || data.error || '解析任务请求失败')
      }
      const text = await response.text()
      throw new StopStreamError(text || `解析任务请求失败: ${response.status} ${response.statusText}`)
    },
    onmessage(msg) {
      if (msg.event === 'error') {
        throw new StopStreamError(msg.data || '解析任务执行失败')
      }
    },
  })

  const snapshot = await getTaskSnapshot(task.task_id)
  if (snapshot.status !== 'succeeded' || !snapshot.result) {
    throw new Error(snapshot.error_message || '解析任务未成功完成')
  }

  return {
    data: snapshot.result,
  }
}

export async function analyzeParsedFileContent(sourceContent: string) {
  const store = useStreamStore.getState()
  const ctrl = store.abortController ?? new AbortController()
  if (!store.abortController) {
    store.setAbortController(ctrl)
  }

  store.setAnalyzing(true)
  store.setAnalysisMessage('正在根据创作场景生成大纲...')

  await fetchEventSourceWithAuth('/api/v1/stream/analyze', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    signal: ctrl.signal,
    openWhenHidden: true,
    body: JSON.stringify(buildCurrentFileAnalyzeRequest(sourceContent)),
    async onopen(response) {
      if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
        return
      }
      if (response.headers.get('content-type')?.includes('application/json')) {
        const data = await response.json()
        throw new StopStreamError(data.message || data.error || '请求失败')
      }
      const text = await response.text()
      throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
    },
    onmessage(msg) {
      if (msg.event === 'done') {
        store.setAnalyzing(false)
        return
      }

      if (msg.event === 'error') {
        throw new StopStreamError(msg.data)
      }

      if (msg.event === 'chunk') {
        try {
          const data = JSON.parse(msg.data)
          store.setAnalysisMessage(data.message)
          store.appendAnalysisHistory({ message: data.message, status: data.status })
          if (data.status === 'analyzing') {
            store.setAnalysisStep(2)
            if (typeof data.message === 'string' && data.message.includes('识别')) {
              store.setResolvedPromptProfile(null, 'classifying')
            }
          } else if (data.status === 'outline') {
            store.setAnalysisStep(3)
          } else if (data.status === 'complete') {
            store.setAnalysisStep(4)
            let outlineResult = data.content
            if (typeof data.content === 'string') {
              outlineResult = JSON.parse(data.content)
            }
            store.setSource('file', outlineResult.source_content || outlineResult.series_title || '')
            const resolvedProfile = outlineResult.resolved_prompt_profile
            if (resolvedProfile) {
              store.setResolvedPromptProfile(
                {
                  key: resolvedProfile.key,
                  displayName: resolvedProfile.display_name,
                  documentKind: resolvedProfile.document_kind,
                  reason: resolvedProfile.reason,
                },
                resolvedProfile.reason?.includes('回退') ? 'fallback' : 'resolved',
              )
            } else {
              store.setResolvedPromptProfile(null)
            }
            store.setSeriesTitle(outlineResult.series_title || '')
            store.setOutline(outlineResult.outline || outlineResult.chapters)
            store.setAnalyzing(false)
            store.setAnalysisMessage('')
          }
        } catch (e) {
          console.error('Failed to parse analysis progress:', e)
        }
      }
    },
    onclose() {
      store.setAnalyzing(false)
    },
    onerror(err) {
      store.setAnalyzing(false)
      if (err instanceof StopStreamError) {
        toast.error(err.message)
        throw err
      }
      throw err
    },
  })
}

export const useFileParser = () => {
  const parseFile = useCallback(async (file: File) => {
    return parseUploadedFile({
      file,
      store: useStreamStore.getState(),
      parseProjectFile: projectService.parseProjectFile,
      createParseTask: projectService.createParseTask,
      getTaskSnapshot: projectService.getTaskSnapshot,
    })
  }, [])

  const analyzeParsedFile = useCallback(async () => {
    const sourceContent = useStreamStore.getState().sourceContent
    if (!sourceContent.trim()) {
      throw new Error('缺少可分析的文件内容')
    }
    await analyzeParsedFileContent(sourceContent)
  }, [])

  return { parseFile, analyzeParsedFile }
}
