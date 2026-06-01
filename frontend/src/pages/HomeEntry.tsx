import { useEffect, useMemo, useState } from 'react'
import { ArrowRight, BookOpen, Clock3, FileText, Sparkles } from 'lucide-react'
import { StepStrip } from '@/components/shared/StepStrip'
import { Button } from '@/components/ui/button'
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

  return (
    <div className="flex-1 overflow-y-auto bg-zinc-50 custom-scrollbar">
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-12">
        <section className="rounded-3xl border border-zinc-200 bg-white px-8 py-10 shadow-sm">
          <div className="space-y-4">
            <span className="inline-flex items-center rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700">
              墨言博客助手 · 工作入口
            </span>
            <div className="space-y-2">
              <h1 className="text-4xl font-bold tracking-tight text-zinc-900">今天你想先完成哪一种任务？</h1>
              <p className="max-w-3xl text-sm leading-6 text-zinc-600">
                这里先帮助你判断现在应该进入哪条路径，再把你送入真实的工作页。首页只保留一个主动作，其余信息全部收敛成支持信息。
              </p>
            </div>
          </div>
        </section>

        <section className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
          <div className="mb-5 flex items-center justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold text-zinc-900">选择工作路径</h2>
              <p className="mt-1 text-sm text-zinc-500">先决定当前目标，再进入真实的页面继续完成后续步骤。</p>
            </div>
            <span className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600">
              {activePath === 'blog' ? '推荐路径' : '内化路径'}
            </span>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <button
              type="button"
              onClick={() => setActivePath('blog')}
              className={`rounded-2xl border p-6 text-left transition ${
                activePath === 'blog'
                  ? 'border-zinc-900 bg-zinc-50 shadow-sm'
                  : 'border-zinc-200 bg-white hover:border-zinc-300'
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-xl font-semibold text-zinc-900">生成博客</h3>
                <span className="rounded-full bg-indigo-50 px-3 py-1 text-xs font-medium text-indigo-700">推荐</span>
              </div>
              <p className="mt-3 text-sm leading-6 text-zinc-600">
                从 GitHub 仓库或本地文档开始，先做解析，再进入创作场景和大纲确认。
              </p>
            </button>

            <button
              type="button"
              onClick={() => setActivePath('review')}
              className={`rounded-2xl border p-6 text-left transition ${
                activePath === 'review'
                  ? 'border-zinc-900 bg-zinc-50 shadow-sm'
                  : 'border-zinc-200 bg-white hover:border-zinc-300'
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-xl font-semibold text-zinc-900">知识复习</h3>
                <span className="rounded-full bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-700">内化</span>
              </div>
              <p className="mt-3 text-sm leading-6 text-zinc-600">
                从知识库中抽取重点内容，先选入口，再进入真实的复述与反馈会话。
              </p>
            </button>
          </div>
        </section>

        <div className="grid gap-8 lg:grid-cols-[minmax(0,1.35fr)_360px]">
          <section className="space-y-6">
            <article className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
              <StepStrip
                title="流程预览"
                description={viewState.description}
                steps={viewState.steps}
                variant="preview"
              />
            </article>

            <article className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  {activePath === 'blog' ? (
                    <FileText className="h-5 w-5 text-indigo-600" />
                  ) : (
                    <Sparkles className="h-5 w-5 text-emerald-600" />
                  )}
                  <h2 className="text-lg font-semibold text-zinc-900">{viewState.title}</h2>
                </div>
                <p className="text-sm leading-6 text-zinc-600">{viewState.recommendation}</p>
              </div>
              <div className="mt-5 rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-4 text-sm leading-6 text-zinc-600">
                {activePath === 'blog'
                  ? '点击后会进入真实的博客生成页，在那里继续完成“选择来源 -> 配置解析 -> 确认大纲 -> 开始生成”的逐步流程。'
                  : '点击后会进入真实的知识复习页，在那里继续完成“选择入口 -> 开始会话 -> 获得反馈”的逐步流程。'}
              </div>
              <div className="mt-5">
                <Button
                  className="gap-2"
                  onClick={() => {
                    if (viewState.targetView === 'knowledge-review') {
                      reviewStore.setShouldResumeSessionOnOpen(false)
                    }
                    setCurrentView(viewState.targetView)
                  }}
                >
                  {viewState.ctaLabel}
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
            </article>
          </section>

          <aside className="space-y-6">
            <article className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <Clock3 className="h-5 w-5 text-zinc-600" />
                <h2 className="text-lg font-semibold text-zinc-900">继续上次任务</h2>
              </div>
              <p className="mt-4 text-sm font-medium text-zinc-900">{resumeCard.title}</p>
              <p className="mt-2 text-sm leading-6 text-zinc-600">{resumeCard.description}</p>
              <div className="mt-4">
                <Button variant="outline" onClick={resumeCard.onAction}>
                  {resumeCard.actionLabel}
                </Button>
              </div>
            </article>

            <article className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <BookOpen className="h-5 w-5 text-zinc-600" />
                <h2 className="text-lg font-semibold text-zinc-900">最近博客记录</h2>
              </div>
              <div className="mt-4 space-y-3">
                {recentBlogs.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-5 text-sm text-zinc-500">
                    还没有博客记录，先进入博客生成开始第一条工作流。
                  </div>
                ) : (
                  recentBlogs.map((blog) => (
                    <article key={blog.id} className="rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-4">
                      <p className="text-sm font-medium text-zinc-900">{blog.title || '无标题博客'}</p>
                      <p className="mt-1 text-xs text-zinc-500">
                        {blog.parent_id ? '系列章节' : '独立文章'} · 最近更新：{new Date(blog.updated_at).toLocaleString()}
                      </p>
                    </article>
                  ))
                )}
              </div>
            </article>

            <article className="rounded-3xl border border-zinc-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <Sparkles className="h-5 w-5 text-zinc-600" />
                <h2 className="text-lg font-semibold text-zinc-900">最近复习记录</h2>
              </div>
              <div className="mt-4 space-y-3">
                {reviewStore.isLoadingHistory && recentReviews.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-5 text-sm text-zinc-500">
                    正在加载复习记录...
                  </div>
                ) : recentReviews.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 px-4 py-5 text-sm text-zinc-500">
                    还没有复习记录，等你完成第一轮知识漫游后会显示在这里。
                  </div>
                ) : (
                  recentReviews.map((item) => (
                    <article key={item.session_id} className="rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-4">
                      <p className="text-sm font-medium text-zinc-900">{item.title}</p>
                      <p className="mt-1 text-xs text-zinc-500">
                        {item.mode === 'detailed_qa' ? '细致提问' : '轻提示复述'} · {item.reviewed_at ? new Date(item.reviewed_at).toLocaleString() : '暂无时间'}
                      </p>
                    </article>
                  ))
                )}
              </div>
            </article>
          </aside>
        </div>
      </div>
    </div>
  )
}
