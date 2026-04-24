import { useState, useRef, useEffect } from 'react'
import type { DragEvent, ChangeEvent } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { Button } from '@/components/ui/button'
import { Loader2, GitBranch, UploadCloud, ArrowUp, ArrowDown, Trash2, Plus, ChevronDown, ChevronUp } from 'lucide-react'

import { MarkdownEngine } from '@/components/MarkdownEngine'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

export function Generator() {
  const store = useStreamStore()
  const { analyzeGit, parseFile, generateSeries, generateSingle, stopAnalyzing, stopGenerating } = useBlogStream()
  const [gitUrl, setGitUrl] = useState('')
  const [subDir, setSubDir] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [isDragging, setIsDragging] = useState(false)
  const [analyzingType, setAnalyzingType] = useState<'git' | 'file'>('git')
  const [isOutlineExpanded, setIsOutlineExpanded] = useState(true)
  const [showChapterDeleteConfirm, setShowChapterDeleteConfirm] = useState<number | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (store.isGenerating) {
      setIsOutlineExpanded(false)
    } else {
      setIsOutlineExpanded(true)
    }
  }, [store.isGenerating, store.isAnalyzing])

  const handleAnalyze = async () => {
    if (!gitUrl) return
    setAnalyzingType('git')
    try {
      await analyzeGit(gitUrl, subDir)
    } catch (err) {
      setGitUrl('')
    }
  }

  const handleGenerate = () => {
    if (store.sourceType === 'file') {
      if (store.sourceContent) {
        generateSingle(store.sourceContent)
      }
    } else {
      generateSeries()
    }
  }

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = async (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) {
      setAnalyzingType('file')
      try {
        await parseFile(file)
      } catch (err) {
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }
  }

  const handleFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setAnalyzingType('file')
      try {
        await parseFile(file)
      } catch (err) {
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }
  }

  return (
    <div className="flex-1 flex flex-col bg-white">
      <div className="border-b border-zinc-200 px-6 py-4">
        <div className="flex items-center gap-4">
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

        <div className="mt-2">
          <button
            type="button"
            onClick={() => setShowAdvanced(v => !v)}
            disabled={store.isAnalyzing || store.isGenerating}
            className="flex items-center text-sm text-zinc-500 hover:text-zinc-700 disabled:opacity-50 disabled:hover:text-zinc-500"
          >
            高级选项
            {showAdvanced ? <ChevronUp className="w-4 h-4 ml-1" /> : <ChevronDown className="w-4 h-4 ml-1" />}
          </button>

          {showAdvanced && (
            <div className="mt-3 p-4 bg-zinc-50 rounded-lg border border-zinc-100">
              <label className="block text-sm font-medium text-zinc-700 mb-1">指定解析子目录 (可选)</label>
              <input
                type="text"
                placeholder="如：src/net/http"
                className="w-full p-2 text-sm border border-zinc-200 rounded-md focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none"
                value={subDir}
                onChange={(e) => setSubDir(e.target.value)}
                disabled={store.isAnalyzing || store.isGenerating}
              />
              <p className="text-xs text-zinc-500 mt-2">针对特大型仓库，建议指定具体模块路径以加速解析并避免超限。</p>
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-8">
        {!store.outline && !store.isAnalyzing && (
          <div className="h-full flex flex-col items-center justify-center text-zinc-400">
            <div
              className={`w-full max-w-2xl border-2 border-dashed rounded-xl p-12 text-center transition-colors cursor-pointer
                ${isDragging ? 'border-indigo-500 bg-indigo-50/50' : 'border-zinc-200 hover:border-zinc-300 hover:bg-zinc-50'}`}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
            >
              <UploadCloud className="w-12 h-12 mx-auto mb-4 text-zinc-400" />
              <h3 className="text-lg font-medium text-zinc-900 mb-2">拖拽或点击上传文件</h3>
              <p className="text-sm text-zinc-500 mb-6">支持 PDF, Word (.docx), Markdown (.md) 格式</p>
              <input
                type="file"
                ref={fileInputRef}
                className="hidden"
                accept=".pdf,.docx,.md,.txt"
                onChange={handleFileChange}
              />
            </div>

            <div className="flex items-center w-full max-w-2xl my-8">
              <div className="flex-1 border-t border-zinc-200"></div>
              <span className="px-4 text-sm text-zinc-400">或</span>
              <div className="flex-1 border-t border-zinc-200"></div>
            </div>

            <div className="text-center">
              <GitBranch className="w-8 h-8 mx-auto mb-3 opacity-20" />
              <p>在上方输入 Git 仓库链接以开始</p>
            </div>
          </div>
        )}

        {store.isAnalyzing && (
          <div className="h-full flex flex-col items-center justify-center text-zinc-500">
            <Loader2 className="w-8 h-8 mb-6 animate-spin text-indigo-600" />
            <div className="space-y-4 max-w-3xl w-full">
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${store.analysisStep >= 0 ? 'bg-indigo-600' : 'bg-zinc-200'}`}></div>
                <span className={store.analysisStep >= 0 ? 'text-zinc-800 font-medium' : 'text-zinc-400'}>
                  {analyzingType === 'file' ? '读取并提取文件文本...' : (store.analysisStep === 0 ? store.analysisMessage : '正在克隆并拉取仓库...')}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${store.analysisStep >= 1 ? 'bg-indigo-600' : 'bg-zinc-200'}`}></div>
                <span className={store.analysisStep >= 1 ? 'text-zinc-800 font-medium' : 'text-zinc-400'}>
                  {analyzingType === 'file' ? '解析文件内容结构...' : (store.analysisStep === 1 ? store.analysisMessage : '分析仓库源码与结构...')}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${store.analysisStep >= 2 ? 'bg-indigo-600' : 'bg-zinc-200'}`}></div>
                <div className="flex-1 flex flex-col">
                  <span className={store.analysisStep >= 2 ? 'text-zinc-800 font-medium' : 'text-zinc-400'}>
                    {analyzingType === 'file' ? '准备进行生成任务...' : (store.analysisStep === 2 ? store.analysisMessage : '并发分析代码分块...')}
                  </span>
                  {store.analysisStep === 2 && Object.keys(store.workers).length > 0 && (
                    <div className="mt-4 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
                      {Object.keys(store.workers).map(key => {
                        const workerId = Number(key);
                        const worker = store.workers[workerId];
                        if (!worker) return (
                          <div key={workerId} className="h-20 bg-zinc-50 border border-zinc-100 rounded-lg flex items-center justify-center text-zinc-300 text-xs">
                            Worker {workerId + 1} 闲置
                          </div>
                        );
                        
                        const isAnalyzing = worker.status === 'chunk_analyzing';
                        const isFailed = worker.status === 'chunk_failed';
                        const isDone = worker.status === 'chunk_done';
                        
                        return (
                          <div key={workerId} className={`p-3 rounded-lg border text-sm transition-all duration-300 ${
                            isAnalyzing ? 'bg-indigo-50 border-indigo-200 shadow-[0_0_10px_rgba(99,102,241,0.2)] animate-pulse' :
                            isFailed ? 'bg-orange-50 border-orange-200' :
                            isDone ? 'bg-green-50 border-green-200' : 'bg-zinc-50 border-zinc-200'
                          }`}>
                            <div className="flex justify-between items-center mb-1">
                              <span className="text-xs font-medium text-zinc-500">Worker {workerId + 1}</span>
                              <span className={
                                isFailed ? 'text-orange-500' : 
                                worker.status === 'chunk_failed_final' ? 'text-red-500' : 
                                isDone ? 'text-green-500' : 'text-indigo-500 font-medium'
                              }>
                                {isFailed ? `重试 (${worker.attempt}/3)` :
                                 worker.status === 'chunk_failed_final' ? '跳过' :
                                 isDone ? '完成' : '分析中'}
                              </span>
                            </div>
                            <div className="truncate font-mono text-xs text-zinc-600" title={worker.dir}>
                              {worker.dir}
                            </div>
                            <div className="text-xs text-zinc-400 mt-1">
                              {worker.index} / {worker.total}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${store.analysisStep >= 3 ? 'bg-indigo-600' : 'bg-zinc-200'}`}></div>
                <span className={store.analysisStep >= 3 ? 'text-zinc-800 font-medium' : 'text-zinc-400'}>
                  {analyzingType === 'file' ? '读取并渲染内容...' : (store.analysisStep === 3 ? store.analysisMessage : '生成项目全局大纲...')}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <div className={`w-2 h-2 rounded-full ${store.analysisStep >= 4 ? 'bg-indigo-600' : 'bg-zinc-200'}`}></div>
                <span className={store.analysisStep >= 4 ? 'text-zinc-800 font-medium' : 'text-zinc-400'}>
                  {analyzingType === 'file' ? '完成' : (store.analysisStep === 4 ? store.analysisMessage : '正在完成最后处理...')}
                </span>
              </div>
            </div>
            
            <div className="mt-8 flex justify-center w-full">
              <Button 
                onClick={() => {
                  stopAnalyzing()
                  setGitUrl('')
                }} 
                variant="outline"
                className="text-zinc-500 hover:text-zinc-700"
              >
                停止分析
              </Button>
            </div>
          </div>
        )}

        {store.outline && !store.isAnalyzing && (
          <div className="max-w-3xl mx-auto">
            <h2 className="text-2xl font-bold text-zinc-800 mb-2">
              {(() => {
                const allCompleted = store.outline.length > 0 && store.outline.every(ch => store.chapterStatus[ch.sort] === 'completed');
                if (allCompleted) return '系列博客生成完毕';
                return store.sourceType === 'file' ? '文件解析成功' : '项目大纲已生成';
              })()}
            </h2>
            <p className="text-zinc-500 mb-8">
              {(() => {
                const allCompleted = store.outline.length > 0 && store.outline.every(ch => store.chapterStatus[ch.sort] === 'completed');
                if (allCompleted) return '所有的章节已经成功生成并保存到数据库。您可以在左侧边栏点击生成的章节查看完整内容。';
                return store.sourceType === 'file'
                  ? '我们已经成功提取了您的文件内容。点击“开始生成”以编写单篇博客。'
                  : '我们已经分析了您的代码库并生成了以下系列博客大纲。点击“开始生成”以编写该系列博客。';
              })()}
            </p>

            {(() => {
              const allCompleted = store.outline.length > 0 && store.outline.every(ch => store.chapterStatus[ch.sort] === 'completed');
              if (allCompleted) return null;

              const hasCompleted = store.outline.length > 0 && store.outline.some(ch => store.chapterStatus[ch.sort] === 'completed');
              const isResume = hasCompleted && !allCompleted;
              const visibleOutline = store.outline.filter(ch => store.chapterStatus[ch.sort] !== 'completed');

              return (
                <>
                  {store.sourceType !== 'file' && visibleOutline.length > 0 && (
                    <div className="mb-8">
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-3">
                          <h3 className="text-lg font-semibold text-zinc-800">
                            {isResume ? '待生成章节大纲' : '系列博客大纲'}
                          </h3>
                          <button
                            onClick={() => setIsOutlineExpanded(!isOutlineExpanded)}
                            className="p-1 hover:bg-zinc-100 rounded text-zinc-500 transition-colors"
                            title={isOutlineExpanded ? "折叠大纲" : "展开大纲"}
                          >
                            {isOutlineExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                          </button>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm text-zinc-500">系列标题:</span>
                          <input
                            type="text"
                            value={store.seriesTitle}
                            onChange={(e) => store.setSeriesTitle(e.target.value)}
                            placeholder="请输入系列标题"
                            className="px-3 py-1.5 bg-zinc-50 border border-zinc-200 rounded-md text-sm w-64 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                            disabled={store.isGenerating}
                          />
                        </div>
                      </div>
                      {isOutlineExpanded && (
                        <div className="space-y-4 max-h-[50vh] overflow-y-auto custom-scrollbar pr-2">
                          {visibleOutline.map((ch, index) => (
                          <div key={ch.sort} className="p-4 bg-white rounded-xl border border-zinc-200 shadow-sm hover:border-indigo-200 transition-colors group">
                            <div className="flex items-start gap-3 mb-3">
                              <div className="w-6 h-6 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center text-sm font-semibold shrink-0 mt-1">
                                {ch.sort}
                              </div>
                              <div className="flex-1 min-w-0">
                                <input
                                  type="text"
                                  value={ch.title}
                                  onChange={(e) => store.updateChapter(ch.sort, 'title', e.target.value)}
                                  className="w-full font-medium text-zinc-900 border-none bg-transparent focus:outline-none focus:ring-0 p-0 text-base"
                                  placeholder="章节标题"
                                  disabled={store.isGenerating}
                                />
                              </div>
                              <div className="flex items-center gap-1 shrink-0 opacity-0 hover:opacity-100 focus-within:opacity-100 group-hover:opacity-100 transition-opacity">
                                <button 
                                  onClick={() => store.moveChapter(ch.sort, 'up')}
                                  disabled={index === 0 || store.isGenerating || isResume}
                                  className="p-1.5 text-zinc-400 hover:text-indigo-600 hover:bg-indigo-50 rounded disabled:opacity-30"
                                >
                                  <ArrowUp className="w-4 h-4" />
                                </button>
                                <button 
                                  onClick={() => store.moveChapter(ch.sort, 'down')}
                                  disabled={index === visibleOutline.length - 1 || store.isGenerating || isResume}
                                  className="p-1.5 text-zinc-400 hover:text-indigo-600 hover:bg-indigo-50 rounded disabled:opacity-30"
                                >
                                  <ArrowDown className="w-4 h-4" />
                                </button>
                                <button 
                                  type="button"
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    if (e.detail > 1) return; // Prevent double click
                                    setShowChapterDeleteConfirm(ch.sort);
                                  }}
                                  disabled={store.outline!.length <= 1 || store.isGenerating || isResume}
                                  className="p-1.5 text-zinc-400 hover:text-red-600 hover:bg-red-50 rounded disabled:opacity-30"
                                  title="删除章节"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </button>
                              </div>
                            </div>
                            <div className="pl-9">
                              <textarea
                                value={ch.summary}
                                onChange={(e) => store.updateChapter(ch.sort, 'summary', e.target.value)}
                                className="w-full text-sm text-zinc-600 bg-zinc-50 border border-transparent hover:border-zinc-200 focus:border-indigo-300 focus:bg-white rounded-md p-2 resize-y min-h-[60px] focus:outline-none transition-colors"
                                placeholder="章节内容摘要或要点..."
                                disabled={store.isGenerating}
                              />
                            </div>
                          </div>
                        ))}
                        <button
                          onClick={() => store.addChapter()}
                          disabled={store.isGenerating || isResume}
                          className="w-full py-3 border-2 border-dashed border-zinc-200 rounded-xl text-zinc-500 hover:text-indigo-600 hover:border-indigo-300 hover:bg-indigo-50/50 transition-all flex items-center justify-center gap-2 font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          <Plus className="w-4 h-4" />
                          添加新章节
                        </button>
                      </div>
                      )}
                    </div>
                  )}

                  <div className="bg-indigo-50 border border-indigo-100 rounded-xl p-6 mb-8">
                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="font-semibold text-indigo-900">{isResume ? '继续生成' : '准备生成'}</h3>
                        <p className="text-sm text-indigo-700 mt-1">
                          {store.sourceType === 'file'
                            ? '系统将根据文件内容生成一篇详细的技术博客。'
                            : `系统将并发生成 ${visibleOutline.length} 篇博客章节。`}
                        </p>
                      </div>
                      <div className="flex items-center gap-3">
                        {store.isGenerating && (
                          <Button 
                            onClick={stopGenerating} 
                            variant="outline"
                            className="text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200"
                          >
                            停止生成
                          </Button>
                        )}
                        <Button 
                          onClick={handleGenerate} 
                          disabled={store.isGenerating}
                          className="bg-indigo-600 text-white hover:bg-indigo-700"
                        >
                          {store.isGenerating ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
                          {store.isGenerating ? '生成中...' : (isResume ? '继续生成' : '开始生成')}
                        </Button>
                      </div>
                    </div>
                  </div>
                </>
              );
            })()}

            {store.isGenerating && (
              <div className="bg-zinc-50 rounded-xl border border-zinc-200 p-8">
                <div className="space-y-6">
                  <div className="flex items-center gap-4 border-b border-zinc-200 pb-4">
                    <div className="flex items-center text-indigo-600">
                      <Loader2 className="w-5 h-5 animate-spin mr-2" />
                      <span className="font-medium">
                        {store.sourceType === 'file' 
                          ? 'AI 正在流式写作中...' 
                          : 'AI 正在并发生成章节...'
                        }
                      </span>
                    </div>
                    <div className="flex-1"></div>
                    {store.sourceType === 'file' && (
                      <span className="text-xs text-zinc-500">{store.generatedContent.length} 字符</span>
                    )}
                  </div>

                  {store.sourceType === 'file' ? (
                    <div className="prose prose-zinc max-w-none text-left max-h-[500px] overflow-y-auto">
                      <MarkdownEngine content={store.generatedContent || '正在构思文章结构...'} />
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-h-[600px] overflow-y-auto pr-2 scrollbar-thin scrollbar-thumb-zinc-300 scrollbar-track-transparent">
                      {store.outline?.filter(ch => store.chapterStatus[ch.sort] !== 'pending').map((ch) => {
                        const status = store.chapterStatus[ch.sort];
                        const content = store.chapterContents[ch.sort] || '';
                        const isGenerating = status === 'generating';
                        const isError = status === 'error';
                        const isCompleted = status === 'completed';

                        if (isCompleted) {
                          return (
                            <div key={ch.sort} className="p-4 flex flex-col justify-center h-24 rounded-xl border transition-all duration-300 bg-green-50/20 border-green-200 shadow-sm">
                              <div className="flex justify-between items-center mb-2">
                                <span className="text-sm font-semibold text-zinc-700 truncate pr-2" title={ch.title}>
                                  第 {ch.sort} 篇：{ch.title}
                                </span>
                                <span className="text-xs font-medium px-2 py-1 rounded-full whitespace-nowrap bg-green-100 text-green-700">
                                  已完成
                                </span>
                              </div>
                              <div className="text-xs text-zinc-400 flex justify-between items-center">
                                <span>{content.length} 字符</span>
                                <span>可在左侧边栏查看</span>
                              </div>
                            </div>
                          );
                        }

                        return (
                          <div key={ch.sort} className={`p-4 flex flex-col h-[300px] rounded-xl border transition-all duration-300 ${
                            isGenerating ? 'bg-indigo-50/50 border-indigo-200 shadow-[0_0_15px_rgba(99,102,241,0.15)]' :
                            isError ? 'bg-red-50/50 border-red-200' : 'bg-white border-zinc-200'
                          }`}>
                            <div className="flex justify-between items-center mb-3 pb-2 border-b border-black/5 shrink-0">
                              <span className="text-sm font-semibold text-zinc-700 truncate pr-2" title={ch.title}>
                                第 {ch.sort} 篇：{ch.title}
                              </span>
                              <span className={`text-xs font-medium px-2 py-1 rounded-full whitespace-nowrap ${
                                isGenerating ? 'bg-indigo-100 text-indigo-700 animate-pulse' :
                                isError ? 'bg-red-100 text-red-700' : 'bg-zinc-100 text-zinc-600'
                              }`}>
                                {isGenerating ? '生成中...' : isError ? '生成失败' : '等待中'}
                              </span>
                            </div>
                            <div className="flex-1 overflow-y-auto prose prose-sm prose-zinc max-w-none text-left scrollbar-thin text-xs">
                              <MarkdownEngine content={content || (isGenerating ? '正在构思内容...' : '暂无内容')} />
                            </div>
                            <div className="mt-2 pt-2 border-t border-black/5 flex justify-between items-center text-xs text-zinc-400 shrink-0">
                              <span>{content.length} 字符</span>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      <ConfirmDialog
        isOpen={showChapterDeleteConfirm !== null}
        title="确认删除章节"
        message="确定要删除该章节吗？此操作不可恢复。"
        confirmText="确认删除"
        onConfirm={() => {
          if (showChapterDeleteConfirm !== null) {
            store.removeChapter(showChapterDeleteConfirm)
            setShowChapterDeleteConfirm(null)
          }
        }}
        onCancel={() => setShowChapterDeleteConfirm(null)}
        isDestructive={true}
      />
    </div>
  )
}
