import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'

export const useProjectScanner = () => {
  const store = useStreamStore()

  const scanGit = useCallback(async (gitUrl: string) => {
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      alert('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
      throw new Error('invalid url')
    }

    store.setScanning(true)
    store.setAnalysisStep(0)
    store.setAnalysisMessage('正在扫描仓库目录与核心模块...')
    
    if (store.abortController) {
      store.abortController.abort()
    }
    const ctrl = new AbortController()
    store.setAbortController(ctrl)
    
    try {
      const token = localStorage.getItem('token')
      
      const response = await fetch('/api/v1/project/scan', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : ''
        },
        signal: ctrl.signal,
        body: JSON.stringify({ git_url: gitUrl }),
      })

      if (response.status === 401) {
        localStorage.removeItem('token')
        window.location.reload()
        throw new Error('登录已过期，请重新登录')
      }

      const data = await response.json()
      if (!response.ok || data.code !== 0) {
        throw new Error(data.message || data.error || '请求失败')
      }

      // 扫描成功
      store.setAnalysisStep(2)
      store.setAnalysisMessage('扫描完成，请选择要分析的模块')
      store.setSource('git', '', gitUrl)
      store.setModules(data.data.modules || [])
      store.setSelectedModules([])
      store.setScanning(false)
      
    } catch (err: any) {
      store.setScanning(false)
      if (err.name !== 'AbortError') {
        alert(err.message || '扫描失败')
      }
      throw err
    }
  }, [store])

  return { scanGit }
}