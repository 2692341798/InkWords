import { useEffect, useMemo, useState } from 'react'
import { ArrowRight, BookOpen, Clock3, FileText, Sparkles } from 'lucide-react'
import { StepStrip } from '@/components/shared/StepStrip'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import { useBlogStore } from '@/store/blogStore'
import { useReviewStore } from '@/store/reviewStore'
import { getHomeEntryViewState, type HomeEntryPath } from './homeEntryViewState'

// Why: 首页入口只做“选择路径 -> 预览流程 -> 进入真实工作页”，
// 把真正的业务执行继续留在 Generator / KnowledgeReview，避免入口页重新堆砌复杂逻辑。
export function HomeEntry() {
  const [activePath, setActivePath] = useState<HomeEntryPath>('blog')
  const { blogs, fetchBlogs, setCurrentView } = useBlogStore()
  const reviewStore = useReviewStore()
  const viewState = getHomeEntryViewState(activePath)

  useEffect(() => {
    if (blogs.length === 0) {
      void fetchBlogs()
    }
  }, [blogs.length, fetchBlogs])

  useEffect(() => {
    if (!reviewStore.recommendationCard && !reviewStore.isLoadingRecommendation) {
      void reviewStore.loadRecommendation()
    }
    if (reviewStore.historyItems.length === 0 && !reviewStore.isLoadingHistory) {
      void reviewStore.loadHistory(3)
    }
  }, [
    reviewStore,
    reviewStore.historyItems.length,
  ])

  const recentBlogs = useMemo(() => blogs.slice(0, 3), [blogs])
  const recentReviews = useMemo(() => reviewStore.historyItems.slice(0, 3), [reviewStore.historyItems])
  const resumableReviewSession =
    reviewStore.currentSession && reviewStore.currentSession.status !== 'completed'
      ? reviewStore.currentSession
      : null

  const resumeCard = resumableReviewSession
    ? {
        title: resumableReviewSession.title,
        description: `你上次停在知识复习，会话仍可继续，当前模式为 ${
          resumableReviewSession.mode === 'detailed_qa' ? '细致提问' : '轻提示复述'
        }。`,
        actionLabel: '继续知识复习',
        onAction: () => {
          reviewStore.setShouldResumeSessionOnOpen(true)
          setCurrentView('knowledge-review')
        },
      }
    : recentBlogs.length > 0
      ? {
          title: recentBlogs[0].title || '最近博客任务',
          description: '从最近处理过的博客任务继续，能最快回到当前工作上下文。',
          actionLabel: '进入博客生成',
          onAction: () => setCurrentView('generator'),
        }
      : {
          title: '开始新的工作流',
          description: '先从推荐路径进入，再根据当前目标切换到生成或复习。',
          actionLabel: viewState.ctaLabel,
          onAction: () => setCurrentView(viewState.targetView),
        }

  const enterActivePath = () => {
    if (viewState.targetView === 'knowledge-review') {
      reviewStore.setShouldResumeSessionOnOpen(false)
    }
    setCurrentView(viewState.targetView)
  }

  return (
    <PageShell wide>
      <PageHeader
        title="从资料到博客，从博客到复习"
        description="墨言会先帮你选定今天的主路径，再把注意力收敛到当前唯一需要完成的动作。"
        meta={<StatusPill tone="brand">墨言博客助手</StatusPill>}
        actions={
          <Button variant="outline" className="gap-2" onClick={resumeCard.onAction}>
            <Clock3 className="h-4 w-4" />
            {resumeCard.actionLabel}
          </Button>
        }
      />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
        <section className="space-y-6">
          <Panel className="p-6">
            <SectionHeader
              eyebrow="工作路径"
              title="今天先完成哪一种任务？"
              description="只展开一个主流程，其他信息作为辅助上下文保留在下方。"
              action={<StatusPill>{activePath === 'blog' ? '推荐路径' : '内化路径'}</StatusPill>}
            />

            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <button
                type="button"
                onClick={() => setActivePath('blog')}
                className={`choice-tile ${activePath === 'blog' ? 'choice-tile-active' : 'choice-tile-muted'}`}
                aria-pressed={activePath === 'blog'}
              >
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <FileText className="h-5 w-5 text-[var(--brand)]" />
                    <h3 className="text-base font-semibold text-foreground">生成博客</h3>
                  </div>
                  <StatusPill tone="brand">推荐</StatusPill>
                </div>
                <p className="mt-3 text-sm leading-6 text-muted-foreground">
                  从 GitHub 仓库或本地文档开始，生成可编辑的结构化技术博客。
                </p>
              </button>

              <button
                type="button"
                onClick={() => setActivePath('review')}
                className={`choice-tile ${activePath === 'review' ? 'choice-tile-active' : 'choice-tile-muted'}`}
                aria-pressed={activePath === 'review'}
              >
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <Sparkles className="h-5 w-5 text-[var(--success)]" />
                    <h3 className="text-base font-semibold text-foreground">知识复习</h3>
                  </div>
                  <StatusPill tone="success">内化</StatusPill>
                </div>
                <p className="mt-3 text-sm leading-6 text-muted-foreground">
                  从知识库中抽取重点内容，进入复述、提示与反馈会话。
                </p>
              </button>
            </div>
          </Panel>

          <Panel className="p-6">
            <StepStrip
              title="流程预览"
              description={viewState.description}
              steps={viewState.steps}
              variant="preview"
            />
          </Panel>

          <Panel className="p-6">
            <SectionHeader
              title={viewState.title}
              description={viewState.recommendation}
              action={
                <Button className="gap-2" onClick={enterActivePath}>
                  {viewState.ctaLabel}
                  <ArrowRight className="h-4 w-4" />
                </Button>
              }
            />
            <div className="mt-5 surface-inset px-4 py-4 text-sm leading-6 text-muted-foreground">
              {activePath === 'blog'
                ? '下一步会进入博客生成工作台，继续完成来源选择、解析配置、大纲确认与生成。'
                : '下一步会进入知识复习工作台，继续完成入口选择、会话开始、提示与反馈。'}
            </div>
          </Panel>
        </section>

        <aside className="summary-rail">
          <SectionHeader eyebrow="当前摘要" title="本次工作" description="右侧只保留影响下一步决策的信息。" />
          <div className="mt-5 space-y-3">
            <div className="summary-row">
              <p className="text-xs text-muted-foreground">已选择</p>
              <p className="mt-1 text-sm font-medium text-foreground">{viewState.title}</p>
            </div>
            <div className="summary-row">
              <p className="text-xs text-muted-foreground">下一步</p>
              <p className="mt-1 text-sm font-medium text-foreground">{viewState.ctaLabel}</p>
            </div>
            <div className="summary-row">
              <p className="text-xs text-muted-foreground">可继续</p>
              <p className="mt-1 text-sm font-medium text-foreground">{resumeCard.title}</p>
              <p className="mt-2 text-xs leading-5 text-muted-foreground">{resumeCard.description}</p>
            </div>
          </div>
        </aside>
      </div>

      <section className="grid gap-6 lg:grid-cols-2">
        <Panel className="p-6">
          <SectionHeader title="最近博客" description="最多展示最近 3 条，作为恢复上下文而不是主流程入口。" />
          <div className="mt-5 space-y-3">
            {recentBlogs.length === 0 ? (
              <div className="surface-inset px-4 py-5 text-sm text-muted-foreground">
                还没有博客记录，先进入博客生成开始第一条工作流。
              </div>
            ) : (
              recentBlogs.map((blog) => (
                <article key={blog.id} className="surface-inset px-4 py-4 transition-colors hover:bg-secondary/70">
                  <div className="flex items-start gap-3">
                    <BookOpen className="mt-0.5 h-4 w-4 text-muted-foreground" />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">{blog.title || '无标题博客'}</p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {blog.parent_id ? '系列章节' : '独立文章'} · 最近更新：{new Date(blog.updated_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </article>
              ))
            )}
          </div>
        </Panel>

        <Panel className="p-6">
          <SectionHeader title="最近复习" description="复习记录保持辅助地位，避免和当前工作路径抢焦点。" />
          <div className="mt-5 space-y-3">
            {reviewStore.isLoadingHistory && recentReviews.length === 0 ? (
              <div className="surface-inset px-4 py-5 text-sm text-muted-foreground">正在加载复习记录...</div>
            ) : recentReviews.length === 0 ? (
              <div className="surface-inset px-4 py-5 text-sm text-muted-foreground">
                还没有复习记录，完成第一轮知识漫游后会显示在这里。
              </div>
            ) : (
              recentReviews.map((item) => (
                <article key={item.session_id} className="surface-inset px-4 py-4 transition-colors hover:bg-secondary/70">
                  <div className="flex items-start gap-3">
                    <Sparkles className="mt-0.5 h-4 w-4 text-muted-foreground" />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-foreground">{item.title}</p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {item.mode === 'detailed_qa' ? '细致提问' : '轻提示复述'} · {item.reviewed_at ? new Date(item.reviewed_at).toLocaleString() : '暂无时间'}
                      </p>
                    </div>
                  </div>
                </article>
              ))
            )}
          </div>
        </Panel>
      </section>
    </PageShell>
  )
}
