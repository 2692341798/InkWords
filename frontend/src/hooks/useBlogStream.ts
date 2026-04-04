import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'

export const useBlogStream = () => {
  const store = useStreamStore()

  const analyzeGit = useCallback(async (gitUrl: string) => {
    store.setAnalyzing(true)
    try {
      const response = await fetch('/api/v1/project/analyze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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
    } finally {
      store.setAnalyzing(false)
    }
  }, [store])

  const generateSeries = useCallback(async () => {
    if (!store.outline || !store.sourceContent) return

    store.setGenerating(true)
    try {
      await fetchEventSource('/api/v1/stream/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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
          } else if (msg.event === 'done') {
            store.setGenerating(false)
          }
        },
        onerror(err) {
          console.error('SSE Connection Error:', err)
          store.setGenerating(false)
          throw err
        }
      })
    } catch (err) {
      console.error(err)
      store.setGenerating(false)
    }
  }, [store])

  return {
    analyzeGit,
    generateSeries
  }
}
