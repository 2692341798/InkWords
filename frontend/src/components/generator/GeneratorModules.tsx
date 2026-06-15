import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'
import { useShallow } from 'zustand/react/shallow'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'

interface GeneratorModulesProps {
  toggleModuleSelection: (path: string) => void
  handleAnalyze: () => void
}

export function GeneratorModules({
  toggleModuleSelection,
  handleAnalyze
}: GeneratorModulesProps) {
  const store = useStreamStore(
    useShallow((state) => ({
      modules: state.modules,
      selectedModules: state.selectedModules,
      isAnalyzing: state.isAnalyzing,
      isGenerating: state.isGenerating,
    })),
  )

  if (!store.modules || store.modules.length === 0) return null

  return (
    <Panel className="p-5">
      <SectionHeader
        title="选择深入解析目录"
        description="只选择和本次写作目标相关的目录，减少后续大纲噪声。"
        action={
          <div className="flex flex-wrap items-center gap-2">
            <StatusPill>已选择 {store.selectedModules.length} 个</StatusPill>
        <Button
          onClick={handleAnalyze}
          disabled={store.selectedModules.length === 0 || store.isAnalyzing}
          className="min-w-[120px]"
        >
          {store.isAnalyzing ? <Loader2 className="w-4 h-4 animate-spin" /> : '深入解析并生成大纲'}
        </Button>
      </div>
        }
      />
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
              className={`choice-tile relative cursor-pointer overflow-hidden
                ${isSelected
                  ? 'choice-tile-active'
                  : 'choice-tile-muted'
                }
                ${(store.isAnalyzing || store.isGenerating) ? 'opacity-50 cursor-not-allowed' : ''}
              `}
            >
              {isSelected && (
                <div className="absolute right-0 top-0 flex h-8 w-8 items-center justify-center rounded-bl-xl bg-[var(--brand)]">
                  <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              )}
              <h4 className={`mb-2 font-medium ${isSelected ? 'pr-6 text-foreground' : 'text-foreground'}`}>
                {mod.name}
              </h4>
              <p className="line-clamp-2 text-sm text-muted-foreground">
                {mod.description}
              </p>
            </div>
          )
        })}
      </div>
    </Panel>
  )
}
