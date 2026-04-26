import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'

class StopStreamError extends Error {}

export const useSeriesGenerator = () => {
  const store = useStreamStore()
  const { fetchBlogs } = useBlogStore()

  const generateSeries = useCallback(async () => {
    store.setContent('')
    store.setProgress('')
    store.setGenerating(true)
    
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
          source_type: store.sourceType,
          git_url: store.gitUrl,
          source_content: store.sourceContent,
          series_title: store.seriesTitle,
          outline: store.outline,
          parent_id: store.parentBlogId
        }),
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
            store.setGenerating(false)
            store.setProgress('生成完成')
            fetchBlogs().then(() => {
              const { blogs, selectBlog } = useBlogStore.getState()
              const parentId = store.parentBlogId
              if (parentId) {
                const parentBlog = blogs.find(b => b.id === parentId)
                if (parentBlog) {
                  selectBlog(parentBlog)
                }
              }
            })
            
            // Auto close/transition after 2 seconds
            setTimeout(() => {
              store.reset()
            }, 2000)
            return
          }
          
          if (msg.event === 'error') {
            throw new StopStreamError(msg.data)
          }

          if (msg.event === 'chapter') {
            store.setCurrentChapterTitle(msg.data)
          } else if (msg.event === 'progress') {
            store.setProgress(msg.data)
          } else if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              const sort = data.chapter_sort
              
              if (data.status === 'generating') {
                store.updateChapterStatus(sort, 'generating')
                if (data.title) {
                  store.setCurrentChapterTitle(data.title)
                }
              } else if (data.status === 'streaming') {
                store.appendChapterContent(sort, data.content)
              } else if (data.status === 'completed') {
                store.updateChapterStatus(sort, 'completed')
              } else if (data.status === 'error') {
                store.updateChapterStatus(sort, 'error')
              } else if (data.status === 'retrying') {
                store.updateChapterStatus(sort, 'pending') // or maybe a special retrying state, but pending is fine
              }
            } catch (e) {
              // If it's not JSON, maybe it's just raw text (for single blog generation)
              store.appendContent(msg.data)
            }
          }
        },
        onclose() {
          store.setGenerating(false)
        },
        onerror(err) {
          store.setGenerating(false)
          if (err instanceof StopStreamError) {
            alert(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      store.setGenerating(false)
      throw err
    }
  }, [store, fetchBlogs])

  const generateSingle = useCallback(async (content: string) => {
    store.setContent('')
    store.setProgress('')
    store.setGenerating(true)
    
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
          source_type: 'file',
          source_content: content,
          outline: []
        }),
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
            store.setGenerating(false)
            store.setProgress('生成完成')
            fetchBlogs()
            return
          }
          
          if (msg.event === 'error') {
            throw new StopStreamError(msg.data)
          }

          if (msg.event === 'progress') {
            store.setProgress(msg.data)
          } else if (msg.event === 'chunk') {
            store.appendContent(msg.data)
          }
        },
        onclose() {
          store.setGenerating(false)
        },
        onerror(err) {
          store.setGenerating(false)
          if (err instanceof StopStreamError) {
            alert(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      store.setGenerating(false)
      throw err
    }
  }, [store, fetchBlogs])

  return { generateSeries, generateSingle }
}