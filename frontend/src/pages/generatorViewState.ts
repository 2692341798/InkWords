import type { Chapter, ModuleCard } from '@/store/streamStore'
import { scenarioModeLabelMap, type ScenarioMode } from '@/lib/scenarioMode'

interface GeneratorViewStateInput {
  sourceType: 'git' | 'file' | null
  sourceContent: string
  modules: ModuleCard[] | null
  outline: Chapter[] | null
  scenarioMode: ScenarioMode
  isScanning: boolean
  isAnalyzing: boolean
  isGenerating: boolean
}

type GeneratorStage = 'source' | 'configure' | 'outline'

/**
 * Why: 页面需要一个明确的“当前步骤”，避免输入、模块选择和大纲编辑同时出现，
 * 把原本平铺的生成流程收敛成单一主任务。
 */
export function getGeneratorViewState({
  sourceType,
  sourceContent,
  modules,
  outline,
  scenarioMode,
  isScanning,
  isAnalyzing,
  isGenerating,
}: GeneratorViewStateInput) {
  const hasOutline = Boolean(outline?.length)
  const hasConfigurableGitModules = sourceType === 'git' && Boolean(modules?.length)
  const hasParsedFileContent = sourceType === 'file' && sourceContent.trim().length > 0
  const isWorkingInConfigure = isScanning || isAnalyzing
  const isWorkingInOutline = isGenerating
  const isProcessing = isWorkingInConfigure || isWorkingInOutline

  let currentStage: GeneratorStage = 'source'

  if (hasOutline) {
    currentStage = 'outline'
  } else if ((sourceType === 'git' && hasConfigurableGitModules) || hasParsedFileContent) {
    currentStage = 'configure'
  }

  const currentStep =
    currentStage === 'source'
      ? 'input'
      : currentStage === 'configure'
        ? 'configure'
        : 'outline'

  return {
    currentStage,
    currentStep,
    currentStepIndex:
      currentStage === 'source'
        ? 0
        : currentStage === 'configure'
          ? 1
          : 2,
    canGoBack: currentStage === 'configure' || currentStage === 'outline',
    shouldRenderProgressPage: false,
    shouldShowInputStep: currentStep === 'input',
    shouldShowConfigureStep: currentStep === 'configure',
    shouldShowOutlineStep: currentStep === 'outline',
    shouldShowScenarioSelector: currentStage === 'configure',
    shouldShowInlineProgress: isWorkingInConfigure || isWorkingInOutline,
    progressHostStage: isWorkingInConfigure ? 'configure' : isWorkingInOutline ? 'outline' : null,
    lockedScenarioLabel: hasOutline ? scenarioModeLabelMap[scenarioMode] : null,
    isProcessing,
  }
}
