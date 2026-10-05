// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { testConfiguredProviderConnection } = vi.hoisted(() => ({ testConfiguredProviderConnection: vi.fn() }))
vi.mock('@/services/textbook', () => ({ textbookService: { testConfiguredProviderConnection } }))

import { ProviderConnectionPanel } from './ProviderConnectionPanel'

describe('ProviderConnectionPanel', () => {
  beforeEach(() => {
    cleanup()
    testConfiguredProviderConnection.mockReset()
  })

  it('only runs the connectivity probe after an explicit click and renders safe telemetry', async () => {
    testConfiguredProviderConnection.mockResolvedValue({ code: 0, data: { provider: 'openai', model: 'gpt-test', request_id: 'request-safe', usage_known: false } })
    render(<ProviderConnectionPanel />)
    expect(testConfiguredProviderConnection).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: '测试已配置的模型连接' }))
    await waitFor(() => expect(testConfiguredProviderConnection).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('status').textContent).toContain('openai / gpt-test')
    expect(screen.getByRole('status').textContent).toContain('Token 未知')
  })

  it('does not expose a transport failure when the probe cannot run', async () => {
    testConfiguredProviderConnection.mockRejectedValue(new Error('OPENAI_API_KEY=must-never-appear'))
    render(<ProviderConnectionPanel />)
    fireEvent.click(screen.getByRole('button', { name: '测试已配置的模型连接' }))
    expect((await screen.findByRole('alert')).textContent).toContain('模型连接测试失败；请检查本机提供商配置后重试')
    expect(screen.getByRole('alert').textContent).not.toContain('must-never-appear')
  })
})
