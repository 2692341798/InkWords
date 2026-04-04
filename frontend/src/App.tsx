import { useState } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { Button } from '@/components/ui/button'
import { Loader2, GitBranch, CheckCircle2, CircleDashed } from 'lucide-react'

function App() {
  const store = useStreamStore()
  const { analyzeGit, generateSeries } = useBlogStream()
  const [gitUrl, setGitUrl] = useState('')

  const handleAnalyze = () => {
    if (!gitUrl) return
    analyzeGit(gitUrl)
  }

  const handleGenerate = () => {
    generateSeries()
  }

  return (
    <div className="min-h-screen bg-zinc-50 flex">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-zinc-200 flex flex-col">
        <div className="p-4 border-b border-zinc-200 flex items-center gap-2 font-semibold text-lg text-zinc-800">
          <GitBranch className="w-5 h-5 text-indigo-600" />
          InkWords
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4">
            Project Outline
          </div>
          {store.outline ? (
            <div className="space-y-3">
              {store.outline.map((ch) => {
                const status = store.chapterStatus[ch.sort]
                return (
                  <div key={ch.sort} className="p-3 bg-zinc-50 rounded-lg border border-zinc-100 flex items-start gap-3">
                    <div className="mt-0.5">
                      {status === 'completed' ? (
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                      ) : status === 'generating' ? (
                        <Loader2 className="w-4 h-4 text-indigo-500 animate-spin" />
                      ) : (
                        <CircleDashed className="w-4 h-4 text-zinc-400" />
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-medium text-zinc-800">{ch.title}</div>
                      <div className="text-xs text-zinc-500 mt-1 line-clamp-2">{ch.summary}</div>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="text-sm text-zinc-400 text-center py-10">
              No project analyzed yet
            </div>
          )}
        </div>
      </div>

      {/* Main Workspace */}
      <div className="flex-1 flex flex-col bg-white">
        <div className="h-16 border-b border-zinc-200 flex items-center px-6 gap-4">
          <input 
            type="text" 
            className="flex-1 max-w-xl px-4 py-2 bg-zinc-50 border border-zinc-200 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            placeholder="Paste GitHub Repository URL here (e.g. https://github.com/gin-gonic/gin)"
            value={gitUrl}
            onChange={(e) => setGitUrl(e.target.value)}
            disabled={store.isAnalyzing || store.isGenerating}
          />
          <Button 
            onClick={handleAnalyze} 
            disabled={!gitUrl || store.isAnalyzing || store.isGenerating}
            className="bg-zinc-900 text-white hover:bg-zinc-800"
          >
            {store.isAnalyzing ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
            {store.isAnalyzing ? 'Analyzing...' : 'Analyze Repo'}
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-8">
          {!store.outline && !store.isAnalyzing && (
            <div className="h-full flex flex-col items-center justify-center text-zinc-400">
              <GitBranch className="w-12 h-12 mb-4 opacity-20" />
              <p>Enter a Git repository URL above to get started</p>
            </div>
          )}

          {store.isAnalyzing && (
            <div className="h-full flex flex-col items-center justify-center text-zinc-500">
              <Loader2 className="w-8 h-8 mb-4 animate-spin text-indigo-600" />
              <p>Cloning repository and analyzing structure...</p>
              <p className="text-xs mt-2 text-zinc-400">This might take a few seconds</p>
            </div>
          )}

          {store.outline && !store.isAnalyzing && (
            <div className="max-w-3xl mx-auto">
              <h2 className="text-2xl font-bold text-zinc-800 mb-2">Project Outline Ready</h2>
              <p className="text-zinc-500 mb-8">
                We've analyzed your repository and generated the following series outline. 
                Click generate to start writing the blog series.
              </p>

              <div className="bg-indigo-50 border border-indigo-100 rounded-xl p-6 mb-8">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-indigo-900">Ready to Generate</h3>
                    <p className="text-sm text-indigo-700 mt-1">
                      This will concurrently generate {store.outline.length} blog chapters.
                    </p>
                  </div>
                  <Button 
                    onClick={handleGenerate} 
                    disabled={store.isGenerating}
                    className="bg-indigo-600 text-white hover:bg-indigo-700"
                  >
                    {store.isGenerating ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
                    {store.isGenerating ? 'Generating Series...' : 'Start Generation'}
                  </Button>
                </div>
              </div>

              {store.isGenerating && (
                <div className="bg-zinc-50 rounded-xl border border-zinc-200 p-8 text-center">
                  <Loader2 className="w-8 h-8 animate-spin text-indigo-600 mx-auto mb-4" />
                  <h3 className="font-medium text-zinc-800">Generating your blog series</h3>
                  <p className="text-sm text-zinc-500 mt-2">
                    Check the sidebar for real-time progress on each chapter.
                    The generated blogs will be saved to your database automatically.
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
