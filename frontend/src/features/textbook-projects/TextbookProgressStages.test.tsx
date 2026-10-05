// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { TextbookProgressStages } from './TextbookProgressStages'

describe('TextbookProgressStages', () => {
  afterEach(cleanup)

  it('renders backend stage facts and does not convert unavailable capabilities into completion', () => {
    render(<TextbookProgressStages progress={{ stages: [
      { key: 'sources', status: 'approved', reason: '已有 3 个可引用资料片段。' },
      { key: 'blueprint', status: 'ready', reason: '可以创建蓝图草稿。' },
      { key: 'sample', status: 'blocked', reason: '需要先批准蓝图。' },
      { key: 'chapters', status: 'in_progress', reason: '已有章节位置。' },
      { key: 'verification', status: 'unavailable', reason: '尚未接入。' },
      { key: 'learning', status: 'unavailable', reason: '尚未接入。' },
		{ key: 'publication', status: 'in_progress', reason: '已有单章个人学习包；整书出版尚未完成。' },
    ] }} />)
    expect(screen.getByText('已有 3 个可引用资料片段。')).toBeTruthy()
	expect(screen.getAllByText('暂不可用')).toHaveLength(2)
    expect(screen.getByText('可开始')).toBeTruthy()
  })
})
