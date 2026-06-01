import { describe, expect, it } from 'vitest'
import { getKnowledgeReviewViewState } from './knowledgeReviewViewState'

describe('getKnowledgeReviewViewState', () => {
  it('starts from the entry step when no picker or session is active', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: false,
        isPickerOpen: false,
        shouldEnterSession: false,
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
        shouldEnterSession: false,
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

  it('keeps the entry step when a remembered session exists but the user did not choose to continue it', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: true,
        isPickerOpen: false,
        shouldEnterSession: false,
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

  it('switches to the session step only when the current visit explicitly enters the session flow', () => {
    expect(
      getKnowledgeReviewViewState({
        hasSession: true,
        isPickerOpen: true,
        shouldEnterSession: true,
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
