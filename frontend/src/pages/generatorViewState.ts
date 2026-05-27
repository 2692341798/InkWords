import type { Chapter, ModuleCard } from '@/store/streamStore'
import { scenarioModeLabelMap, type ScenarioMode } from '@/lib/scenarioMode'

interface GeneratorViewStateInput {
  sourceType: 'git' | 'file' | null
  modules: ModuleCard[] | null
  outline: Chapter[] | null
  scenarioMode: ScenarioMode
  isScanning: boolean
  isAnalyzing: boolean
  isGenerating: boolean
}

/**
 * Why: 页面需要一个明确的“当前步骤”，避免输入、模块选择和大纲编辑同时出现，
 * 把原本平铺的生成流程收敛成单一主任务。
 */
export function getGeneratorViewState({
  sourceType,
  modules,
  outline,
  scenarioMode,
  isScanning,
  isAnalyzing,
  isGenerating,
}: GeneratorViewStateInput) {
  const hasOutline = Boolean(outline && outline.length > 0)
  const hasConfigurableGitModules = sourceType === 'git' && Boolean(modules && modules.length > 0)
  const isProcessing = isScanning || isAnalyzing || isGenerating

  let currentStep: 'input' | 'configure' | 'outline' | 'processing' = 'input'

  if (isProcessing) {
    currentStep = 'processing'
  } else if (hasOutline) {
    currentStep = 'outline'
  } else if (hasConfigurableGitModules) {
    currentStep = 'configure'
  }

  return {
    currentStep,
    currentStepIndex:
      currentStep === 'input'
        ? 0
        : currentStep === 'configure'
          ? 1
          : currentStep === 'outline'
            ? 2
            : 3,
    shouldShowInputStep: currentStep === 'input',
    shouldShowConfigureStep: currentStep === 'configure',
    shouldShowOutlineStep: currentStep === 'outline',
    shouldShowScenarioSelector: currentStep === 'input' || currentStep === 'configure',
    lockedScenarioLabel: hasOutline ? scenarioModeLabelMap[scenarioMode] : null,
    isProcessing,
  }
}
