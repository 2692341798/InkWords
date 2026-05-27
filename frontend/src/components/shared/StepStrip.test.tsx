import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { StepStrip } from './StepStrip'

describe('StepStrip', () => {
  it('renders a preview strip with all provided step titles', () => {
    const html = renderToStaticMarkup(
      <StepStrip
        title="流程预览"
        description="用于首页预览路径。"
        variant="preview"
        steps={[
          { key: 'source', title: '选择来源', description: '先确定资料入口。' },
          { key: 'analysis', title: '完成解析' },
          { key: 'outline', title: '确认大纲' },
        ]}
      />,
    )

    expect(html).toContain('流程预览')
    expect(html).toContain('选择来源')
    expect(html).toContain('完成解析')
    expect(html).toContain('确认大纲')
    expect(html).toContain('先确定资料入口。')
    expect(html).toContain('data-step-state="preview"')
    expect(html).toContain('data-variant="preview"')
  })

  it('renders progress state markers for current and completed steps', () => {
    const html = renderToStaticMarkup(
      <StepStrip
        variant="progress"
        currentStepIndex={1}
        steps={[
          { key: 'entry', title: '选择入口' },
          { key: 'session', title: '开始会话' },
          { key: 'feedback', title: '获得反馈' },
        ]}
      />,
    )

    expect(html).toContain('data-variant="progress"')
    expect(html).toContain('data-step-state="complete"')
    expect(html).toContain('data-step-state="current"')
    expect(html).toContain('data-step-state="upcoming"')
    expect(html).toContain('data-step-emphasis="strong"')
    expect(html).toContain('data-step-emphasis="soft"')
  })
})
