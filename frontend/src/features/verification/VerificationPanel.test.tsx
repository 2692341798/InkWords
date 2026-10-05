// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { VerificationPanel } from './VerificationPanel'

afterEach(cleanup)

describe('VerificationPanel', () => {
  it('identifies the revision so historical code is not mistaken for the current manuscript', () => {
    render(<VerificationPanel revisionNumbers={{ 'revision-1': 12 }} currentSourceRevisionId="revision-1" artifacts={[{
      id: 'artifact', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'unverified', created_at: '',
    }]} evidence={[]} assets={[]} />)
    expect(screen.getByRole('article', { name: '教学工件 r12' })).toBeTruthy()
    expect(screen.getByText('来源修订：r12 · 当前稿件的代码来源')).toBeTruthy()
  })
  it('does not confuse staged code with a verified execution', () => {
    render(<VerificationPanel artifacts={[{
      id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', entrypoint: 'router.go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: ['这是教学模型，不是 Gin 源码。'], status: 'unverified', created_at: '2026-09-03T00:00:00Z',
    }]} evidence={[]} assets={[]} />)

    expect(screen.getByText('教学代码与运行证据')).toBeTruthy()
    expect(screen.getByText('尚无当前有效的运行证据')).toBeTruthy()
    expect(screen.getByText('未验证')).toBeTruthy()
    expect(screen.getByText('尚未采集运行证据。请不要把此工件的讲解或预期输出标为“已实测”。')).toBeTruthy()
  })

  it('treats expired evidence as unverified until it is collected again', () => {
    render(<VerificationPanel artifacts={[{
      id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', entrypoint: 'router.go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified', created_at: '2026-09-03T00:00:00Z',
    }]} evidence={[{
      id: 'evidence-1', revision_id: 'revision-1', code_artifact_id: 'artifact-1', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', output_truncated: false, expires_at: '2020-01-01T00:00:00Z', created_at: '2020-01-01T00:00:00Z',
    }]} assets={[]} />)

    expect(screen.getByText('尚无当前有效的运行证据')).toBeTruthy()
    expect(screen.getByText('运行证据已过期，需重新验证后才能作为当前事实使用。')).toBeTruthy()
    expect(screen.getAllByText('未验证')).toHaveLength(2)
    expect(screen.queryByText('已验证')).toBeNull()
  })

  it('treats contract-stale evidence as unverified even before its time expiry', () => {
    render(<VerificationPanel artifacts={[{
      id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', entrypoint: 'router.go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified', created_at: '2026-09-03T00:00:00Z',
    }]} evidence={[{
      id: 'evidence-1', revision_id: 'revision-1', code_artifact_id: 'artifact-1', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', output_truncated: false, expires_at: '2099-09-03T00:00:00Z', stale_reason: 'BookContract 已变化，需重新验证。', created_at: '2026-09-03T00:00:00Z',
    }]} assets={[]} />)
    expect(screen.getAllByText('未验证')).toHaveLength(2)
    expect(screen.queryByText('已验证')).toBeNull()
    expect(screen.getByText('BookContract 已变化，需重新验证。')).toBeTruthy()
  })

  it('shows why a manual screenshot is needed instead of structured text', () => {
    render(<VerificationPanel artifacts={[{
      id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified', created_at: '2026-09-03T00:00:00Z',
    }]} evidence={[{
      id: 'evidence-1', revision_id: 'revision-1', code_artifact_id: 'artifact-1', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', output_truncated: false, expires_at: '2099-09-03T00:00:00Z', created_at: '2026-09-03T00:00:00Z',
    }]} assets={[{
      id: 'asset-1', revision_id: 'revision-1', evidence_id: 'evidence-1', stable_ref: 'inkwords-asset:stack', kind: 'screenshot', content_hash: 'sha256:asset', alt_text: '断点处的调用栈', source: '本地 GoLand 手工截图', generation_method: 'manual_capture', visual_purpose: 'call_stack', rights_status: 'pending', status: 'unverified', created_at: '2026-09-03T00:00:00Z',
    }]} />)
    expect(screen.getByText('视觉用途：调用栈')).toBeTruthy()
  })

  it('labels tool observations separately from explanations', () => {
    render(<VerificationPanel artifacts={[{
      id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified', created_at: '2026-09-03T00:00:00Z',
    }]} evidence={[{
      id: 'evidence-1', revision_id: 'revision-1', code_artifact_id: 'artifact-1', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'terminal_output', status: 'verified', output_truncated: false, expires_at: '2099-09-03T00:00:00Z', structured_output: '{"format":"inkwords.runtime-observation.v1","observations":{"commands":[{"exit_code":0}]},"interpretations":[]}', created_at: '2026-09-03T00:00:00Z',
    }]} assets={[]} />)

    expect(screen.getByText('结构化观察：1 条工具结果；解释：未记录。')).toBeTruthy()
    expect(screen.getByText('查看运行详情与原始记录')).toBeTruthy()
    expect(screen.getByLabelText('原始运行记录').textContent).toContain('"exit_code":0')
  })

	it('shows captured browser versions and distinguishes automatic screenshots from manual captures', () => {
		render(<VerificationPanel artifacts={[{
			id: 'artifact-1', revision_id: 'revision-1', kind: 'teaching_implementation', language: 'go', manifest_hash: 'sha256:manifest', artifact_hash: 'sha256:artifact', limitations_json: [], status: 'verified', created_at: '2026-09-03T00:00:00Z',
		}]} evidence={[{
			id: 'browser-evidence', revision_id: 'revision-1', code_artifact_id: 'artifact-1', code_artifact_hash: 'sha256:artifact', input_hash: 'sha256:input', kind: 'browser_page', status: 'verified', toolchain_version: 'go1.25.4', tool_name: 'Playwright/Chromium', tool_version: 'Playwright 1.62.1; Chromium 151.0.7922.34; runner-image:sha256:runner', structured_output: '{"format":"inkwords.runtime-observation.v1","observations":{"browser_pages":[{"browser_name":"Chromium","browser_version":"151.0.7922.34","playwright_version":"1.62.1"}]},"interpretations":[]}', output_truncated: false, captured_at: '2026-09-03T00:00:00Z', expires_at: '2099-09-03T00:00:00Z', created_at: '2026-09-03T00:00:00Z',
		}]} assets={[{
			id: 'asset-1', revision_id: 'revision-1', evidence_id: 'browser-evidence', stable_ref: 'inkwords-asset:browser', kind: 'screenshot', content_hash: 'sha256:asset', alt_text: '受控教学页面', source: 'Bubblewrap 内 Playwright', generation_method: 'isolated_playwright_probe', visual_purpose: 'rendered_ui', rights_status: 'pending', status: 'verified', created_at: '2026-09-03T00:00:00Z',
		}]} />)

		expect(screen.getByText(/工具：Playwright\/Chromium/)).toBeTruthy()
		expect(screen.getByText(/Playwright 1.62.1；Chromium 151.0.7922.34/)).toBeTruthy()
		expect(screen.getByText('自动浏览器截图资产')).toBeTruthy()
	})
})
