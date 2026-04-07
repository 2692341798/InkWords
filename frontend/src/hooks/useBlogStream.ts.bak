import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'

class StopStreamError extends Error {}

export const useBlogStream = () => {
  const store = useStreamStore()
  const { fetchBlogs } = useBlogStore()

  const analyzeGit = useCallback(async (gitUrl: string) => {
    // 基础拦截，防止用户输入非法的 git URL
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      alert('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    store.setAnalyzing(true)
    store.setAnalysisStep(-1)
    store.setAnalysisMessage('正在建立连接...')
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      
      await fetchEventSource('/api/v1/stream/analyze', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true, // Prevents fetch-event-source from aborting when tab is hidden
        body: JSON.stringify({ git_url: gitUrl }),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return; // everything's good
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              if (data.step !== undefined) {
                store.setAnalysisStep(data.step)
                store.setAnalysisMessage(data.message || '')
                if (data.data?.status && data.data.status.startsWith('chunk_')) {
                  store.setMapReduceProgress(data.data)
                } else if (data.step === 3 || data.step === 4) {
                  store.setMapReduceProgress(null) // clear when entering next stages
                }
              }
              if (data.data?.outline) {
                store.setSource('git', data.data.source_content, gitUrl)
                store.setOutline(data.data.outline)
                if (data.data.series_title) {
                  store.setSeriesTitle(data.data.series_title)
                }
              }
            } catch {
              // Ignore parse error
            }
          } else if (msg.event === 'error') {
            console.error('SSE Error:', msg.data)
            throw new StopStreamError(msg.data)
          } else if (msg.event === 'done') {
            store.setAnalyzing(false)
            store.setAnalysisStep(-1)
            throw new StopStreamError('done')
          }
        },
        onclose() {
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted') || e?.message?.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('SSE Connection Error:', err)
          store.setAnalyzing(false)
          store.setAnalysisStep(-1)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message !== 'done' && err.message !== 'aborted') {
          alert(`分析失败: ${err.message}`)
          throw err
        }
        return
      }
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) return
      console.error(err)
      store.setAnalyzing(false)
      store.setAnalysisStep(-1)
      alert(`分析出错: ${e?.message || '未知错误'}`)
      throw err 
    }
  }, [store])

  const parseFile = useCallback(async (file: File) => {
    store.setAnalyzing(true)
    store.setAnalysisStep(0)
    
    // For file, we simulate progress visually since it's usually fast
    const timer1 = setTimeout(() => store.setAnalysisStep(1), 300)
    const timer2 = setTimeout(() => store.setAnalysisStep(2), 800)
    const timer3 = setTimeout(() => store.setAnalysisStep(3), 1300)
    const timer4 = setTimeout(() => store.setAnalysisStep(4), 1800)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await fetch('/api/v1/project/parse', {
        method: 'POST',
        headers: { 
          'Authorization': localStorage.getItem('token') ? `Bearer ${localStorage.getItem('token')}` : ''
        },
        body: formData
      })
      
      const res = await response.json()
      if (res.code === 200 && res.data) {
        store.setSource('file', res.data.source_content)
        // file doesn't have an outline by default, but we can set an empty outline to trigger single generation UI
        store.setOutline([])
      } else {
        throw new Error(res.message || '文件解析失败')
      }
    } catch (err) {
      // Catch specific errors sent from the backend
      const errMsg = err instanceof Error ? err.message : '文件解析失败，请确保上传了有效的文件格式'
      alert(errMsg)
      throw err
    } finally {
      clearTimeout(timer1)
      clearTimeout(timer2)
      clearTimeout(timer3)
      clearTimeout(timer4)
      store.setAnalyzing(false)
      store.setAnalysisStep(-1)
    }
  }, [store])

  const generateSingle = useCallback(async (sourceContent: string) => {
    store.setGenerating(true)
    store.clearGeneratedContent()
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      
      await fetchEventSource('/api/v1/stream/generate', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify({
          source_content: sourceContent,
          source_type: 'file',
          outline: []
        }),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            store.appendGeneratedContent(msg.data)
          } else if (msg.event === 'error') {
            console.error('SSE Error:', msg.data)
            throw new StopStreamError(msg.data)
          } else if (msg.event === 'done') {
            store.setGenerating(false)
            store.reset() // clear generator state completely
            fetchBlogs() // Refresh to get the saved blog from DB
            throw new StopStreamError('done')
          }
        },
        onclose() {
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted') || e?.message?.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message !== 'done' && err.message !== 'aborted') {
          store.setGenerating(false)
          alert(`生成失败: ${err.message}`)
        }
        return
      }
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted')) return
      console.error(err)
      store.setGenerating(false)
      alert(`生成出错: ${e?.message || '未知错误'}`)
    }
  }, [store, fetchBlogs])

  const generateSeries = useCallback(async () => {
    if (!store.outline || !store.sourceContent) return

    store.setGenerating(true)
    store.clearGeneratedContent()
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      
      const remainingOutline = store.outline.filter(ch => store.chapterStatus[ch.sort] !== 'completed')
      
      await fetchEventSource('/api/v1/stream/generate', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify({
          source_content: store.sourceContent,
          outline: remainingOutline,
          source_type: store.sourceType || 'git',
          git_url: store.gitUrl || '',
          series_title: store.seriesTitle || '',
          parent_id: store.parentBlogId || ''
        }),
        async onopen(response) {
          if (response.ok && response.headers.get('content-type')?.startsWith('text/event-stream')) {
            return;
          }
          if (response.headers.get('content-type')?.includes('application/json')) {
            const data = await response.json();
            throw new StopStreamError(data.error || '请求失败');
          }
          const text = await response.text();
          throw new StopStreamError(text || `请求失败: ${response.status} ${response.statusText}`);
        },
        onmessage(msg) {
          if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              if (data.status === 'generating') {
                store.clearGeneratedContent()
                store.updateChapterStatus(data.chapter_sort, 'generating')
                if (data.parent_id && !store.parentBlogId) {
                  store.setParentBlogId(data.parent_id)
                }
              } else if (data.status === 'streaming') {
                store.appendGeneratedContent(data.content)
              } else if (data.status === 'completed') {
                store.updateChapterStatus(data.chapter_sort, 'completed')
                fetchBlogs() // Refresh to show the newly completed chapter in history
              }
            } catch {
              // Ignore parse error
            }
          } else if (msg.event === 'error') {
            console.error('SSE Error:', msg.data)
            throw new StopStreamError(msg.data)
          } else if (msg.event === 'done') {
            store.setGenerating(false)
            fetchBlogs() // Make sure we refresh to get the latest blogs
            throw new StopStreamError('done')
          }
        },
        onclose() {
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            throw err
          }
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted') || e?.message?.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('Generate SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message !== 'done' && err.message !== 'aborted') {
          alert(`生成失败: ${err.message}`)
        }
        return
      }
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted')) return
      console.error(err)
      store.setGenerating(false)
      alert(`生成出错: ${e?.message || '未知错误'}`)
    }
  }, [store, fetchBlogs])

  const stopAnalyzing = useCallback(() => {
    store.stopAllStreams()
  }, [store])

  const stopGenerating = useCallback(() => {
    store.stopAllStreams()
  }, [store])

  return {
    analyzeGit,
    parseFile,
    generateSingle,
    generateSeries,
    stopAnalyzing,
    stopGenerating
  }
}
