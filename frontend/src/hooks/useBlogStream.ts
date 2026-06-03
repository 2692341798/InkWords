import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useProjectScanner } from './generator/useProjectScanner'
import { useProjectAnalyzer } from './generator/useProjectAnalyzer'
import { useFileParser } from './generator/useFileParser'
import { useSeriesGenerator } from './generator/useSeriesGenerator'
import { cancelGenerationTask } from '@/services/generationTasks'

/**
 * Provides a single orchestration surface for the generator workflow so pages
 * can trigger scan, analyze, parse, generate, and stop actions without knowing
 * how each sub-hook coordinates with the shared stream store.
 */
export const useBlogStream = () => {
  const { scanGit } = useProjectScanner()
  const { analyzeGit } = useProjectAnalyzer()
  const { parseFile, analyzeParsedFile } = useFileParser()
  const { generateSeries, generateSingle } = useSeriesGenerator()

  const stopAnalyzing = useCallback(() => {
    const store = useStreamStore.getState()
    if (store.abortController) {
      store.abortController.abort()
      store.setAbortController(null)
    }
    store.setAnalyzing(false)
    store.setScanning(false)
    store.setAnalysisMessage('解析已取消')
  }, [])

  const stopGenerating = useCallback(() => {
    const store = useStreamStore.getState()
    const currentTaskId = store.currentTaskId
    if (currentTaskId) {
      void cancelGenerationTask(currentTaskId).catch(() => {
        // Why: 取消任务是“尽力而为”的后台动作，前端本地中止仍然应该立刻响应用户点击。
      })
      store.setCurrentTaskId(null)
    }
    if (store.abortController) {
      store.abortController.abort()
      store.setAbortController(null)
    }
    store.setGenerating(false)
    store.setProgress('已停止生成')
  }, [])

  return {
    scanGit,
    analyzeGit,
    parseFile,
    analyzeParsedFile,
    generateSeries,
    generateSingle,
    stopAnalyzing,
    stopGenerating
  }
}
