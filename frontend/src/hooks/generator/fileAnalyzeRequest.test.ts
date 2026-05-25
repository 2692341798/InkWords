import { beforeEach, describe, expect, it } from 'vitest'
import { useStreamStore } from '@/store/streamStore'
import { buildCurrentFileAnalyzeRequest } from './fileAnalyzeRequest'

describe('buildCurrentFileAnalyzeRequest', () => {
  beforeEach(() => {
    useStreamStore.getState().reset()
  })

  it('reads the latest scenario mode from the store', () => {
    useStreamStore.getState().setScenarioMode('open_book_exam_review')
    const staleSnapshot = useStreamStore.getState()

    staleSnapshot.setSource('file', 'demo.zip')

    expect(buildCurrentFileAnalyzeRequest('parsed file content')).toEqual({
      source_type: 'file',
      source_content: 'parsed file content',
      scenario_mode: 'open_book_exam_review',
    })
  })
})
