import { describe, expect, it } from 'vitest'
import {
  buildFileAnalyzeRequest,
  extractParsedFileContent,
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
    expect(buildFileAnalyzeRequest('parsed pdf content')).toEqual({
      source_type: 'file',
      source_content: 'parsed pdf content',
    })
  })
})
