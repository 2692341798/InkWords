import { GeneratorStatus } from '@/components/generator/GeneratorStatus'

interface GeneratorProgressStageProps {
  title?: string
  description?: string
}

/**
 * Why: Task 2 retires the dedicated progress page shell, but this compatibility
 * shim keeps existing imports working until the page-level flow removes it.
 */
export function GeneratorProgressStage(_props: GeneratorProgressStageProps) {
  return <GeneratorStatus />
}
