import { useEffect, useSyncExternalStore } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { Sidebar } from '@/components/Sidebar'
import { HomeEntry } from '@/pages/HomeEntry'
import { Generator } from '@/pages/Generator'
import { Editor } from '@/pages/Editor'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
import { KnowledgeReview } from '@/pages/KnowledgeReview'
import { Toaster } from '@/components/ui/sonner'
import { authTokenStore } from '@/lib/authTokenStore'

function App() {
  const { selectedBlog, currentView } = useBlogStore()
  const token = useSyncExternalStore(authTokenStore.subscribe, authTokenStore.getSnapshot, authTokenStore.getServerSnapshot)

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    const tokenFromUrl = urlParams.get('token')
    if (tokenFromUrl) {
      authTokenStore.setToken(tokenFromUrl)
      window.history.replaceState({}, document.title, window.location.pathname)
    }
  }, [])

  if (!token) {
    return <Login />
  }

  return (
    <div className="h-screen overflow-hidden bg-zinc-50 flex print:bg-white print:block print:h-auto print:overflow-visible">
      <Sidebar />
      {selectedBlog ? (
        <Editor key={selectedBlog.id} />
      ) : currentView === 'home-entry' ? (
        <HomeEntry />
      ) : currentView === 'dashboard' ? (
        <Dashboard />
      ) : currentView === 'knowledge-review' ? (
        <KnowledgeReview />
      ) : (
        <Generator />
      )}
      <Toaster />
    </div>
  )
}

export default App
