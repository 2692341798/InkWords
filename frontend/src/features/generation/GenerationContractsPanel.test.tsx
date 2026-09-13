// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { GenerationContractsPanel } from './GenerationContractsPanel'

const project = { id: 'project-1', title: 'Gin 自学教材', audience_level: 'foundation' as const, status: 'draft' as const, revision_version: 1, updated_at: '2026-09-02T00:00:00Z' }

describe('GenerationContractsPanel', () => {
  afterEach(cleanup)

  it('saves an explicit foundation-reader BookContract draft instead of treating it as approved', async () => {
    const onCreateBookContract = vi.fn().mockResolvedValue(undefined)
    render(<GenerationContractsPanel project={project} isBusy={false} onCreateBookContract={onCreateBookContract} onCreateStyleSheet={vi.fn()} onApproveBookContract={vi.fn()} onApproveStyleSheet={vi.fn()} />)

    expect(screen.getAllByText('未创建')).toHaveLength(2)
    const promise = screen.getByRole('textbox', { name: '教学承诺' })
    fireEvent.change(promise, { target: { value: '让读者理解 Gin 为什么需要路由。' } })
    fireEvent.submit(promise.closest('form')!)

    await waitFor(() => expect(onCreateBookContract).toHaveBeenCalledWith(expect.objectContaining({
      promise: '让读者理解 Gin 为什么需要路由。',
      reader: expect.objectContaining({ audience: 'foundation', learning_outcomes: expect.arrayContaining(['能用自己的话解释核心原理']) }),
      chapter_profiles: ['concept', 'hands_on'],
    })))
  })

  it('requires a separate approval click for each current draft', async () => {
    const onApproveBookContract = vi.fn().mockResolvedValue(undefined)
    const onApproveStyleSheet = vi.fn().mockResolvedValue(undefined)
    render(<GenerationContractsPanel project={project} bookContract={{ id: 'book-r1', project_id: project.id, revision_number: 1, document_json: {}, content_hash: 'a'.repeat(64), status: 'draft', created_at: project.updated_at }} styleSheet={{ id: 'style-r1', project_id: project.id, revision_number: 1, document_json: {}, content_hash: 'b'.repeat(64), status: 'draft', created_at: project.updated_at }} isBusy={false} onCreateBookContract={vi.fn()} onCreateStyleSheet={vi.fn()} onApproveBookContract={onApproveBookContract} onApproveStyleSheet={onApproveStyleSheet} />)

    expect(screen.getAllByText('草稿待审')).toHaveLength(2)
    const buttons = screen.getAllByRole('button', { name: '我已审阅并批准' })
    fireEvent.click(buttons[0])
    fireEvent.click(buttons[1])

    await waitFor(() => expect(onApproveBookContract).toHaveBeenCalledWith('book-r1'))
    await waitFor(() => expect(onApproveStyleSheet).toHaveBeenCalledWith('style-r1'))
  })
})
