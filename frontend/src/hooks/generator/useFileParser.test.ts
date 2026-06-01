import { beforeEach, describe, expect, it, vi } from 'vitest'
import { parseUploadedFile } from './useFileParser'

describe('parseUploadedFile', () => {
  const createMockStore = () => ({
    abortController: null as AbortController | null,
    setAnalyzing: vi.fn(),
    setAnalysisStep: vi.fn(),
    clearAnalysisHistory: vi.fn(),
    setAnalysisMessage: vi.fn(),
    appendAnalysisHistory: vi.fn(),
    setSource: vi.fn(),
    setOutline: vi.fn(),
    setAbortController: vi.fn(),
    setSourceContent: vi.fn(),
  })

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('stops after successful file parsing and leaves scene selection to the next step', async () => {
    const store = createMockStore()
    const parseProjectFile = vi.fn().mockResolvedValue({
      data: {
        source_content: 'parsed zip content',
      },
    })

    await expect(
      parseUploadedFile({
        file: new File(['zip'], 'course.zip'),
        store,
        parseProjectFile,
      }),
    ).resolves.toBe('parsed zip content')

    expect(store.setSource).toHaveBeenCalledWith('file', '')
    expect(store.setSourceContent).toHaveBeenCalledWith('parsed zip content')
    expect(store.setAnalysisStep).toHaveBeenLastCalledWith(1)
    expect(store.setAnalysisMessage).toHaveBeenLastCalledWith('文件解析完成，请选择创作场景')
    expect(store.setAnalyzing).toHaveBeenLastCalledWith(false)
  })

  it('shows an extra hint when the parsed document is very large', async () => {
    const store = createMockStore()
    const parseProjectFile = vi.fn().mockResolvedValue({
      data: {
        source_content: 'A'.repeat(1000001),
      },
    })

    await expect(
      parseUploadedFile({
        file: new File(['pdf'], 'course.pdf'),
        store,
        parseProjectFile,
      }),
    ).resolves.toHaveLength(1000001)

    expect(store.appendAnalysisHistory).toHaveBeenCalledWith(
      expect.objectContaining({
        message: expect.stringContaining('超大文档'),
        status: 'parsed',
      }),
    )
  })
})
