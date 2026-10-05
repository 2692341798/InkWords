// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { UsagePanel } from './UsagePanel'

describe('UsagePanel', () => {
  afterEach(cleanup)
  it('shows recorded tokens, calls, latency, and an explicit unknown cost', () => {
    render(<UsagePanel usage={{ known: true, input_tokens: 12, output_tokens: 8, cached_tokens: 3, provider_call_count: 1, provider_latency_ms: 42, local_cache_hit: false, estimated_cost_known: false }} />)
    expect(screen.getByLabelText('生成使用量').textContent).toContain('输入 Token')
    expect(screen.getByLabelText('生成使用量').textContent).toContain('供应商调用：1 次 · 调用耗时：42 ms')
    expect(screen.getByLabelText('生成使用量').textContent).toContain('未知（尚未配置版本化定价规则）')
  })

  it('does not render unknown usage as zero', () => {
    render(<UsagePanel usage={{ known: false, provider_call_count: 0, local_cache_hit: true, estimated_cost_known: false }} />)
    expect(screen.getByText('供应商未返回 Token 使用量（未知，不按 0 计）。')).toBeTruthy()
    expect(screen.getByText('供应商调用：0 次 · 调用耗时：未知 ms · 本地缓存命中，未发起新的供应商调用。')).toBeTruthy()
  })

  it('keeps each omitted telemetry field unknown even when other token fields were recorded', () => {
    render(<UsagePanel usage={{ known: true, input_tokens: 4064, output_tokens: 5191 }} />)
    const panel = screen.getByLabelText('生成使用量')
    expect(panel.textContent).toContain('缓存 Token未知')
    expect(panel.textContent).toContain('供应商调用：未知 次 · 调用耗时：未知 ms')
    expect(panel.textContent).toContain('本地缓存状态未知。')
    expect(panel.textContent).not.toContain('本次没有本地缓存命中。')
  })
})
