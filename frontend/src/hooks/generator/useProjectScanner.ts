import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'

class StopStreamError extends Error {}

export const useProjectScanner = () => {
  const store = useStreamStore()

  const scanGit = useCallback(async (gitUrl: string) => {
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      alert('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    store.setScanning(true)
    store.setAnalysisStep(-1)
    store.setAnalysisMessage('正在建立连接...')
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      
      await fetchEventSource('/api/v1/stream/scan', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        openWhenHidden: true,
        body: JSON.stringify({ git_url: gitUrl }),
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
            store.setScanning(false)
            return
          }
          
          if (msg.event === 'error') {
            throw new StopStreamError(msg.data)
          }

          if (msg.event === 'chunk') {
            try {
              const data = JSON.parse(msg.data)
              store.setAnalysisMessage(data.message)
              if (data.status === 'cloning') {
                store.setAnalysisStep(0)
              } else if (data.status === 'scanning') {
                store.setAnalysisStep(1)
              } else if (data.status === 'complete') {
                store.setAnalysisStep(2)
                const modules = JSON.parse(data.content)
                store.setSource('git', '', gitUrl)
                store.setModules(modules)
                store.setSelectedModules([])
                store.setScanning(false)
              }
            } catch (e) {
              console.error('Failed to parse scanning progress:', e)
            }
          }
        },
        onclose() {
          store.setScanning(false)
        },
        onerror(err) {
          store.setScanning(false)
          if (err instanceof StopStreamError) {
            alert(err.message)
            throw err
          }
          throw err
        }
      })
    } catch (err) {
      store.setScanning(false)
      throw err
    }
  }, [store])

  return { scanGit }
}