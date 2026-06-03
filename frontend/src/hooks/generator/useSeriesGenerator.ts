import { useCallback } from 'react'
import type { ScenarioMode } from '@/lib/scenarioMode'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
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

export const useSeriesGenerator = () => {
  const fetchBlogs = useBlogStore((state) => state.fetchBlogs)

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
    
    try {
      await fetchEventSourceWithAuth('/api/v1/stream/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify(
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
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.message || data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          const currentStore = useStreamStore.getState()
          if (msg.event === 'done') {
            currentStore.flushBufferedChapterContents()
            currentStore.flushBufferedContent()
            currentStore.setGenerating(false)
            currentStore.setProgress('生成完成')
            fetchBlogs().then(() => {
              const { blogs, selectBlog } = useBlogStore.getState()
              const parentId = useStreamStore.getState().parentBlogId
              if (parentId) {
                const parentBlog = blogs.find(b => b.id === parentId)
                if (parentBlog) {
                  selectBlog(parentBlog)
                }
              }
            })
            
            // Auto close/transition after 2 seconds
            setTimeout(() => {
              useStreamStore.getState().reset()
            }, 2000)
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
            if (useStreamStore.getState().progress === '准备生成环境...') {
              currentStore.setProgress('')
            }
            try {
              const data = JSON.parse(msg.data)
              const sort = data.chapter_sort
              
              if (data.status === 'generating') {
                currentStore.clearChapterError(sort)
                currentStore.updateChapterStatus(sort, 'generating')
                if (data.title) {
                  currentStore.setCurrentChapterTitle(data.title)
                }
              } else if (data.status === 'progress') {
                currentStore.setProgress(data.message)
              } else if (data.status === 'streaming') {
                currentStore.bufferChapterContent(sort, data.content)
              } else if (data.status === 'completed') {
                currentStore.flushBufferedChapterContents()
                currentStore.clearChapterError(sort)
                currentStore.updateChapterStatus(sort, 'completed')
              } else if (data.status === 'error') {
                currentStore.flushBufferedChapterContents()
                if (typeof data.message === 'string' && data.message.trim()) {
                  currentStore.setChapterError(sort, data.message)
                }
                currentStore.updateChapterStatus(sort, 'error')
              } else if (data.status === 'retrying') {
                currentStore.flushBufferedChapterContents()
                currentStore.clearChapterError(sort)
                currentStore.updateChapterStatus(sort, 'pending')
              }
            } catch {
              // If it's not JSON, maybe it's just raw text (for single blog generation)
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
          if (err instanceof StopStreamError) {
            toast.error(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      useStreamStore.getState().setGenerating(false)
      throw err
    }
  }, [fetchBlogs])

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
    
    try {
      await fetchEventSourceWithAuth('/api/v1/stream/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify(
          buildSingleGenerateRequest(
            content,
            store.scenarioMode,
            store.resolvedPromptProfile?.key,
            store.resolvedPromptProfile?.documentKind,
          ),
        ),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.message || data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          const currentStore = useStreamStore.getState()
          if (msg.event === 'done') {
            currentStore.flushBufferedContent()
            currentStore.setGenerating(false)
            currentStore.setProgress('生成完成')
            fetchBlogs()
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
        },
        onerror(err) {
          const currentStore = useStreamStore.getState()
          if (ctrl.signal.aborted) {
            currentStore.clearBufferedContent()
          } else {
            currentStore.flushBufferedContent()
          }
          currentStore.setGenerating(false)
          if (err instanceof StopStreamError) {
            toast.error(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      useStreamStore.getState().setGenerating(false)
      throw err
    }
  }, [fetchBlogs])

  return { generateSeries, generateSingle }
}
