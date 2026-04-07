import { useEffect, useState } from 'react'
import { useBlogStore } from '@/store/blogStore'
import { Sidebar } from '@/components/Sidebar'
import { Generator } from '@/components/Generator'
import { Editor } from '@/components/Editor'
import { Login } from '@/components/Login'
import { Dashboard } from '@/components/Dashboard'

function App() {
  const { selectedBlog, currentView } = useBlogStore()
  const [isAuthenticated] = useState<boolean>(() => {
    const urlParams = new URLSearchParams(window.location.search)
    const token = urlParams.get('token')
    if (token) {
      localStorage.setItem('token', token)
      return true
    }
    return !!localStorage.getItem('token')
  })

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    if (urlParams.has('token')) {
      window.history.replaceState({}, document.title, window.location.pathname)
    }
  }, [])

  if (!isAuthenticated) {
    return <Login />
  }

  return (
    <div className="h-screen overflow-hidden bg-zinc-50 flex print:bg-white print:block print:h-auto print:overflow-visible">
      <Sidebar />
      {selectedBlog ? <Editor key={selectedBlog.id} /> : currentView === 'dashboard' ? <Dashboard /> : <Generator />}
    </div>
  )
}

export default App
