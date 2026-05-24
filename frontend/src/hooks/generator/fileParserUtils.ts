import type { ScenarioMode } from '@/lib/scenarioMode'

export interface ArchiveSummary {
  total_files: number
  supported_files?: number
  kept_files: number
  duplicate_files: number
  ignored_files: number
  failed_files: number
  kept_paths?: string[]
}

interface ParseFileResponse {
  content?: string
  data?: {
    source_content?: string
    archive_summary?: ArchiveSummary
  }
}

export function extractParsedFileContent(response: ParseFileResponse): string {
  return response.data?.source_content ?? response.content ?? ''
}

export function extractArchiveSummary(response: ParseFileResponse): ArchiveSummary | undefined {
  return response.data?.archive_summary
}

export function formatArchiveSummaryMessage(summary: ArchiveSummary): string {
  return `压缩包共扫描 ${summary.total_files} 个文件，保留 ${summary.kept_files} 个，去重 ${summary.duplicate_files} 个，忽略 ${summary.ignored_files} 个，失败 ${summary.failed_files} 个`
}

export function buildFileAnalyzeRequest(
  sourceContent: string,
  scenarioMode: ScenarioMode,
) {
  return {
    source_type: 'file' as const,
    source_content: sourceContent,
    scenario_mode: scenarioMode,
  }
}
