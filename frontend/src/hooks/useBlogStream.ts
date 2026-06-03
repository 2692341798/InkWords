import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useProjectScanner } from './generator/useProjectScanner'
import { useProjectAnalyzer } from './generator/useProjectAnalyzer'
import { useFileParser } from './generator/useFileParser'
import { useSeriesGenerator } from './generator/useSeriesGenerator'

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
