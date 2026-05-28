import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
import { projectService } from '@/services/project'
import {
  extractArchiveSummary,
  extractParsedFileContent,
  formatArchiveSummaryMessage,
} from './fileParserUtils'
import { buildCurrentFileAnalyzeRequest } from './fileAnalyzeRequest'

class StopStreamError extends Error {}

export const useFileParser = () => {
  const store = useStreamStore()

  const parseFile = useCallback(async (file: File) => {
    store.setAnalyzing(true)
    store.setAnalysisStep(0)
    store.clearAnalysisHistory()
    store.setAnalysisMessage('正在上传并解析文件...')
    store.appendAnalysisHistory({ message: '正在上传并解析文件...' })
    store.setSource('file', file.name)
    store.setOutline([])
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const data = await projectService.parseProjectFile(formData, ctrl.signal)
      const content = extractParsedFileContent(data)
      const archiveSummary = extractArchiveSummary(data)
      if (!content) {
        throw new Error('文件解析成功但未返回可用内容')
      }
      store.setSourceContent(content)

      // Why: ZIP 解析摘要只在上传完成时返回一次，落到分析历史里可以复用现有状态面板，
      // 避免为首版额外引入新的展示组件和状态分支。
      if (archiveSummary) {
        store.appendAnalysisHistory({
          message: formatArchiveSummaryMessage(archiveSummary),
          status: 'parsed',
        })
      }
      
      store.setAnalysisStep(1)
      store.setAnalysisMessage('文件解析成功，正在生成大纲...')

      await fetchEventSourceWithAuth('/api/v1/stream/analyze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: ctrl.signal,
        openWhenHidden: true,
        // Why: 显式声明 file 来源，避免后端将文档上传误判为 git 解析链路。
        body: JSON.stringify(buildCurrentFileAnalyzeRequest(content)),
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
              if (data.status === 'analyzing') {
                store.setAnalysisStep(2)
              } else if (data.status === 'outline') {
                store.setAnalysisStep(3)
              } else if (data.status === 'complete') {
                store.setAnalysisStep(4)
                let outlineResult = data.content;
                if (typeof data.content === 'string') {
                  outlineResult = JSON.parse(data.content);
                }
                store.setSource('file', outlineResult.source_content || outlineResult.series_title || file.name)
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
            alert(err.message)
            throw err
          }
          throw err
        }
      })

    } catch (err) {
      store.setAnalyzing(false)
      const errMsg = err instanceof Error ? err.message : '文件解析失败'
      if (errMsg !== 'AbortError' && errMsg !== 'The user aborted a request.') {
        alert(errMsg)
      }
      throw err
    }
  }, [store])

  return { parseFile }
}
