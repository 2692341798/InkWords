import { useCallback } from 'react'
import type { ScenarioMode } from '@/lib/scenarioMode'
import { useStreamStore } from '@/store/streamStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
import { apiRoutes } from '@/services/apiRoutes'
import { toast } from 'sonner'

class StopStreamError extends Error {}

export function buildAnalyzeGitRequest(
  gitUrl: string,
  selectedModules: string[],
  scenarioMode: ScenarioMode,
) {
  return {
    git_url: gitUrl,
    selected_modules: selectedModules,
    scenario_mode: scenarioMode,
  }
}

export const useProjectAnalyzer = () => {
  const analyzeGit = useCallback(async (gitUrl: string, selectedModules: string[]) => {
    const store = useStreamStore.getState()
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      toast.error('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    store.setAnalyzing(true)
    store.setAnalysisStep(-1)
    store.clearAnalysisHistory()
    store.setAnalysisMessage('正在建立连接...')
    store.appendAnalysisHistory({ message: '正在建立连接...' })
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    let receivedComplete = false
    
    try {
      await fetchEventSourceWithAuth(apiRoutes.llmStream.analyze, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify(
          buildAnalyzeGitRequest(gitUrl, selectedModules, store.scenarioMode),
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
              if (data.status === 'cloning') {
                store.setAnalysisStep(0)
              } else if (data.status === 'scanning') {
                store.setAnalysisStep(1)
              } else if (data.status === 'analyzing') {
                store.setAnalysisStep(2)
              } else if (data.status === 'outline') {
                store.setAnalysisStep(3)
              } else if (data.status === 'complete') {
                // data.content might be an object directly or a JSON string.
                let outlineResult = data.content;
                if (typeof data.content === 'string') {
                  outlineResult = JSON.parse(data.content);
                }
                if (!outlineResult || typeof outlineResult !== 'object') {
                  throw new Error('大纲响应缺少内容，请重试')
                }
                const outline = outlineResult.outline || outlineResult.chapters
                if (!Array.isArray(outline) || outline.length === 0) {
                  throw new Error('大纲响应缺少章节，请重试')
                }

                receivedComplete = true
                store.setAnalysisStep(4)
                
                store.setSource('git', outlineResult.source_content || '', gitUrl)
                store.setSeriesTitle(outlineResult.series_title || '')
                store.setOutline(outline)
                if (outlineResult.parent_id || data.parent_id) {
                  store.setParentBlogId(outlineResult.parent_id || data.parent_id)
                } else {
                  store.setParentBlogId(null)
                }
                store.setAnalyzing(false)
                store.setAnalysisMessage('')
              }
            } catch (e) {
              console.error('Failed to parse analysis progress:', e)
              throw e
            }
          }
        },
        onclose() {
          store.setAnalyzing(false)
          if (!receivedComplete && !ctrl.signal.aborted) {
            throw new StopStreamError('项目分析连接提前结束，请重试')
          }
        },
        onerror(err) {
          store.setAnalyzing(false)
          if (err instanceof StopStreamError) {
            toast.error(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      store.setAnalyzing(false)
      throw err
    }
  }, [])

  return { analyzeGit }
}
