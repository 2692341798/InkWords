import { useCallback, useRef, useEffect, useState } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'

class StopStreamError extends Error {}

export const useBlogStream = () => {
  const store = useStreamStore()
  const { fetchBlogs } = useBlogStore()
  const abortCtrlRef = useRef<AbortController | null>(null)

  const [analysisStep, setAnalysisStep] = useState<number>(-1)
  const [analysisMessage, setAnalysisMessage] = useState<string>('')

  // Cleanup pending streams when the component unmounts
  useEffect(() => {
    return () => {
      if (abortCtrlRef.current) {
        abortCtrlRef.current.abort()
      }
    }
  }, [])

  const analyzeGit = useCallback(async (gitUrl: string) => {
    // 基础拦截，防止用户输入非法的 git URL
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      alert('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    store.setAnalyzing(true)
    setAnalysisStep(-1)
    setAnalysisMessage('正在建立连接...')
    
    if (abortCtrlRef.current) {
      abortCtrlRef.current.abort()
    }
    const ctrl = new AbortController()
    abortCtrlRef.current = ctrl
    
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
        onmessage(msg) {
          if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              if (data.step !== undefined) {
                setAnalysisStep(data.step)
                setAnalysisMessage(data.message || '')
                if (data.data?.status && data.data.status.startsWith('chunk_')) {
                  store.setMapReduceProgress(data.data)
                } else if (data.step === 3 || data.step === 4) {
                  store.setMapReduceProgress(null) // clear when entering next stages
                }
              }
              if (data.data?.outline) {
                store.setSource('git', data.data.source_content)
                store.setOutline(data.data.outline)
              }
            } catch {
              // Ignore parse error
            }
          } else if (msg.event === 'error') {
            console.error('SSE Error:', msg.data)
            throw new StopStreamError(msg.data)
          } else if (msg.event === 'done') {
            store.setAnalyzing(false)
            setAnalysisStep(-1)
            throw new StopStreamError('done')
          }
        },
        onclose() {
          throw new StopStreamError('closed by server')
        },
        onerror(err: unknown) {
          if (err instanceof StopStreamError) {
            // 如果是我们自己 throw 的 StopStreamError，不要向外抛出（特别是 aborted 时）
            // fetchEventSource 如果遇到内部的 throw 会停止，并且它可能会打印一个默认错误
            // 但是我们不想它变成一个 Uncaught Error。由于 fetchEventSource 内部逻辑，
            // 只要我们抛出任何 Error 都会触发中止连接，但最好是被我们外层 catch 捕获。
            throw err
          }
          // 拦截浏览器抛出的 AbortError，将其转换为我们的内部错误
          if (err instanceof DOMException && err.name === 'AbortError') {
            throw new StopStreamError('aborted')
          }
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted') || e?.message?.includes('Failed to fetch')) {
            throw new StopStreamError('aborted')
          }
          console.error('SSE Connection Error:', err)
          store.setAnalyzing(false)
          setAnalysisStep(-1)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) {
        if (err.message !== 'done' && err.message !== 'aborted') {
          alert(`分析失败: ${err.message}`)
          throw err
        }
        return // 正常结束或手动中止，不往外抛异常，不触发控制台红色 Uncaught
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const e = err as any
      // AbortError 经常在网络请求被主动中止时由浏览器抛出
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) return
      console.error(err)
      store.setAnalyzing(false)
      setAnalysisStep(-1)
      alert(`分析出错: ${e?.message || '未知错误'}`)
      throw err // Rethrow to let the component know it failed
    }
  }, [store])

  const parseFile = useCallback(async (file: File) => {
    store.setAnalyzing(true)
    setAnalysisStep(0)
    
    // For file, we simulate progress visually since it's usually fast
    const timer1 = setTimeout(() => setAnalysisStep(1), 300)
    const timer2 = setTimeout(() => setAnalysisStep(2), 800)
    const timer3 = setTimeout(() => setAnalysisStep(3), 1300)
    const timer4 = setTimeout(() => setAnalysisStep(4), 1800)

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
      setAnalysisStep(-1)
    }
  }, [store])

  const generateSingle = useCallback(async (sourceContent: string) => {
    store.setGenerating(true)
    store.clearGeneratedContent()
    if (abortCtrlRef.current) {
      abortCtrlRef.current.abort()
    }
    const ctrl = new AbortController()
    abortCtrlRef.current = ctrl
    
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
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
          alert(`生成失败: ${err.message}`)
        }
        return
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
    if (abortCtrlRef.current) {
      abortCtrlRef.current.abort()
    }
    const ctrl = new AbortController()
    abortCtrlRef.current = ctrl
    
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
          source_content: store.sourceContent,
          outline: store.outline,
          source_type: store.sourceType || 'git'
        }),
        onmessage(msg) {
          if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              if (data.status === 'generating') {
                store.clearGeneratedContent()
                store.updateChapterStatus(data.chapter_sort, 'generating')
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
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError') || e?.message?.includes('aborted')) return
      console.error(err)
      store.setGenerating(false)
      alert(`生成出错: ${e?.message || '未知错误'}`)
    }
  }, [store, fetchBlogs])

  return {
    analysisStep,
    analysisMessage,
    analyzeGit,
    parseFile,
    generateSingle,
    generateSeries,
    abortCtrlRef
  }
}
