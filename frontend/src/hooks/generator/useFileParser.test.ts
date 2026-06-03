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

  it('uses task-based parsing for zip archives and stores the async result', async () => {
    const store = createMockStore()
    const parseProjectFile = vi.fn()
    const createParseTask = vi.fn().mockResolvedValue({
      task_id: 'task-zip-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-zip-1/stream',
    })
    const getTaskSnapshot = vi.fn().mockResolvedValue({
      id: 'task-zip-1',
      status: 'succeeded',
      result: {
        source_content: 'async parsed zip content',
        archive_summary: {
          total_files: 3,
          kept_files: 2,
          duplicate_files: 0,
          ignored_files: 1,
          failed_files: 0,
        },
      },
    })
    fetchEventSourceWithAuth.mockResolvedValue(undefined)

    await expect(
      parseUploadedFile({
        file: new File(['zip'], 'course.zip'),
        store,
        parseProjectFile,
        createParseTask,
        getTaskSnapshot,
      }),
    ).resolves.toBe('async parsed zip content')

    expect(parseProjectFile).not.toHaveBeenCalled()
    expect(createParseTask).toHaveBeenCalled()
    expect(getTaskSnapshot).toHaveBeenCalledWith('task-zip-1')
    expect(store.setSourceContent).toHaveBeenCalledWith('async parsed zip content')
  })

  it('uses task-based parsing for non-zip files larger than 50MB', async () => {
    const store = createMockStore()
    const parseProjectFile = vi.fn()
    const createParseTask = vi.fn().mockResolvedValue({
      task_id: 'task-file-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-file-1/stream',
    })
    const getTaskSnapshot = vi.fn().mockResolvedValue({
      id: 'task-file-1',
      status: 'succeeded',
      result: {
        source_content: 'async parsed pdf content',
      },
    })
    fetchEventSourceWithAuth.mockResolvedValue(undefined)

    const file = new File(['pdf'], 'course.pdf', { type: 'application/pdf' })
    Object.defineProperty(file, 'size', { configurable: true, value: 50 * 1024 * 1024 + 1 })

    await expect(
      parseUploadedFile({
        file,
        store,
        parseProjectFile,
        createParseTask,
        getTaskSnapshot,
      }),
    ).resolves.toBe('async parsed pdf content')

    expect(parseProjectFile).not.toHaveBeenCalled()
    expect(createParseTask).toHaveBeenCalledWith(file)
    expect(getTaskSnapshot).toHaveBeenCalledWith('task-file-1')
  })

  it('keeps small non-zip files on the synchronous parse path', async () => {
    const store = createMockStore()
    const parseProjectFile = vi.fn().mockResolvedValue({
      data: {
        source_content: 'sync parsed markdown content',
      },
    })
    const createParseTask = vi.fn()
    const getTaskSnapshot = vi.fn()

    const file = new File(['markdown'], 'guide.md', { type: 'text/markdown' })
    Object.defineProperty(file, 'size', { configurable: true, value: 5 * 1024 * 1024 })

    await expect(
      parseUploadedFile({
        file,
        store,
        parseProjectFile,
        createParseTask,
        getTaskSnapshot,
      }),
    ).resolves.toBe('sync parsed markdown content')

    expect(parseProjectFile).toHaveBeenCalledOnce()
    expect(createParseTask).not.toHaveBeenCalled()
    expect(getTaskSnapshot).not.toHaveBeenCalled()
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
