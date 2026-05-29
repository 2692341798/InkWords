import { describe, expect, it } from 'vitest'
import { getKnowledgeReviewViewState } from './knowledgeReviewViewState'

describe('getKnowledgeReviewViewState', () => {
  it('starts from the entry step when no picker or session is active', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: false,
        isPickerOpen: false,
      }),
    ).toEqual({
      currentStep: 'entry',
      currentStepIndex: 0,
      shouldShowEntryStep: true,
      shouldShowPickerStep: false,
      shouldShowSessionStep: false,
      shouldShowHistory: true,
    })
  })

  it('switches to the picker step when manual selection is opened', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: false,
        isPickerOpen: true,
      }),
    ).toEqual({
      currentStep: 'picker',
      currentStepIndex: 1,
      shouldShowEntryStep: false,
      shouldShowPickerStep: true,
      shouldShowSessionStep: false,
      shouldShowHistory: false,
    })
  })

  it('switches to the session step when a review session exists', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: true,
        isPickerOpen: true,
      }),
    ).toEqual({
      currentStep: 'session',
      currentStepIndex: 2,
      shouldShowEntryStep: false,
      shouldShowPickerStep: false,
      shouldShowSessionStep: true,
      shouldShowHistory: false,
    })
  })
})
