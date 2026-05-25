import { useStreamStore } from '@/store/streamStore'
import { buildFileAnalyzeRequest } from './fileParserUtils'

/**
 * Why: 文件上传流程会先更新来源类型，再发起分析请求。
 * 这里统一从最新 store 读取场景，避免旧渲染快照导致“界面显示”和“请求参数”不一致。
 */
export function buildCurrentFileAnalyzeRequest(sourceContent: string) {
  return buildFileAnalyzeRequest(
    sourceContent,
    useStreamStore.getState().scenarioMode,
  )
}
