import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useStreamStore } from '@/store/streamStore'
import { analyzeParsedFileContent, parseUploadedFile } from './useFileParser'

const { fetchEventSourceWithAuth } = vi.hoisted(() => ({
  fetchEventSourceWithAuth: vi.fn(),
}))

vi.mock('@/services/sse', () => ({
  fetchEventSourceWithAuth,
}))

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
    setResolvedPromptProfile: vi.fn(),
  })

  beforeEach(() => {
    vi.restoreAllMocks()
    fetchEventSourceWithAuth.mockReset()
    useStreamStore.getState().reset()
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

  it('writes the resolved prompt profile into the store when analyze completes', async () => {
    fetchEventSourceWithAuth.mockImplementation(async (_url, options) => {
      options.onmessage?.({
        event: 'chunk',
        data: JSON.stringify({
          status: 'analyzing',
          message: '正在识别文档内容类型',
        }),
      })
      options.onmessage?.({
        event: 'chunk',
        data: JSON.stringify({
          status: 'complete',
          message: '大纲生成完成',
          content: JSON.stringify({
            source_content: '解析后的内容',
            series_title: '《非暴力沟通》解读',
            outline: [{ sort: 1, title: '第一章', summary: '观察与感受' }],
            resolved_prompt_profile: {
              key: 'psychology_communication_book',
              display_name: '心理学经典解读',
              document_kind: 'psychology_communication',
              reason: '命中沟通与情绪表达主题',
            },
          }),
        }),
      })
      options.onmessage?.({ event: 'done', data: '[DONE]' })
    })

    await analyzeParsedFileContent('解析后的内容')

    expect(useStreamStore.getState()).toMatchObject({
      classificationStatus: 'resolved',
      classificationReason: '命中沟通与情绪表达主题',
      seriesTitle: '《非暴力沟通》解读',
      resolvedPromptProfile: {
        key: 'psychology_communication_book',
        displayName: '心理学经典解读',
        documentKind: 'psychology_communication',
      },
    })
  })
})
