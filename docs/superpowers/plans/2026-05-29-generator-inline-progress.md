# Generator Inline-Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the generator from a 4-step flow with a standalone progress step into a 3-step flow where progress is embedded inside `配置解析` and `确认大纲`.

**Architecture:** Keep the existing source/configure/outline wrappers and shared store/hook orchestration, but collapse the top-level stage model from `source -> configure -> outline -> progress` to `source -> configure -> outline`. Reuse `GeneratorStatus` as an inline progress panel rendered conditionally inside configure and outline, and update the file flow so upload/parse stops at step 2 until the user explicitly starts outline generation.

**Tech Stack:** React 19, TypeScript, Zustand, Vitest, Tailwind CSS, Vite, Docker Compose

---

### Task 1: Collapse The Stage Model To Three Steps

**Files:**
- Modify: `frontend/src/pages/generatorViewState.ts`
- Modify: `frontend/src/pages/generatorViewState.test.ts`

- [ ] **Step 1: Write the failing view-state tests for the 3-step model**

Replace the top-level expectations in `frontend/src/pages/generatorViewState.test.ts` with these cases:

```ts
import { describe, expect, it } from 'vitest'
import { getGeneratorViewState } from './generatorViewState'

describe('getGeneratorViewState', () => {
  it('stays on source before any source is ready', () => {
    expect(
      getGeneratorViewState({
        sourceType: null,
        sourceContent: '',
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
      shouldShowInlineProgress: false,
      progressHostStage: null,
    })
  })

  it('keeps parsing and analysis inside configure instead of switching to a progress stage', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'file',
        sourceContent: 'parsed zip content',
        modules: null,
        outline: null,
        scenarioMode: 'ebook_interpretation',
        isScanning: false,
        isAnalyzing: true,
        isGenerating: false,
      }),
    ).toMatchObject({
      currentStage: 'configure',
      currentStepIndex: 1,
      shouldShowInlineProgress: true,
      progressHostStage: 'configure',
    })
  })

  it('keeps generation inside outline instead of switching to a progress stage', () => {
    expect(
      getGeneratorViewState({
        sourceType: 'git',
        sourceContent: 'repo summary',
        modules: [{ path: 'cmd', name: 'cmd', description: '入口目录' }],
        outline: [{ sort: 1, title: '第一篇', summary: '摘要' }],
        scenarioMode: 'ebook_interpretation',
        isScanning: false,
        isAnalyzing: false,
        isGenerating: true,
      }),
    ).toMatchObject({
      currentStage: 'outline',
      currentStepIndex: 2,
      shouldShowInlineProgress: true,
      progressHostStage: 'outline',
    })
  })
})
```

- [ ] **Step 2: Run the test to confirm the current 4-step logic fails**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts
```

Expected:

```text
FAIL  src/pages/generatorViewState.test.ts
```

- [ ] **Step 3: Implement the minimal 3-stage helper**

Update `frontend/src/pages/generatorViewState.ts` so it no longer derives `progress` as a top-level stage:

```ts
type GeneratorStage = 'source' | 'configure' | 'outline'

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

  let currentStage: GeneratorStage = 'source'

  if (hasOutline) {
    currentStage = 'outline'
  } else if ((sourceType === 'git' && hasConfigurableGitModules) || hasParsedFileContent) {
    currentStage = 'configure'
  }

  return {
    currentStage,
    currentStepIndex: currentStage === 'source' ? 0 : currentStage === 'configure' ? 1 : 2,
    canGoBack: currentStage === 'configure' || currentStage === 'outline',
    shouldShowScenarioSelector: currentStage === 'configure',
    shouldShowInlineProgress: isWorkingInConfigure || isWorkingInOutline,
    progressHostStage: isWorkingInConfigure ? 'configure' : isWorkingInOutline ? 'outline' : null,
    lockedScenarioLabel: hasOutline ? scenarioModeLabelMap[scenarioMode] : null,
  }
}
```

- [ ] **Step 4: Re-run the focused test to confirm green**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts
```

Expected:

```text
PASS  src/pages/generatorViewState.test.ts
```

- [ ] **Step 5: Commit the stage-model slice**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/generatorViewState.ts frontend/src/pages/generatorViewState.test.ts
git commit -m "test(generator): collapse stage model to three steps"
```

### Task 2: Make `GeneratorStatus` A Pure Inline Progress Panel

**Files:**
- Modify: `frontend/src/components/generator/GeneratorStatus.tsx`
- Modify: `frontend/src/components/generator/GeneratorStageViews.test.tsx`
- Optionally delete later: `frontend/src/components/generator/GeneratorProgressStage.tsx`

- [ ] **Step 1: Write a failing rendering test that progress belongs inside configure/outline, not a standalone step**

Add a focused assertion in `frontend/src/components/generator/GeneratorStageViews.test.tsx`:

```tsx
it('renders GeneratorStatus as embeddable inline content for active steps', () => {
  mockStreamState.isAnalyzing = true
  mockStreamState.analysisStep = 1
  mockStreamState.analysisMessage = '正在生成大纲...'
  mockStreamState.analysisHistory = [{ id: 1, message: '正在生成大纲...', status: 'outline' }]
  mockStreamState.sourceType = 'file'

  const html = renderToStaticMarkup(<GeneratorStatus />)

  expect(html).toContain('解析进度')
  expect(html).not.toContain('fixed inset-0')
  expect(html).toContain('overflow-hidden rounded-3xl')
})
```

- [ ] **Step 2: Run the focused test to ensure it fails only if the contract is wrong**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
PASS or FAIL only if the current progress component breaks the inline contract
```

If it already passes, keep the test and continue without changing behavior in this file.

- [ ] **Step 3: Simplify the progress component contract if needed**

Keep `frontend/src/components/generator/GeneratorStatus.tsx` inline-only and remove any remaining standalone-step wording:

```tsx
// keep the component as a reusable panel
// do not let it own page-level titles like "当前正在处理当前任务"
return (
  <div className="overflow-hidden rounded-3xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
    ...
  </div>
)
```

- [ ] **Step 4: Remove or retire the standalone progress wrapper**

If `frontend/src/components/generator/GeneratorProgressStage.tsx` is no longer needed, delete it:

```bash
rm /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend/src/components/generator/GeneratorProgressStage.tsx
```

If a temporary compatibility stub is safer, replace it with:

```tsx
export { GeneratorStatus as GeneratorProgressStage } from '@/components/generator/GeneratorStatus'
```

- [ ] **Step 5: Re-run the focused test**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
PASS  src/components/generator/GeneratorStageViews.test.tsx
```

- [ ] **Step 6: Commit the progress-panel cleanup**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/generator/GeneratorStatus.tsx frontend/src/components/generator/GeneratorStageViews.test.tsx frontend/src/components/generator/GeneratorProgressStage.tsx
git commit -m "refactor(generator): keep progress inline within active steps"
```

### Task 3: Embed Analysis Progress Inside Configure

**Files:**
- Modify: `frontend/src/components/generator/GeneratorConfigureStage.tsx`
- Modify: `frontend/src/pages/Generator.tsx`
- Reference: `frontend/src/components/generator/GeneratorModules.tsx`
- Reference: `frontend/src/hooks/generator/useFileParser.ts`

- [ ] **Step 1: Write a failing stage-wrapper test for inline configure progress**

Add a test to `frontend/src/components/generator/GeneratorStageViews.test.tsx`:

```tsx
it('shows configure content and inline progress together while analyzing', () => {
  const html = renderToStaticMarkup(
    <GeneratorConfigureStage
      sourceLabel="本地文档"
      scenarioSelector={<div>场景选择器</div>}
      fileSummary={<div>文件摘要</div>}
      progressPanel={<div>解析进度面板</div>}
      onBack={() => {}}
    />,
  )

  expect(html).toContain('配置解析方式')
  expect(html).toContain('场景选择器')
  expect(html).toContain('文件摘要')
  expect(html).toContain('解析进度面板')
})
```

- [ ] **Step 2: Run the wrapper test and confirm red**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
FAIL with "progressPanel" prop missing
```

- [ ] **Step 3: Extend the configure wrapper with an optional inline progress slot**

Update `frontend/src/components/generator/GeneratorConfigureStage.tsx`:

```tsx
interface GeneratorConfigureStageProps {
  sourceLabel: string
  scenarioSelector: ReactNode
  modulePicker?: ReactNode
  fileSummary?: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

export function GeneratorConfigureStage({
  sourceLabel,
  scenarioSelector,
  modulePicker,
  fileSummary,
  progressPanel,
  onBack,
}: GeneratorConfigureStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      ...
      {scenarioSelector}
      {modulePicker}
      {fileSummary}
      {progressPanel}
    </section>
  )
}
```

- [ ] **Step 4: Render `GeneratorStatus` inline in configure while scanning/analyzing**

Update the configure branch in `frontend/src/pages/Generator.tsx`:

```tsx
{viewState.currentStage === 'configure' ? (
  <GeneratorConfigureStage
    sourceLabel={store.sourceType === 'git' ? 'GitHub 仓库' : '本地文档'}
    scenarioSelector={renderScenarioSelector()}
    modulePicker={store.sourceType === 'git' ? <GeneratorModules ... /> : undefined}
    fileSummary={store.sourceType === 'file' ? <section>...</section> : undefined}
    progressPanel={viewState.progressHostStage === 'configure' ? <GeneratorStatus /> : undefined}
    onBack={backFromConfigure}
  />
) : null}
```

- [ ] **Step 5: Re-run the wrapper test**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
PASS  src/components/generator/GeneratorStageViews.test.tsx
```

- [ ] **Step 6: Commit the configure-inline-progress slice**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/generator/GeneratorConfigureStage.tsx frontend/src/pages/Generator.tsx frontend/src/components/generator/GeneratorStageViews.test.tsx
git commit -m "feat(generator): show analysis progress inside configure"
```

### Task 4: Embed Writing Progress Inside Outline

**Files:**
- Modify: `frontend/src/components/generator/GeneratorOutlineStage.tsx`
- Modify: `frontend/src/pages/Generator.tsx`
- Reference: `frontend/src/components/generator/GeneratorOutline.tsx`

- [ ] **Step 1: Write a failing wrapper test for inline outline progress**

Add to `frontend/src/components/generator/GeneratorStageViews.test.tsx`:

```tsx
it('shows outline editor and generation progress in the same stage', () => {
  const html = renderToStaticMarkup(
    <GeneratorOutlineStage
      lockedScenarioLabel="电子书解读"
      outlineEditor={<div>大纲编辑器</div>}
      progressPanel={<div>生成进度面板</div>}
      onBack={() => {}}
    />,
  )

  expect(html).toContain('确认并调整大纲')
  expect(html).toContain('大纲编辑器')
  expect(html).toContain('生成进度面板')
})
```

- [ ] **Step 2: Run the test and confirm red**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
FAIL with "progressPanel" prop missing
```

- [ ] **Step 3: Extend the outline wrapper with an optional inline progress slot**

Update `frontend/src/components/generator/GeneratorOutlineStage.tsx`:

```tsx
interface GeneratorOutlineStageProps {
  lockedScenarioLabel?: string | null
  outlineEditor: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

export function GeneratorOutlineStage({
  lockedScenarioLabel,
  outlineEditor,
  progressPanel,
  onBack,
}: GeneratorOutlineStageProps) {
  return (
    <section className="space-y-6 rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      ...
      {outlineEditor}
      {progressPanel}
    </section>
  )
}
```

- [ ] **Step 4: Render `GeneratorStatus` inline in outline while generating**

Update `frontend/src/pages/Generator.tsx`:

```tsx
{viewState.currentStage === 'outline' ? (
  <GeneratorOutlineStage
    lockedScenarioLabel={viewState.lockedScenarioLabel}
    onBack={backFromOutline}
    outlineEditor={<GeneratorOutline ... lockedScenarioLabel={null} />}
    progressPanel={viewState.progressHostStage === 'outline' ? <GeneratorStatus /> : undefined}
  />
) : null}
```

- [ ] **Step 5: Remove any remaining standalone-progress branch from `Generator.tsx`**

Delete the old branch entirely:

```tsx
{viewState.currentStage === 'progress' ? (
  <GeneratorProgressStage ... />
) : null}
```

- [ ] **Step 6: Re-run the wrapper test**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/components/generator/GeneratorStageViews.test.tsx
```

Expected:

```text
PASS  src/components/generator/GeneratorStageViews.test.tsx
```

- [ ] **Step 7: Commit the outline-inline-progress slice**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/components/generator/GeneratorOutlineStage.tsx frontend/src/pages/Generator.tsx frontend/src/components/generator/GeneratorStageViews.test.tsx
git commit -m "feat(generator): show writing progress inside outline"
```

### Task 5: Update The 3-Step Strip And Verify Real Flow

**Files:**
- Modify: `frontend/src/pages/Generator.tsx`
- Modify if needed: `frontend/src/components/shared/StepStrip.tsx`
- Test: `frontend/src/pages/generatorViewState.test.ts`
- Test: `frontend/src/components/generator/GeneratorStageViews.test.tsx`

- [ ] **Step 1: Update the generator step strip to 3 items**

In `frontend/src/pages/Generator.tsx`, change:

```tsx
const generatorSteps: StepStripItem[] = [
  { key: 'source', title: '选择来源', description: '先决定资料入口。' },
  { key: 'configure', title: '配置解析', description: '选择场景并完成解析。' },
  { key: 'outline', title: '确认大纲', description: '调整大纲并查看写作进度。' },
]
```

- [ ] **Step 2: Run the focused test suite**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- src/pages/generatorViewState.test.ts src/components/generator/GeneratorStageViews.test.tsx src/components/shared/StepStrip.test.tsx
```

Expected:

```text
PASS  all selected test files
```

- [ ] **Step 3: Run the frontend build**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run build
```

Expected:

```text
✓ built in ...
```

- [ ] **Step 4: Rebuild Docker for the `http://localhost` workflow**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
OBSIDIAN_VAULT_PATH='/Users/huangqijun/Documents/obsidian_knowledge/knowledge' docker compose up -d --build
```

Expected:

```text
Container inkwords-frontend Started
Container inkwords-backend Started
```

- [ ] **Step 5: Manually verify the 3-step UX**

Use the running app and confirm this checklist:

```text
1. 顶部流程条只有 3 步，不再显示“处理进度”。
2. 上传 ZIP 后先进入“配置解析”，而不是跳到独立进度页。
3. “配置解析”内可以看到解析进度面板。
4. “确认大纲”内开始写作后，生成进度仍留在同一步里。
5. 不会再出现 step 4 在 step 3 前面的错觉。
```

- [ ] **Step 6: Commit the verification slice**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/pages/Generator.tsx frontend/src/pages/generatorViewState.ts frontend/src/pages/generatorViewState.test.ts frontend/src/components/generator/GeneratorStatus.tsx frontend/src/components/generator/GeneratorConfigureStage.tsx frontend/src/components/generator/GeneratorOutlineStage.tsx frontend/src/components/generator/GeneratorStageViews.test.tsx frontend/src/components/generator/GeneratorProgressStage.tsx
git commit -m "fix(generator): embed progress within configure and outline"
```

## Self-Review
- **Spec coverage:** The plan covers collapsing the step model to 3 stages, embedding analysis progress in configure, embedding generation progress in outline, removing the standalone progress step, and verifying both file and git flows.
- **Placeholder scan:** The plan contains exact file paths, commands, and code snippets for each task.
- **Type consistency:** The plan consistently uses `currentStage`, `currentStepIndex`, `shouldShowInlineProgress`, and `progressHostStage` as the revised view-state contract.
