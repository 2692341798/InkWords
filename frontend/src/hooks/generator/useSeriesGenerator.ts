import { useCallback } from 'react'
import type { ScenarioMode } from '@/lib/scenarioMode'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
import {
  buildGenerationTaskRequest,
  createGenerationTask,
} from '@/services/generationTasks'
import type { Chapter } from '@/store/streamStore'
import { toast } from 'sonner'

class StopStreamError extends Error {}

interface SeriesGenerateRequestInput {
  sourceType: 'git' | 'file' | null
  gitUrl: string
  sourceContent: string
  seriesTitle: string
  outline: Chapter[] | null
  parentBlogId: string | null
  scenarioMode: ScenarioMode
  promptProfileKey?: string
  documentKind?: string
}

export function buildSeriesGenerateRequest(input: SeriesGenerateRequestInput) {
  return {
    source_type: input.sourceType,
    git_url: input.gitUrl,
    source_content: input.sourceContent,
    series_title: input.seriesTitle,
    outline: input.outline,
    parent_id: input.parentBlogId,
    scenario_mode: input.scenarioMode,
    prompt_profile_key: input.promptProfileKey,
    document_kind: input.documentKind,
  }
}

export function buildSingleGenerateRequest(
  content: string,
  scenarioMode: ScenarioMode,
  promptProfileKey?: string,
  documentKind?: string,
) {
  return {
    source_type: 'file' as const,
    source_content: content,
    outline: [],
    scenario_mode: scenarioMode,
    prompt_profile_key: promptProfileKey,
    document_kind: documentKind,
  }
}

type SeriesChunkStore = Pick<
  ReturnType<typeof useStreamStore.getState>,
  | 'appendChapterContent'
  | 'appendContent'
  | 'bufferContent'
  | 'clearChapterError'
  | 'setChapterUsage'
  | 'setChapterError'
  | 'setCurrentChapterTitle'
  | 'setProgress'
  | 'updateChapterPhase'
  | 'updateChapterStatus'
>

/**
 * Why: 系列生成的 SSE `chunk` 事件已经同时承载正文流和结构化进度，
 * 抽成纯函数后可以先用测试锁定事件映射，再让 Hook 复用同一套解析逻辑。
 */
export function handleSeriesChunkMessage(store: SeriesChunkStore, rawData: string) {
  const data = JSON.parse(rawData)
  const sort = Number(data.chapter_sort)

  if (storeProgressShouldClear()) {
    store.setProgress('')
  }

  if (data.title) {
    store.setCurrentChapterTitle(data.title)
  }

  if (data.status === 'progress') {
    store.setProgress(data.message)
    return
  }

  if (data.status === 'usage') {
    store.setChapterUsage(sort, {
      prompt_tokens: data.prompt_tokens,
      completion_tokens: data.completion_tokens,
      prompt_cache_hit_tokens: data.prompt_cache_hit_tokens,
      prompt_cache_miss_tokens: data.prompt_cache_miss_tokens,
    })
    return
  }

  if (
    data.status === 'understanding' ||
    data.status === 'drafting' ||
    data.status === 'reviewing' ||
    data.status === 'revising' ||
    data.status === 'streaming'
  ) {
    store.clearChapterError(sort)
    store.updateChapterStatus(sort, 'generating')
    store.updateChapterPhase(sort, data.status)
    if (data.status === 'streaming') {
      store.appendChapterContent(sort, data.content)
    }
    return
  }

  if (data.status === 'completed') {
    store.clearChapterError(sort)
    store.updateChapterStatus(sort, 'completed')
    store.updateChapterPhase(sort, 'completed')
    return
  }

  if (data.status === 'error') {
    if (typeof data.message === 'string' && data.message.trim()) {
      store.setChapterError(sort, data.message)
    }
    store.updateChapterStatus(sort, 'error')
    store.updateChapterPhase(sort, 'error')
    return
  }

  if (data.status === 'retrying') {
    store.clearChapterError(sort)
    store.updateChapterStatus(sort, 'pending')
    store.updateChapterPhase(sort, 'pending')
  }
}

function storeProgressShouldClear() {
  return useStreamStore.getState().progress === '准备生成环境...'
}

/**
 * Why: 生成 Hook 直接承接后端 SSE 事件，是系列创作流程的单一入口；
 * 在这里统一解析阶段事件，能避免 UI 组件承担协议细节。
 */
export const useSeriesGenerator = () => {
  const fetchBlogs = useBlogStore((state) => state.fetchBlogs)

  const finalizeGeneration = useCallback(async () => {
    const currentStore = useStreamStore.getState()
    currentStore.flushBufferedChapterContents()
    currentStore.flushBufferedContent()
    currentStore.setGenerating(false)
    currentStore.setProgress('生成完成')
    currentStore.setCurrentTaskId(null)

    await fetchBlogs()

    const { blogs, selectBlog } = useBlogStore.getState()
    const parentId = useStreamStore.getState().parentBlogId
    if (parentId) {
      const parentBlog = blogs.find((blog) => blog.id === parentId)
      if (parentBlog) {
        selectBlog(parentBlog)
      }
    }
  }, [fetchBlogs])

  const generateSeries = useCallback(async () => {
    const store = useStreamStore.getState()
    store.setContent('')
    store.setProgress('准备生成环境...')
    store.setGenerating(true)
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    store.setCurrentTaskId(null)
    
    try {
      const task = await createGenerationTask(
        buildGenerationTaskRequest(
          'generate_series',
          buildSeriesGenerateRequest({
            sourceType: store.sourceType,
            gitUrl: store.gitUrl,
            sourceContent: store.sourceContent,
            seriesTitle: store.seriesTitle,
            outline: store.outline,
            parentBlogId: store.parentBlogId,
            scenarioMode: store.scenarioMode,
            promptProfileKey: store.resolvedPromptProfile?.key,
            documentKind: store.resolvedPromptProfile?.documentKind,
          }),
        ),
      )
      useStreamStore.getState().setCurrentTaskId(task.task_id)

      await fetchEventSourceWithAuth(task.stream_url, {
        method: 'GET',
        signal: ctrl.signal,
        openWhenHidden: true,
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json()
            throw new StopStreamError(data.message || data.error || '请求失败');
          }
          const text = await response.text()
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`)
        },
        onmessage(msg) {
          const currentStore = useStreamStore.getState()
          if (msg.event === 'done') {
            void finalizeGeneration().then(() => {
              setTimeout(() => {
                useStreamStore.getState().reset()
              }, 2000)
            })
            return
          }
          
          if (msg.event === 'error') {
            throw new StopStreamError(msg.data)
          }

          if (msg.event === 'chapter') {
            currentStore.setCurrentChapterTitle(msg.data)
          } else if (msg.event === 'progress') {
            currentStore.setProgress(msg.data)
          } else if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              if (
                data.status === 'completed' ||
                data.status === 'error' ||
                data.status === 'retrying'
              ) {
                currentStore.flushBufferedChapterContents()
              }
              handleSeriesChunkMessage(
                {
                  ...currentStore,
                  appendChapterContent: currentStore.bufferChapterContent,
                },
                msg.data,
              )
            } catch {
              currentStore.bufferContent(msg.data)
            }
          }
        },
        onclose() {
          const currentStore = useStreamStore.getState()
          if (ctrl.signal.aborted) {
            currentStore.clearBufferedChapterContents()
            currentStore.clearBufferedContent()
          } else {
            currentStore.flushBufferedChapterContents()
            currentStore.flushBufferedContent()
          }
          currentStore.setGenerating(false)
          currentStore.setCurrentTaskId(null)
        },
        onerror(err) {
          const currentStore = useStreamStore.getState()
          if (ctrl.signal.aborted) {
            currentStore.clearBufferedChapterContents()
            currentStore.clearBufferedContent()
          } else {
            currentStore.flushBufferedChapterContents()
            currentStore.flushBufferedContent()
          }
          currentStore.setGenerating(false)
          currentStore.setCurrentTaskId(null)
          if (err instanceof StopStreamError) {
            toast.error(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      const currentStore = useStreamStore.getState()
      currentStore.setGenerating(false)
      currentStore.setCurrentTaskId(null)
      throw err
    }
  }, [finalizeGeneration])

  const generateSingle = useCallback(async (content: string) => {
    const store = useStreamStore.getState()
    store.setContent('')
    store.setProgress('准备生成环境...')
    store.setGenerating(true)
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    store.setCurrentTaskId(null)
    
    try {
      const task = await createGenerationTask(
        buildGenerationTaskRequest(
          'generate_single',
          buildSingleGenerateRequest(
            content,
            store.scenarioMode,
            store.resolvedPromptProfile?.key,
            store.resolvedPromptProfile?.documentKind,
          ),
        ),
      )
      useStreamStore.getState().setCurrentTaskId(task.task_id)

      await fetchEventSourceWithAuth(task.stream_url, {
        method: 'GET',
        signal: ctrl.signal,
        openWhenHidden: true,
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
          const currentStore = useStreamStore.getState()
          if (msg.event === 'done') {
            currentStore.flushBufferedContent()
            currentStore.setGenerating(false)
            currentStore.setProgress('生成完成')
            currentStore.setCurrentTaskId(null)
            void fetchBlogs()
            return
          }
          
          if (msg.event === 'error') {
            throw new StopStreamError(msg.data)
          }

          if (msg.event === 'progress') {
            currentStore.setProgress(msg.data)
          } else if (msg.event === 'chunk') {
            if (useStreamStore.getState().progress === '准备生成环境...') {
              currentStore.setProgress('')
            }
            currentStore.bufferContent(msg.data)
          }
        },
        onclose() {
          const currentStore = useStreamStore.getState()
          if (ctrl.signal.aborted) {
            currentStore.clearBufferedContent()
          } else {
            currentStore.flushBufferedContent()
          }
          currentStore.setGenerating(false)
          currentStore.setCurrentTaskId(null)
        },
        onerror(err) {
          const currentStore = useStreamStore.getState()
          if (ctrl.signal.aborted) {
            currentStore.clearBufferedContent()
          } else {
            currentStore.flushBufferedContent()
          }
          currentStore.setGenerating(false)
          currentStore.setCurrentTaskId(null)
          if (err instanceof StopStreamError) {
            toast.error(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      const currentStore = useStreamStore.getState()
      currentStore.setGenerating(false)
      currentStore.setCurrentTaskId(null)
      throw err
    }
  }, [fetchBlogs])

  return { generateSeries, generateSingle }
}
