import { lazy, Suspense } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { Sidebar } from '@/components/Sidebar'
import { Toaster } from '@/components/ui/sonner'

const HomeEntry = lazy(() => import('@/pages/HomeEntry').then(({ HomeEntry }) => ({ default: HomeEntry })))
const Generator = lazy(() => import('@/pages/Generator').then(({ Generator }) => ({ default: Generator })))
const Editor = lazy(() => import('@/pages/Editor').then(({ Editor }) => ({ default: Editor })))
const KnowledgeReview = lazy(() => import('@/pages/KnowledgeReview').then(({ KnowledgeReview }) => ({ default: KnowledgeReview })))
const TextbookProjectsPage = lazy(() => import('@/features/textbook-projects/TextbookProjectsPage').then(({ TextbookProjectsPage }) => ({ default: TextbookProjectsPage })))

function PageLoading() {
  return <main className="flex flex-1 items-center justify-center p-6 text-sm text-muted-foreground" aria-busy="true">正在加载工作台…</main>
}

function App() {
  const { selectedBlog, currentView } = useBlogStore()

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background print:block print:h-auto print:overflow-visible print:bg-white lg:flex-row">
      <Sidebar />
      <Suspense fallback={<PageLoading />}>
      {selectedBlog ? (
        <Editor key={selectedBlog.id} />
      ) : currentView === 'home-entry' ? (
        <HomeEntry />
      ) : currentView === 'knowledge-review' ? (
        <KnowledgeReview />
      ) : currentView === 'textbook-projects' ? (
        <TextbookProjectsPage />
      ) : (
        <Generator />
      )}
      </Suspense>
      <Toaster />
    </div>
  )
}

export default App
