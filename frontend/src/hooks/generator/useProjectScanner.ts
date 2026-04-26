import { useCallback } from 'react'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import { useStreamStore } from '@/store/streamStore'

export const useProjectScanner = () => {
  const store = useStreamStore()

  const scanGit = useCallback(async (gitUrl: string) => {
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      alert('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    // Clear previous state before starting a new scan
    store.setModules(null)
    store.setSelectedModules([])
    store.setOutline(null)
    store.setParentBlogId(null)
    
    store.setScanning(true)
    // Only use analysis steps if not already analyzing
    if (!store.isAnalyzing) {
      store.setAnalysisStep(0)
      store.setAnalysisMessage('正在建立连接...')
      store.clearAnalysisHistory?.()
      store.appendAnalysisHistory?.({ message: '正在建立连接...', status: 'scanning' })
    }
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      let modulesResult: any = null;
      
      await fetchEventSource('/api/v1/stream/scan', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        body: JSON.stringify({ git_url: gitUrl }),
        async onopen(response) {
          if (response.status === 401) {
            localStorage.removeItem('token')
            window.location.reload()
            throw new Error('登录已过期，请重新登录')
          }
          if (!response.ok) {
            throw new Error('请求失败')
          }
        },
        onmessage(msg) {
          if (msg.event === 'error') {
            throw new Error(msg.data)
          }
          if (msg.event === 'progress') {
            store.setAnalysisMessage(msg.data)
            store.appendAnalysisHistory?.({ message: msg.data, status: 'scanning' })
          }
          if (msg.event === 'result') {
            modulesResult = JSON.parse(msg.data)
          }
          if (msg.event === 'done') {
            // Stream finished
          }
        },
        onclose() {
          // Do nothing
        },
        onerror(err) {
          throw err
        }
      })

      if (modulesResult) {
        // 扫描成功
        if (!store.isAnalyzing) {
          store.setAnalysisStep(2)
          store.setAnalysisMessage('扫描完成，请选择要分析的模块')
        }
        store.setSource('git', '', gitUrl)
        store.setModules(modulesResult.data?.modules || modulesResult.modules || modulesResult || [])
        store.setSelectedModules([])
        store.setScanning(false)
      } else {
        throw new Error('未收到扫描结果')
      }
      
    } catch (err: unknown) {
      store.setScanning(false)
      const error = err as Error
      if (error.name !== 'AbortError') {
        alert(error.message || '扫描失败')
      }
      throw error
    }
  }, [store])

  return { scanGit }
}