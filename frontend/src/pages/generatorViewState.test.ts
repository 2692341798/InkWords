import { describe, expect, it } from 'vitest'
import { getGeneratorViewState } from './generatorViewState'

describe('getGeneratorViewState', () => {
  it('starts from the input step before the source is fully configured', () => {
    expect(
      getGeneratorViewState({
        sourceType: null,
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toEqual({
      currentStep: 'input',
      currentStepIndex: 0,
      shouldShowInputStep: true,
      shouldShowConfigureStep: false,
      shouldShowOutlineStep: false,
      shouldShowScenarioSelector: true,
      lockedScenarioLabel: null,
      isProcessing: false,
    })
  })

  it('switches to the configure step after git scan returns modules', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toEqual({
      currentStep: 'configure',
      currentStepIndex: 1,
      shouldShowInputStep: false,
      shouldShowConfigureStep: true,
      shouldShowOutlineStep: false,
      shouldShowScenarioSelector: true,
      lockedScenarioLabel: null,
      isProcessing: false,
    })
  })

  it('switches to the outline step after outline generation', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: [{ sort: 1, title: '章节一', summary: '摘要' }],
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toEqual({
      currentStep: 'outline',
      currentStepIndex: 2,
      shouldShowInputStep: false,
      shouldShowConfigureStep: false,
      shouldShowOutlineStep: true,
      shouldShowScenarioSelector: false,
      lockedScenarioLabel: '开卷复习',
      isProcessing: false,
    })
  })

  it('marks the page as processing while a scan or generation is running', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: true,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toEqual({
      currentStep: 'processing',
      currentStepIndex: 3,
      shouldShowInputStep: false,
      shouldShowConfigureStep: false,
      shouldShowOutlineStep: false,
      shouldShowScenarioSelector: false,
      lockedScenarioLabel: null,
      isProcessing: true,
    })
  })
})
