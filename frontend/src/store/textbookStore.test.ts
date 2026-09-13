// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { textbookService } from '@/services/textbook'
import { useTextbookStore } from './textbookStore'

vi.mock('@/services/textbook', () => ({
  textbookService: {
    getChapterWorkspace: vi.fn(),
    applyCandidate: vi.fn(),
  },
}))

const getChapterWorkspace = vi.mocked(textbookService.getChapterWorkspace)

describe('textbookStore chapter task recovery', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    window.sessionStorage.clear()
    useTextbookStore.setState({
      selectedChapterWorkspace: null,
      selectedProject: null,
      latestSampleTask: null,
      isLoading: false,
      error: null,
    })
  })

  it('recovers the latest sample task from the server when browser-local state is empty', async () => {
    getChapterWorkspace.mockResolvedValue({
      code: 0,
      data: {
        chapter: { id: 'chapter-1' },
        generation_target: { provider_name: 'deepseek', model_name: 'deepseek-v4-flash' },
        revisions: [],
        candidate_reviews: [],
        code_artifacts: [],
        runtime_evidence: [],
        assets: [],
        latest_sample_task: {
          task_id: 'task-server-latest',
          status: 'failed',
          created_at: '2026-09-06T00:00:00Z',
        },
      },
    } as never)

    await useTextbookStore.getState().openChapter('chapter-1')

    expect(useTextbookStore.getState().latestSampleTask).toEqual({
      task_id: 'task-server-latest',
      status: 'failed',
      created_at: '2026-09-06T00:00:00Z',
    })
    expect(window.sessionStorage.getItem('inkwords:textbook:sample-task:chapter-1')).toBe('task-server-latest')
  })

  it('sends delegated provenance together with the current lock and revision', async () => {
    const workspace = {chapter: {id: 'chapter', revision_version: 8}, lock: {owner_id: 'owner', version: 2}, revisions: [], candidate_reviews: [], code_artifacts: [], runtime_evidence: [], assets: []}
    useTextbookStore.setState({selectedChapterWorkspace: workspace as never})
    getChapterWorkspace.mockResolvedValue({code: 0, data: workspace} as never)
    vi.mocked(textbookService.applyCandidate).mockResolvedValue({code: 0, data: {}} as never)
    await useTextbookStore.getState().applyCandidate({candidateRevisionId: 'candidate', ownerId: 'owner', reviewNote: '逐项核对后批准。', dimensionScores: [], reviewerKind: 'delegated_ai', delegationNote: '用户明确委托 Codex 执行审阅。'})
    expect(textbookService.applyCandidate).toHaveBeenCalledWith('chapter', 'candidate', expect.objectContaining({expected_version: 8, lock_version: 2, lock_owner_id: 'owner', reviewer_kind: 'delegated_ai', delegation_note: '用户明确委托 Codex 执行审阅。'}))
  })
})
