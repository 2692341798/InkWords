interface ParseFileResponse {
  content?: string
  data?: {
    source_content?: string
  }
}

export function extractParsedFileContent(response: ParseFileResponse): string {
  return response.data?.source_content ?? response.content ?? ''
}

export function buildFileAnalyzeRequest(sourceContent: string) {
  return {
    source_type: 'file' as const,
    source_content: sourceContent,
  }
}
