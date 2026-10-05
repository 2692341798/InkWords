// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { VideoRunbookPanel } from './VideoRunbookPanel'

afterEach(cleanup)

const runbook = {
  format: 'inkwords.video-runbook.v1', stack: 'web', observation_goal: '性能',
  recommendation: { primary: '浏览器开发者工具', alternative: 'VS Code', reason: '性能证据来自浏览器。', manual_capture_required: false },
  steps: [{ tool: '浏览器开发者工具', tool_version: '当前稳定版', start_state: '教学页已打开。', action: '打开开发者工具。', shortcut_or_menu: 'F12', input: '打开 Performance 面板', expected_view: '性能时间线可见。', narration: '观察真实数据。', capture_point: '性能时间线', recovery: '从菜单打开。', completion_signal: '面板无旧记录。' }],
  capture_checklist: ['记录版本。'], manual_capture_pending: true, verification_status: 'unverified',
}

describe('VideoRunbookPanel', () => {
  it('shows a complete manual filming guide but does not overstate verification', () => {
    render(<VideoRunbookPanel documentJson={{ video_runbook: runbook }} />)
    expect(screen.getByText('副屏视频操作教案')).toBeTruthy()
    expect(screen.getByText('待人工采集证据')).toBeTruthy()
    expect(screen.getByText('F12')).toBeTruthy()
    expect(screen.getByText('人工采集清单')).toBeTruthy()
  })

  it('hides malformed untrusted document data', () => {
    const { container } = render(<VideoRunbookPanel documentJson={{ video_runbook: { format: 'inkwords.video-runbook.v1' } }} />)
    expect(container.textContent).toBe('')
  })
})
