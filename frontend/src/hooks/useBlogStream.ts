import { useCallback } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useProjectScanner } from './generator/useProjectScanner'
import { useProjectAnalyzer } from './generator/useProjectAnalyzer'
import { useFileParser } from './generator/useFileParser'
import { useSeriesGenerator } from './generator/useSeriesGenerator'

export const useBlogStream = () => {
  const store = useStreamStore()
  const { scanGit } = useProjectScanner()
  const { analyzeGit } = useProjectAnalyzer()
  const { parseFile } = useFileParser()
  const { generateSeries, generateSingle } = useSeriesGenerator()

  const stopAnalyzing = useCallback(() => {
    if (store.abortController) {
      store.abortController.abort()
      store.setAbortController(null)
    }
    store.setAnalyzing(false)
    store.setScanning(false)
    store.setAnalysisMessage('解析已取消')
  }, [store])

  const stopGenerating = useCallback(() => {
    if (store.abortController) {
      store.abortController.abort()
      store.setAbortController(null)
    }
    store.setGenerating(false)
    store.setProgress('已停止生成')
  }, [store])

  return {
    scanGit,
    analyzeGit,
    parseFile,
    generateSeries,
    generateSingle,
    stopAnalyzing,
    stopGenerating
  }
}
