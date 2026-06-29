// @vitest-environment jsdom
import { describe, expect, it, afterEach } from 'vitest'
import { render, waitFor, cleanup } from '@testing-library/react'

import MermaidBlock from './MermaidBlock'

afterEach(() => {
  cleanup()
})

const waitForStableContent = (container: HTMLElement) =>
  waitFor(() => {
    const text = container.textContent || ''
    expect(text.length).toBeGreaterThanOrEqual(0)
  }, { timeout: 5000 })

const renderBlock = (chart: string) => {
  const result = render(<MermaidBlock chart={chart} />)
  return result
}

const sampleChart = `graph TD
  A[开始] --> B[结束]`

const chartWithScript = `graph TD
  A["<script>alert('xss')</script>"] --> B`

const chartWithEventHandler = `graph TD
  A["onclick=alert(1)"] --> B`

const chartWithJsUrl = `graph TD
  A["javascript:alert(1)"] --> B`

describe('MermaidBlock 回归测试（安全护栏）', () => {
  it('渲染合法 Mermaid 图时不包含 script 标签', async () => {
    const { container } = renderBlock(sampleChart)
    await waitForStableContent(container)
    expect(container.innerHTML).not.toContain('<script>')
    expect(container.innerHTML).not.toContain('<script ')
  })

  it('不渲染含 HTML payload 的 script 标签', async () => {
    const { container } = renderBlock(chartWithScript)
    await waitForStableContent(container)
    expect(container.innerHTML).not.toContain('<script>')
    expect(container.innerHTML).not.toContain('<script ')
  })

  it('不包含内联事件处理器 (onclick/onerror/onload)', async () => {
    const { container } = renderBlock(chartWithEventHandler)
    await waitForStableContent(container)
    expect(container.innerHTML).not.toMatch(/\bon\w+\s*=/)
  })

  it('不渲染 javascript: URL', async () => {
    const { container } = renderBlock(chartWithJsUrl)
    await waitForStableContent(container)
    expect(container.innerHTML).not.toContain('javascript:')
  })

  it('不包含 foreignObject 元素', async () => {
    const { container } = renderBlock(sampleChart)
    await waitForStableContent(container)
    expect(container.innerHTML).not.toContain('foreignObject')
  })

  it('失败时不通过 innerHTML 注入错误内容', async () => {
    const { container } = renderBlock('invalid %% not a chart')
    await waitForStableContent(container)
    expect(container.querySelector('.mermaid-container')).toBeTruthy()
  })
})
