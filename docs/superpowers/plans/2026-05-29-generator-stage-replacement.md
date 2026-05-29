# Generator Stage-Replacement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the generator into a source-first, full-stage workflow where each stage replaces the previous workspace and processing uses a dedicated progress page instead of a modal overlay.

**Architecture:** Keep `useBlogStream` and `useStreamStore` as the business orchestration layer, and move the UX change into a stricter page-level stage model. `Generator.tsx` becomes a stage shell that renders one stage component at a time, while `GeneratorStatus` is converted from a fixed overlay into reusable progress-page content owned by the new progress stage.

**Tech Stack:** React 18, TypeScript, Zustand, Vitest, Tailwind CSS, Shadcn UI

---

### Task 1: Tighten Generator Stage Derivation

**Files:**
- Modify: `frontend/src/pages/generatorViewState.ts`
- Modify: `frontend/src/pages/generatorViewState.test.ts`

- [ ] **Step 1: Expand the view-state test first**

Add/replace the unit cases in `frontend/src/pages/generatorViewState.test.ts` so the helper covers the stricter stage rules:

```ts
import { describe, expect, it } from 'vitest'
import { getGeneratorViewState } from './generatorViewState'

describe('getGeneratorViewState', () => {
  it('keeps the user on the source stage before a source is ready', () => {
    expect(
      getGeneratorViewState({
        sourceType: null,
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'source',
      currentStepIndex: 0,
      canGoBack: false,
      shouldRenderProgressPage: false,
      shouldShowScenarioSelector: false,
    })
  })

  it('moves git sources to configure after module scan completes', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'configure',
      currentStepIndex: 1,
      canGoBack: true,
      shouldShowScenarioSelector: true,
    })
  })

  it('moves file sources to configure once a file source exists', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'file',
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'configure',
      currentStepIndex: 1,
      canGoBack: true,
      shouldShowScenarioSelector: true,
    })
  })

  it('locks the scenario and enters outline once outline exists', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: [{ sort: 1, title: '章节一', summary: '摘要' }],
        scenarioMode: 'open_book_exam_review',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'outline',
      currentStepIndex: 2,
      canGoBack: true,
      lockedScenarioLabel: '开卷复习',
    })
  })

  it('forces the progress page during scan, analyze, and generate', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        modules: null,
        outline: null,
        scenarioMode: 'open_book_exam_review',
        isScanning: true,
        isAnalyzing: false,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'progress',
      currentStepIndex: 3,
      canGoBack: false,
      shouldRenderProgressPage: true,
    })
  })
})
```

- [ ] **Step 2: Run the focused test to confirm the new expectations fail**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts
```

Expected:

```text
FAIL  src/pages/generatorViewState.test.ts
```

- [ ] **Step 3: Update the helper to expose the stricter stage contract**

Refactor `frontend/src/pages/generatorViewState.ts` to return a stage-first contract like this:

```ts
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

type GeneratorStage = 'source' | 'configure' | 'outline' | 'progress'

export function getGeneratorViewState({
  sourceType,
  modules,
  outline,
  scenarioMode,
  isScanning,
  isAnalyzing,
  isGenerating,
}: GeneratorViewStateInput) {
  const hasOutline = Boolean(outline?.length)
  const hasGitModules = sourceType === 'git' && Boolean(modules?.length)
  const hasChosenSource = Boolean(sourceType)
  const isProcessing = isScanning || isAnalyzing || isGenerating

  let currentStage: GeneratorStage = 'source'

  if (isProcessing) {
    currentStage = 'progress'
  } else if (hasOutline) {
    currentStage = 'outline'
  } else if ((sourceType === 'git' && hasGitModules) || sourceType === 'file') {
    currentStage = 'configure'
  } else if (hasChosenSource) {
    currentStage = 'source'
  }

  return {
    currentStage,
    currentStepIndex:
      currentStage === 'source' ? 0 :
      currentStage === 'configure' ? 1 :
      currentStage === 'outline' ? 2 : 3,
    isProcessing,
    canGoBack: currentStage === 'configure' || currentStage === 'outline',
    shouldRenderProgressPage: currentStage === 'progress',
    shouldShowScenarioSelector: currentStage === 'configure',
    lockedScenarioLabel: hasOutline ? scenarioModeLabelMap[scenarioMode] : null,
  }
}
```

- [ ] **Step 4: Run the focused test again**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts
```

Expected:

```text
PASS  src/pages/generatorViewState.test.ts
```

- [ ] **Step 5: Commit the stage-helper slice**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/generatorViewState.ts frontend/src/pages/generatorViewState.test.ts
git commit -m "test(generator): tighten stage derivation rules"
```

### Task 2: Extract Stage-Level Generator Views

**Files:**
- Create: `frontend/src/components/generator/GeneratorSourceStage.tsx`
- Create: `frontend/src/components/generator/GeneratorConfigureStage.tsx`
- Create: `frontend/src/components/generator/GeneratorOutlineStage.tsx`
- Modify: `frontend/src/components/generator/GeneratorInput.tsx`
- Modify: `frontend/src/components/generator/GeneratorModules.tsx`
- Modify: `frontend/src/components/generator/GeneratorOutline.tsx`

- [ ] **Step 1: Create the source stage wrapper**

Add `frontend/src/components/generator/GeneratorSourceStage.tsx` as the full source-selection page block:

```tsx
import type { ChangeEvent, DragEvent, RefObject } from 'react'
import { GeneratorInput } from '@/components/generator/GeneratorInput'

interface GeneratorSourceStageProps {
  gitUrl: string
  setGitUrl: (value: string) => void
  isDragging: boolean
  handleScan: () => void
  handleDragOver: (event: DragEvent<HTMLDivElement>) => void
  handleDragLeave: (event: DragEvent<HTMLDivElement>) => void
  handleDrop: (event: DragEvent<HTMLDivElement>) => void
  handleFileChange: (event: ChangeEvent<HTMLInputElement>) => void
  fileInputRef: RefObject<HTMLInputElement | null>
  stopAnalyzing: () => void
}

/**
 * Why: 用户的第一个问题只应该是“资料从哪里来”，所以来源阶段独占主工作区。
 */
export function GeneratorSourceStage(props: GeneratorSourceStageProps) {
  return (
    <section className="rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-8 space-y-2">
        <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">先选择资料来源</h2>
        <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
          先确认你要解析的是 GitHub 仓库还是本地文档，完成这一步后再进入下一页配置解析策略。
        </p>
      </div>
      <GeneratorInput {...props} />
    </section>
  )
}
```

- [ ] **Step 2: Create the configure stage wrapper**

Add `frontend/src/components/generator/GeneratorConfigureStage.tsx` to isolate scenario selection plus git/file-specific setup:

```tsx
import type { ReactNode } from 'react'

interface GeneratorConfigureStageProps {
  sourceLabel: string
  scenarioSelector: ReactNode
  modulePicker?: ReactNode
  fileSummary?: ReactNode
  onBack: () => void
}

/**
 * Why: 选择来源之后，用户只需要处理“如何解析”这一件事，避免继续看到来源入口和大纲编辑。
 */
export function GeneratorConfigureStage({
  sourceLabel,
  scenarioSelector,
  modulePicker,
  fileSummary,
  onBack,
}: GeneratorConfigureStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="text-sm text-zinc-500 dark:text-zinc-400">当前来源：{sourceLabel}</div>
          <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">配置解析方式</h2>
          <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
            这一步只保留当前来源所需的配置项，确认后系统会进入处理页并生成可编辑大纲。
          </p>
        </div>
        <button
          type="button"
          onClick={onBack}
          className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
        >
          返回上一步
        </button>
      </div>

      {scenarioSelector}
      {modulePicker}
      {fileSummary}
    </section>
  )
}
```

- [ ] **Step 3: Create the outline stage wrapper**

Add `frontend/src/components/generator/GeneratorOutlineStage.tsx`:

```tsx
import type { ReactNode } from 'react'

interface GeneratorOutlineStageProps {
  lockedScenarioLabel?: string | null
  outlineEditor: ReactNode
  onBack: () => void
}

/**
 * Why: 大纲出现后，用户应只专注于修改结构和开始生成，而不是回看前序配置面板。
 */
export function GeneratorOutlineStage({
  lockedScenarioLabel,
  outlineEditor,
  onBack,
}: GeneratorOutlineStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">确认并调整大纲</h2>
          <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
            当前页面只保留大纲相关操作，确认无误后直接开始生成博客。
          </p>
          {lockedScenarioLabel ? (
            <div className="inline-flex items-center rounded-full border border-zinc-200 bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
              当前创作场景：{lockedScenarioLabel}
            </div>
          ) : null}
        </div>
        <button
          type="button"
          onClick={onBack}
          className="rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-600 transition hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:text-zinc-100"
        >
          返回上一步
        </button>
      </div>

      {outlineEditor}
    </section>
  )
}
```

- [ ] **Step 4: Slim the child components so the wrappers own the page framing**

Make these minimal component adjustments:

```tsx
// frontend/src/components/generator/GeneratorInput.tsx
// Change the outer spacing from page-level margin to local grid-only layout.
return (
  <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
    ...
  </div>
)
```

```tsx
// frontend/src/components/generator/GeneratorModules.tsx
// Remove outer page margin so it can sit cleanly inside GeneratorConfigureStage.
return (
  <div>
    ...
  </div>
)
```

```tsx
// frontend/src/components/generator/GeneratorOutline.tsx
// Keep the editor behavior, but remove the stage-level scenario chip because the wrapper now owns it.
interface GeneratorOutlineProps {
  ...
  lockedScenarioLabel?: string | null
}

...
{lockedScenarioLabel ? null : null}
```

- [ ] **Step 5: Run a fast type/build check on the changed component surface**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run build
```

Expected:

```text
vite v...
✓ built in ...
```

- [ ] **Step 6: Commit the stage component extraction**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/generator/GeneratorSourceStage.tsx frontend/src/components/generator/GeneratorConfigureStage.tsx frontend/src/components/generator/GeneratorOutlineStage.tsx frontend/src/components/generator/GeneratorInput.tsx frontend/src/components/generator/GeneratorModules.tsx frontend/src/components/generator/GeneratorOutline.tsx
git commit -m "feat(generator): extract stage-level generator views"
```

### Task 3: Convert Progress Overlay Into Dedicated Progress Content

**Files:**
- Create: `frontend/src/components/generator/GeneratorProgressStage.tsx`
- Modify: `frontend/src/components/generator/GeneratorStatus.tsx`

- [ ] **Step 1: Change `GeneratorStatus` from modal overlay to embeddable content**

Refactor `frontend/src/components/generator/GeneratorStatus.tsx` so it returns only the inner progress content and no longer uses fixed overlay classes:

```tsx
export function GeneratorStatus() {
  const store = useStreamStore()
  ...

  if (!store.isScanning && !store.isAnalyzing && !store.isGenerating && !store.progress && !store.currentChapterTitle && !store.analysisMessage) {
    return null
  }

  return (
    <div className="overflow-hidden rounded-3xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="border-b border-zinc-200 bg-zinc-50 px-6 py-4 dark:border-zinc-800 dark:bg-zinc-800/50">
        ...
      </div>
      <div className="max-h-[70vh] overflow-y-auto p-6 custom-scrollbar">
        ...
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Create the progress page wrapper**

Add `frontend/src/components/generator/GeneratorProgressStage.tsx`:

```tsx
import { GeneratorStatus } from '@/components/generator/GeneratorStatus'

interface GeneratorProgressStageProps {
  title: string
  description: string
}

/**
 * Why: 处理中禁止用户在多个操作面板之间来回切换，所以独立进度页接管主工作区。
 */
export function GeneratorProgressStage({
  title,
  description,
}: GeneratorProgressStageProps) {
  return (
    <section className="space-y-6">
      <div className="rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">{title}</h2>
        <p className="mt-2 text-sm leading-6 text-zinc-500 dark:text-zinc-400">{description}</p>
      </div>
      <GeneratorStatus />
    </section>
  )
}
```

- [ ] **Step 3: Add a focused smoke test for progress-only rendering rules through the page shell later**

Document the intended behavior in the plan by preserving these conditions for later verification:

```ts
expect(viewState.shouldRenderProgressPage).toBe(true)
expect(viewState.canGoBack).toBe(false)
```

- [ ] **Step 4: Run the helper test again because progress semantics changed**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts
```

Expected:

```text
PASS  src/pages/generatorViewState.test.ts
```

- [ ] **Step 5: Commit the progress conversion**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/generator/GeneratorStatus.tsx frontend/src/components/generator/GeneratorProgressStage.tsx
git commit -m "feat(generator): move progress into dedicated stage"
```

### Task 4: Rebuild `Generator.tsx` As A Single-Stage Shell

**Files:**
- Modify: `frontend/src/pages/Generator.tsx`
- Reference: `frontend/src/components/shared/StepStrip.tsx`
- Reference: `frontend/src/store/streamStore.ts`

- [ ] **Step 1: Replace the mixed layout with stage-only rendering**

Refactor `frontend/src/pages/Generator.tsx` so it renders exactly one of the stage wrappers:

```tsx
import { GeneratorSourceStage } from '@/components/generator/GeneratorSourceStage'
import { GeneratorConfigureStage } from '@/components/generator/GeneratorConfigureStage'
import { GeneratorOutlineStage } from '@/components/generator/GeneratorOutlineStage'
import { GeneratorProgressStage } from '@/components/generator/GeneratorProgressStage'

...

const generatorSteps: StepStripItem[] = [
  { key: 'source', title: '选择来源', description: '先决定资料入口。' },
  { key: 'configure', title: '配置解析', description: '只处理当前来源的解析配置。' },
  { key: 'outline', title: '确认大纲', description: '只保留大纲调整和开始生成。' },
  { key: 'progress', title: '处理进度', description: '处理期间进入专注的进度页面。' },
]

...

{viewState.currentStage === 'source' ? (
  <GeneratorSourceStage ... />
) : null}

{viewState.currentStage === 'configure' ? (
  <GeneratorConfigureStage
    sourceLabel={store.sourceType === 'git' ? 'GitHub 仓库' : '本地文档'}
    scenarioSelector={renderScenarioSelector()}
    modulePicker={
      store.sourceType === 'git'
        ? <GeneratorModules toggleModuleSelection={toggleModuleSelection} handleAnalyze={handleAnalyze} />
        : undefined
    }
    fileSummary={
      store.sourceType === 'file'
        ? (
          <section className="rounded-2xl border border-zinc-200 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-950/40">
            <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">已选择本地文档</h3>
            <p className="mt-2 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
              当前文档已进入解析准备阶段，确认创作场景后系统会继续生成大纲。
            </p>
          </section>
        )
        : undefined
    }
    onBack={backFromConfigure}
  />
) : null}

{viewState.currentStage === 'outline' ? (
  <GeneratorOutlineStage
    lockedScenarioLabel={viewState.lockedScenarioLabel}
    onBack={backFromOutline}
    outlineEditor={
      <GeneratorOutline
        isOutlineExpanded={isOutlineExpanded}
        setIsOutlineExpanded={setIsOutlineExpanded}
        setShowChapterDeleteConfirm={setShowChapterDeleteConfirm}
        handleGenerate={handleGenerate}
        stopGenerating={stopGenerating}
        lockedScenarioLabel={null}
      />
    }
  />
) : null}

{viewState.currentStage === 'progress' ? (
  <GeneratorProgressStage
    title="正在处理当前任务"
    description="当前页面只保留进度反馈，处理完成后会自动回到下一阶段。"
  />
) : null}
```

- [ ] **Step 2: Remove the old right summary sidebar from the progress-stage layout**

Keep the page header and step strip, but delete the old `aside` summary layout from `Generator.tsx` so the dedicated progress stage uses the full content width:

```tsx
// remove:
<div className="grid gap-8 lg:grid-cols-[minmax(0,1.4fr)_340px]">
  <div>...</div>
  <aside>...</aside>
</div>

// replace with:
<div className="space-y-8">
  ...
</div>
```

- [ ] **Step 3: Keep back behavior strict**

Preserve and simplify the existing back handlers:

```tsx
const backFromConfigure = () => {
  store.setModules(null)
  store.setSelectedModules([])
  store.setOutline(null)
  store.setParentBlogId(null)
  if (store.sourceType === 'file') {
    store.setSourceContent('')
    store.setSource('file', '')
  }
}

const backFromOutline = () => {
  store.setOutline(null)
  store.setParentBlogId(null)
  if (store.sourceType !== 'git') {
    store.setSourceContent('')
  }
}
```

Then ensure the progress stage gets no back button at all.

- [ ] **Step 4: Run the build**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run build
```

Expected:

```text
vite v...
✓ built in ...
```

- [ ] **Step 5: Commit the page-shell refactor**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/Generator.tsx
git commit -m "feat(generator): render one full stage at a time"
```

### Task 5: Verify The Full Frontend Flow

**Files:**
- Test: `frontend/src/pages/generatorViewState.test.ts`
- Modify if needed: `frontend/src/pages/Generator.tsx`
- Modify if needed: `frontend/src/components/generator/GeneratorStatus.tsx`

- [ ] **Step 1: Run the focused frontend test suite**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts src/components/shared/StepStrip.test.tsx
```

Expected:

```text
PASS  src/pages/generatorViewState.test.ts
PASS  src/components/shared/StepStrip.test.tsx
```

- [ ] **Step 2: Run the frontend build one more time**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run build
```

Expected:

```text
vite v...
✓ built in ...
```

- [ ] **Step 3: Manually verify the four stage transitions**

Use the running app and confirm this checklist:

```text
1. 初始打开生成器时，只看到来源选择页。
2. 选择 Git 来源并完成扫描后，页面切换到配置解析页，来源入口消失。
3. 大纲生成后，页面切换到大纲确认页，场景只读显示。
4. 开始生成后，页面切换到独立进度页，且没有返回操作。
5. 文件来源路径也遵循“来源 -> 配置 -> 进度/大纲”的单阶段显示。
```

- [ ] **Step 4: Commit the verification pass**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/generatorViewState.test.ts frontend/src/pages/Generator.tsx frontend/src/components/generator/GeneratorStatus.tsx frontend/src/components/generator/GeneratorSourceStage.tsx frontend/src/components/generator/GeneratorConfigureStage.tsx frontend/src/components/generator/GeneratorOutlineStage.tsx frontend/src/components/generator/GeneratorProgressStage.tsx frontend/src/components/generator/GeneratorInput.tsx frontend/src/components/generator/GeneratorModules.tsx frontend/src/components/generator/GeneratorOutline.tsx
git commit -m "fix(generator): verify staged workspace flow"
```

## Self-Review
- **Spec coverage:** The plan covers source-first flow, configure-stage scenario selection, dedicated outline stage, dedicated progress page, disabled back during processing, and frontend verification for both `git` and `file` paths.
- **Placeholder scan:** No `TODO`, `TBD`, or “handle later” language remains in the task steps.
- **Type consistency:** The plan consistently uses `currentStage`, `currentStepIndex`, `canGoBack`, `shouldRenderProgressPage`, and `shouldShowScenarioSelector` across tests and implementation tasks.

