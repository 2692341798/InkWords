import type { StepStripItem } from '@/components/shared/StepStrip'

export type HomeEntryPath = 'blog' | 'review'
type HomeEntryTargetView = 'generator' | 'knowledge-review'

interface HomeEntryViewState {
  activePath: HomeEntryPath
  title: string
  description: string
  recommendation: string
  ctaLabel: string
  targetView: HomeEntryTargetView
  steps: StepStripItem[]
}

/**
 * Why: 首页入口只负责帮助用户决定现在先做什么，
 * 所以路径文案、步骤预览和 CTA 目标应集中在纯函数里统一定义，避免页面和侧边栏各写一套。
 */
export function getHomeEntryViewState(activePath: HomeEntryPath): HomeEntryViewState {
  if (activePath === 'review') {
    return {
      activePath,
      title: '知识复习',
      description: '从知识库中抽取重点内容，进入真实的复习会话页面。',
      recommendation: '当内容已经沉淀下来时，用复习路径把知识从存档变成能力。',
      ctaLabel: '进入知识复习',
      targetView: 'knowledge-review',
      steps: [
        { key: 'entry', title: '选择入口' },
        { key: 'session', title: '开始会话' },
        { key: 'feedback', title: '获得反馈' },
      ],
    }
  }

  return {
    activePath: 'blog',
    title: '生成博客',
    description: '从 GitHub 仓库或本地文档开始，进入真实的博客生成工作流。',
    recommendation: '推荐先从博客生成开始，再回到知识复习做内化。',
    ctaLabel: '进入博客生成',
    targetView: 'generator',
    steps: [
      { key: 'source', title: '选择来源' },
      { key: 'analysis', title: '完成解析' },
      { key: 'outline', title: '确认大纲' },
      { key: 'generate', title: '开始生成' },
    ],
  }
}
