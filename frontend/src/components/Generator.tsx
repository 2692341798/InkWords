import { useState, useRef } from 'react'
import type { DragEvent, ChangeEvent } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStream } from '@/hooks/useBlogStream'
import { Button } from '@/components/ui/button'
import { Loader2, GitBranch, UploadCloud } from 'lucide-react'

import { MarkdownEngine } from '@/components/MarkdownEngine'

export function Generator() {
  const store = useStreamStore()
  const { analyzeGit, parseFile, generateSeries, generateSingle, stopAnalyzing } = useBlogStream()
  const [gitUrl, setGitUrl] = useState('')
  const [isDragging, setIsDragging] = useState(false)
  const [analyzingType, setAnalyzingType] = useState<'git' | 'file'>('git')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleAnalyze = async () => {
    if (!gitUrl) return
    setAnalyzingType('git')
    try {
      await analyzeGit(gitUrl)
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
            <div className="space-y-4 max-w-sm w-full">
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
                    {analyzingType === 'file' ? '准备进行生成任务...' : (store.analysisStep === 2 ? store.analysisMessage : '评估大模型并生成项目大纲...')}
                  </span>
                  {store.analysisStep === 2 && store.mapReduceProgress && (
                    <div className="mt-2 text-sm text-zinc-500 bg-zinc-50 p-3 rounded-lg border border-zinc-100">
                      <div className="flex justify-between mb-1">
                        <span>正在处理分块 {store.mapReduceProgress.index} / {store.mapReduceProgress.total}</span>
                        <span className={
                          store.mapReduceProgress.status === 'chunk_failed' ? 'text-orange-500' : 
                          store.mapReduceProgress.status === 'chunk_failed_final' ? 'text-red-500' : 
                          store.mapReduceProgress.status === 'chunk_done' ? 'text-green-500' : 'text-indigo-500'
                        }>
                          {store.mapReduceProgress.status === 'chunk_failed' ? `重试中 (${store.mapReduceProgress.attempt}/3)` :
                           store.mapReduceProgress.status === 'chunk_failed_final' ? '已跳过' :
                           store.mapReduceProgress.status === 'chunk_done' ? '完成' : '分析中'}
                        </span>
                      </div>
                      <div className="truncate font-mono text-xs text-zinc-400">{store.mapReduceProgress.dir}</div>
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

              return (
                <div className="bg-indigo-50 border border-indigo-100 rounded-xl p-6 mb-8">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="font-semibold text-indigo-900">准备生成</h3>
                      <p className="text-sm text-indigo-700 mt-1">
                        {store.sourceType === 'file'
                          ? '系统将根据文件内容生成一篇详细的技术博客。'
                          : `系统将并发生成 ${store.outline.length} 篇博客章节。`}
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
                          : (() => {
                              const activeChapter = store.outline?.find(ch => store.chapterStatus[ch.sort] === 'generating')
                              return activeChapter 
                                ? `正在生成第 ${activeChapter.sort} 篇：${activeChapter.title}...` 
                                : '正在准备生成...'
                            })()
                        }
                      </span>
                    </div>
                    <div className="flex-1"></div>
                    <span className="text-xs text-zinc-500">{store.generatedContent.length} 字符</span>
                  </div>
                  <div className="prose prose-zinc max-w-none text-left max-h-[500px] overflow-y-auto">
                    <MarkdownEngine content={store.generatedContent || '正在构思文章结构...'} />
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
