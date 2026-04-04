import { useCallback, useRef, useEffect } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'

class StopStreamError extends Error {}

export const useBlogStream = () => {
  const store = useStreamStore()
  const { fetchBlogs } = useBlogStore()
  const abortCtrlRef = useRef<AbortController | null>(null)

  // Cleanup pending streams when the component unmounts
  useEffect(() => {
    return () => {
      if (abortCtrlRef.current) {
        abortCtrlRef.current.abort()
      }
    }
  }, [])

  const analyzeGit = useCallback(async (gitUrl: string) => {
    store.setAnalyzing(true)
    try {
      const response = await fetch('/api/v1/project/analyze', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': localStorage.getItem('token') ? `Bearer ${localStorage.getItem('token')}` : ''
        },
        body: JSON.stringify({ git_url: gitUrl })
      })
      
      const res = await response.json()
      if (res.code === 200 && res.data) {
        store.setSource('git', res.data.source_content)
        store.setOutline(res.data.outline)
      } else {
        throw new Error(res.message || 'Failed to analyze project')
      }
    } catch (err) {
      console.error(err)
      // TODO: Handle error via Toast
      throw err
    } finally {
      store.setAnalyzing(false)
    }
  }, [store])

  const parseFile = useCallback(async (file: File) => {
    store.setAnalyzing(true)
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
      store.setAnalyzing(false)
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
            store.setOutline([]) // clear generator state
            store.setSource('file', '')
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
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) {
            throw new StopStreamError('aborted')
          }
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) return
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) return
      console.error(err)
      store.setGenerating(false)
    }
  }, [store, fetchBlogs])

  const generateSeries = useCallback(async () => {
    if (!store.outline || !store.sourceContent) return

    store.setGenerating(true)
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
                store.updateChapterStatus(data.chapter_sort, 'generating')
              } else if (data.status === 'completed') {
                store.updateChapterStatus(data.chapter_sort, 'completed')
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
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const e = err as any
          if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) {
            throw new StopStreamError('aborted')
          }
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err: unknown) {
      if (err instanceof StopStreamError) return
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const e = err as any
      if (e?.name === 'AbortError' || e?.message?.includes('AbortError')) return
      console.error(err)
      store.setGenerating(false)
    }
  }, [store, fetchBlogs])

  return {
    analyzeGit,
    parseFile,
    generateSingle,
    generateSeries
  }
}
