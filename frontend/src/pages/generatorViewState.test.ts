import { describe, expect, it } from 'vitest'
import { getGeneratorViewState } from './generatorViewState'

describe('getGeneratorViewState', () => {
  it('stays on source before any source is ready', () => {
    expect(
      getGeneratorViewState({
        sourceType: null,
        sourceContent: '',
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'source',
      currentStepIndex: 0,
      shouldShowInlineProgress: false,
      progressHostStage: null,
      shouldShowScenarioSelector: false,
    })
  })

  it('keeps parsing and analysis inside configure instead of switching to a progress stage', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'file',
        sourceContent: 'parsed zip content',
        modules: null,
        outline: null,
        scenarioMode: 'ebook_interpretation',
        isScanning: false,
        isAnalyzing: true,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'configure',
      currentStepIndex: 1,
      shouldShowInlineProgress: true,
      progressHostStage: 'configure',
    })
  })

  it('keeps generation inside outline instead of switching to a progress stage', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        sourceContent: 'repo summary',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: [{ sort: 1, title: '第一篇', summary: '摘要' }],
        scenarioMode: 'ebook_interpretation',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: true,
      }),
    ).toMatchObject({
      currentStage: 'outline',
      currentStepIndex: 2,
      shouldShowInlineProgress: true,
      progressHostStage: 'outline',
    })
  })
})
