import { useCallback } from 'react'
import type { ScenarioMode } from '@/lib/scenarioMode'
import { useStreamStore } from '@/store/streamStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
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
    
    try {
      await fetchEventSourceWithAuth('/api/v1/stream/analyze', {
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
                store.setAnalysisStep(4)
                
                // data.content might be an object directly now due to the backend change,
                // or it might still be a JSON string. Let's handle both.
                let outlineResult = data.content;
                if (typeof data.content === 'string') {
                  outlineResult = JSON.parse(data.content);
                }
                
                store.setSource('git', outlineResult.source_content || '', gitUrl)
                store.setSeriesTitle(outlineResult.series_title || '')
                store.setOutline(outlineResult.outline || outlineResult.chapters)
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
        }
      })
    } catch (err) {
      store.setAnalyzing(false)
      throw err
    }
  }, [])

  return { analyzeGit }
}
