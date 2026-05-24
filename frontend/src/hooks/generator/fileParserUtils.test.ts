import { describe, expect, it } from 'vitest'
import {
  buildFileAnalyzeRequest,
  extractArchiveSummary,
  extractParsedFileContent,
  formatArchiveSummaryMessage,
} from './fileParserUtils'

describe('fileParserUtils', () => {
  it('reads parsed file content from the backend data wrapper', () => {
    expect(
      extractParsedFileContent({
        data: {
          source_content: 'parsed pdf content',
        },
      }),
    ).toBe('parsed pdf content')
  })

  it('builds analyze payload with file source type', () => {
    expect(buildFileAnalyzeRequest('parsed pdf content', 'ebook_interpretation')).toEqual({
      source_type: 'file',
      source_content: 'parsed pdf content',
      scenario_mode: 'ebook_interpretation',
    })
  })

  it('reads archive summary from the backend data wrapper', () => {
    expect(
      extractArchiveSummary({
        data: {
          archive_summary: {
            total_files: 8,
            kept_files: 3,
            duplicate_files: 2,
            ignored_files: 2,
            failed_files: 1,
          },
        },
      }),
    ).toEqual({
      total_files: 8,
      kept_files: 3,
      duplicate_files: 2,
      ignored_files: 2,
      failed_files: 1,
    })
  })

  it('formats archive summary into a Chinese history message', () => {
    expect(
      formatArchiveSummaryMessage({
        total_files: 8,
        kept_files: 3,
        duplicate_files: 2,
        ignored_files: 2,
        failed_files: 1,
      }),
    ).toBe('压缩包共扫描 8 个文件，保留 3 个，去重 2 个，忽略 2 个，失败 1 个')
  })
})
