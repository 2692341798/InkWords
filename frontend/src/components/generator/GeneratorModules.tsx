import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'

interface GeneratorModulesProps {
  toggleModuleSelection: (path: string) => void
  handleAnalyze: () => void
}

export function GeneratorModules({
  toggleModuleSelection,
  handleAnalyze
}: GeneratorModulesProps) {
  const store = useStreamStore()

  if (!store.modules || store.modules.length === 0) return null

  return (
    <div className="mb-12">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100">请选择要深入解析的目录</h3>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
            已选择 {store.selectedModules.length} 个目录
          </p>
        </div>
        <Button
          onClick={handleAnalyze}
          disabled={store.selectedModules.length === 0 || store.isAnalyzing}
          className="min-w-[120px]"
        >
          {store.isAnalyzing ? <Loader2 className="w-4 h-4 animate-spin" /> : '深入解析并生成大纲'}
        </Button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {store.modules.map(mod => {
          const isSelected = store.selectedModules.includes(mod.path)
          return (
            <div
              key={mod.path}
              onClick={() => {
                if (!store.isAnalyzing && !store.isGenerating) {
                  toggleModuleSelection(mod.path)
                }
              }}
              className={`p-4 rounded-xl border transition-all cursor-pointer relative overflow-hidden group
                ${isSelected
                  ? 'bg-blue-50/50 border-blue-500 dark:bg-blue-900/10 dark:border-blue-400 shadow-sm'
                  : 'bg-white border-zinc-200 hover:border-zinc-300 dark:bg-zinc-900 dark:border-zinc-800 dark:hover:border-zinc-700'
                }
                ${(store.isAnalyzing || store.isGenerating) ? 'opacity-50 cursor-not-allowed' : ''}
              `}
            >
              {isSelected && (
                <div className="absolute top-0 right-0 w-8 h-8 bg-blue-500 dark:bg-blue-400 rounded-bl-xl flex items-center justify-center">
                  <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              )}
              <h4 className={`font-medium mb-2 ${isSelected ? 'text-blue-900 dark:text-blue-100 pr-6' : 'text-zinc-900 dark:text-zinc-100'}`}>
                {mod.name}
              </h4>
              <p className={`text-sm line-clamp-2 ${isSelected ? 'text-blue-700/80 dark:text-blue-200/80' : 'text-zinc-500 dark:text-zinc-400'}`}>
                {mod.description}
              </p>
            </div>
          )
        })}
      </div>
    </div>
  )
}
