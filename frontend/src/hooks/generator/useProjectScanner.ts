import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import type { ModuleCard } from '@/store/streamStore'
import { fetchEventSourceWithAuth } from '@/services/sse'
import { toast } from 'sonner'

export const useProjectScanner = () => {
  const normalizeModules = (input: unknown): ModuleCard[] => {
    const isModuleCard = (value: unknown): value is ModuleCard => {
      if (!value || typeof value !== 'object') return false
      const obj = value as Record<string, unknown>
      return typeof obj.path === 'string' && typeof obj.name === 'string' && typeof obj.description === 'string'
    }

    const pickFromArray = (value: unknown): ModuleCard[] => {
      if (!Array.isArray(value)) return []
      return value.filter(isModuleCard)
    }

    if (!input) return []
    if (Array.isArray(input)) return pickFromArray(input)
    if (typeof input === 'object') {
      const obj = input as Record<string, unknown>
      const data = obj.data
      if (data && typeof data === 'object') {
        const modules = (data as Record<string, unknown>).modules
        const picked = pickFromArray(modules)
        if (picked.length > 0) return picked
      }
      const picked = pickFromArray(obj.modules)
      if (picked.length > 0) return picked
    }
    return []
  }

  const scanGit = useCallback(async (gitUrl: string) => {
    const store = useStreamStore.getState()
    if (!gitUrl.startsWith('http://') && !gitUrl.startsWith('https://') && !gitUrl.startsWith('git@') && !gitUrl.startsWith('file://')) {
      toast.error('请输入有效的 Git 仓库链接 (以 http://, https://, git@ 或 file:// 开头)')
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
      let modulesResult: unknown = null
      
      await fetchEventSourceWithAuth('/api/v1/stream/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        signal: ctrl.signal,
        body: JSON.stringify({ git_url: gitUrl }),
        async onopen(response) {
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
        store.setModules(normalizeModules(modulesResult))
        store.setSelectedModules([])
        store.setScanning(false)
      } else {
        throw new Error('未收到扫描结果')
      }
      
    } catch (err: unknown) {
      store.setScanning(false)
      const error = err as Error
      if (error.name !== 'AbortError') {
        toast.error(error.message || '扫描失败')
      }
      throw error
    }
  }, [])

  return { scanGit }
}
