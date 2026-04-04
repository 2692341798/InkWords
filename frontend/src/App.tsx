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
          墨言博客助手
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4">
            项目大纲
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
              暂未分析任何项目
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
            placeholder="请粘贴 GitHub 仓库链接 (例如: https://github.com/gin-gonic/gin)"
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
            {store.isAnalyzing ? '分析中...' : '分析仓库'}
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-8">
          {!store.outline && !store.isAnalyzing && (
            <div className="h-full flex flex-col items-center justify-center text-zinc-400">
              <GitBranch className="w-12 h-12 mb-4 opacity-20" />
              <p>在上方输入 Git 仓库链接以开始</p>
            </div>
          )}

          {store.isAnalyzing && (
            <div className="h-full flex flex-col items-center justify-center text-zinc-500">
              <Loader2 className="w-8 h-8 mb-4 animate-spin text-indigo-600" />
              <p>正在克隆并分析仓库结构...</p>
              <p className="text-xs mt-2 text-zinc-400">这可能需要几秒钟的时间</p>
            </div>
          )}

          {store.outline && !store.isAnalyzing && (
            <div className="max-w-3xl mx-auto">
              <h2 className="text-2xl font-bold text-zinc-800 mb-2">项目大纲已生成</h2>
              <p className="text-zinc-500 mb-8">
                我们已经分析了您的代码库并生成了以下系列博客大纲。
                点击“开始生成”以编写该系列博客。
              </p>

              <div className="bg-indigo-50 border border-indigo-100 rounded-xl p-6 mb-8">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-indigo-900">准备生成</h3>
                    <p className="text-sm text-indigo-700 mt-1">
                      系统将并发生成 {store.outline.length} 篇博客章节。
                    </p>
                  </div>
                  <Button 
                    onClick={handleGenerate} 
                    disabled={store.isGenerating}
                    className="bg-indigo-600 text-white hover:bg-indigo-700"
                  >
                    {store.isGenerating ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
                    {store.isGenerating ? '生成中...' : '开始生成'}
                  </Button>
                </div>
              </div>

              {store.isGenerating && (
                <div className="bg-zinc-50 rounded-xl border border-zinc-200 p-8 text-center">
                  <Loader2 className="w-8 h-8 animate-spin text-indigo-600 mx-auto mb-4" />
                  <h3 className="font-medium text-zinc-800">正在生成您的系列博客</h3>
                  <p className="text-sm text-zinc-500 mt-2">
                    请在左侧边栏查看每个章节的实时生成进度。
                    生成的内容将自动保存到数据库中。
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
