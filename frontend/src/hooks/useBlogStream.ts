import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'

class StopStreamError extends Error {}

export const useBlogStream = () => {
  const store = useStreamStore()
  const { fetchBlogs } = useBlogStore()

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
        throw new Error(res.message || 'Failed to parse file')
      }
    } catch (err) {
      console.error(err)
      throw err
    } finally {
      store.setAnalyzing(false)
    }
  }, [store])

  const generateSingle = useCallback(async (sourceContent: string) => {
    store.setGenerating(true)
    const ctrl = new AbortController()
    
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
            // Content is streaming, we can stay in loading state in Generator.tsx
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
        onerror(err) {
          if (err instanceof StopStreamError) {
            throw err
          }
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err) {
      if (err instanceof StopStreamError) return
      console.error(err)
      store.setGenerating(false)
    }
  }, [store, fetchBlogs])

  const generateSeries = useCallback(async () => {
    if (!store.outline || !store.sourceContent) return

    store.setGenerating(true)
    const ctrl = new AbortController()
    
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
        onerror(err) {
          if (err instanceof StopStreamError) {
            throw err
          }
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err) {
      if (err instanceof StopStreamError) return
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
