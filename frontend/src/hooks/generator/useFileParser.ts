import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'

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
      const token = localStorage.getItem('token')
      const formData = new FormData()
      formData.append('file', file)

      const uploadRes = await fetch('/api/v1/project/parse', {
        method: 'POST',
        headers: {
          'Authorization': token ? `Bearer ${token}` : ''
        },
        body: formData,
        signal: ctrl.signal
      })

      if (uploadRes.status === 401) {
        localStorage.removeItem('token')
        window.location.reload()
        throw new Error('登录已过期，请重新登录')
      }

      if (!uploadRes.ok) {
        const errorData = await uploadRes.json()
        throw new Error(errorData.error || '文件解析失败')
      }

      const data = await uploadRes.json()
      const content = data.content
      store.setSourceContent(content)
      
      store.setAnalysisStep(1)
      store.setAnalysisMessage('文件解析成功，正在生成大纲...')

      await fetchEventSource('/api/v1/stream/analyze', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify({ source_content: content }),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            if (response.status === 401) {
              localStorage.removeItem('token');
              window.location.reload();
              throw new StopStreamError('登录已过期，请重新登录');
            }
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