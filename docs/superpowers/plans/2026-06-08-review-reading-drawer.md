# Review Reading Drawer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将复习页从压迫感较强的训练仪表盘改成极简阅读优先的工作台，并把原文改为带独立滚动条的抽屉式阅读面板。

**Architecture:** 保留现有复习业务流和接口契约，只重构 `KnowledgeReview` 与 `ReviewSessionCard` 的前端视图层。会话页默认聚焦当前问题、复述输入与反馈，将原文收进可开合的阅读抽屉；抽屉内部用固定高度滚动容器承载正文，避免整页被撑长。

**Tech Stack:** React 18, TypeScript, Tailwind CSS, Vitest, Zustand.

---

### Task 1: 用失败测试锁定“原文抽屉 + 独立滚动”结构

**Files:**
- Modify: `frontend/src/components/review/ReviewSessionCard.test.tsx`
- Test: `frontend/src/components/review/ReviewSessionCard.test.tsx`

- [ ] **Step 1: Write the failing test for drawer-style source reading**

```tsx
it('renders a source drawer trigger and a scrollable source panel', () => {
  const html = renderToStaticMarkup(
    <ReviewSessionCard
      session={{
        session_id: 'session-1',
        status: 'in_progress',
        mode: 'light_recall',
        title: '并发控制与速率限制',
        source_title: 'InkWords 内容生成平台架构解析系列',
        source_preview: '第一段原文。第二段原文。',
        ready_to_answer: true,
        opening_prompt: '先说主线',
        initial_hints: [],
        session_outline: {
          summary: '并发控制摘要',
          main_question: '它主要解决什么问题？',
          core_concepts: ['并发控制'],
          process_steps: [],
          application_cases: [],
          checkpoints: [],
        },
        turn_index: 3,
        turns: [],
      }}
      selectedMode="light_recall"
      latestStageFeedback={null}
      latestHint={null}
      finalFeedback={null}
      onModeChange={() => {}}
      onStartAnswering={() => {}}
      onRespond={async () => {}}
      onRequestHint={async () => {}}
      onFinish={async () => {}}
    />,
  )

  expect(html).toContain('查看原文')
  expect(html).toContain('data-slot="source-drawer-scroll"')
  expect(html).toContain('overflow-y-auto')
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm --prefix frontend test -- ReviewSessionCard`
Expected: FAIL because the current session card does not render a source drawer trigger or a scrollable source panel.

- [ ] **Step 3: Implement the minimal structure to pass**

```tsx
const [isSourceDrawerOpen, setIsSourceDrawerOpen] = useState(false)

<Button variant="outline" onClick={() => setIsSourceDrawerOpen((open) => !open)}>
  {isSourceDrawerOpen ? '收起原文' : '查看原文'}
</Button>

{isSourceDrawerOpen ? (
  <aside className="rounded-[24px] border border-zinc-200 bg-white">
    <div
      data-slot="source-drawer-scroll"
      className="max-h-[28rem] overflow-y-auto whitespace-pre-wrap px-5 py-4 text-sm leading-7 text-zinc-700 custom-scrollbar"
    >
      {session.source_preview || session.session_outline.summary}
    </div>
  </aside>
) : null}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm --prefix frontend test -- ReviewSessionCard`
Expected: PASS

### Task 2: 重构复习主卡为极简阅读工作台

**Files:**
- Modify: `frontend/src/components/review/ReviewSessionCard.tsx`
- Modify: `frontend/src/pages/KnowledgeReview.tsx`
- Test: `frontend/src/components/review/ReviewSessionCard.test.tsx`

- [ ] **Step 1: Add a failing test for the lighter document-workbench tone**

```tsx
it('renders a reading-first workspace instead of a dark training hero', () => {
  const html = renderToStaticMarkup(/* ...session props... */)
  expect(html).toContain('阅读工作台')
  expect(html).toContain('当前问题')
  expect(html).not.toContain('训练工作台')
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm --prefix frontend test -- ReviewSessionCard`
Expected: FAIL because the current component still renders the dark training-workbench presentation.

- [ ] **Step 3: Implement the minimal reading-first layout**

```tsx
<section data-slot="review-reading-workspace" className="rounded-[32px] border border-zinc-200 bg-white shadow-[0_20px_60px_rgba(15,23,42,0.06)]">
  <div className="border-b border-zinc-100 px-6 py-5">
    <p className="text-xs font-medium tracking-[0.24em] text-zinc-500">阅读工作台</p>
    <h2 className="mt-2 text-2xl font-semibold text-zinc-950">{session.title}</h2>
  </div>
</section>
```

- [ ] **Step 4: Reshape `KnowledgeReview` to match the quieter document-shell**

```tsx
<div className="flex-1 overflow-y-auto bg-[#f7f6f3] custom-scrollbar">
  <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-8">
    <section className="rounded-[32px] border border-stone-200 bg-white px-8 py-8 shadow-[0_18px_48px_rgba(15,23,42,0.05)]">
      <span className="inline-flex items-center rounded-full bg-stone-100 px-3 py-1 text-xs text-stone-600">知识漫游复习</span>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight text-stone-950">像整理笔记一样，把重点重新讲出来</h1>
    </section>
  </div>
</div>
```

- [ ] **Step 5: Run the focused frontend suite**

Run: `npm --prefix frontend test -- ReviewSessionCard KnowledgeReview`
Expected: PASS

### Task 3: 回归验证与文档同步

**Files:**
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

- [ ] **Step 1: Verify the broader review-related frontend tests**

Run: `npm --prefix frontend test -- ReviewSessionCard useKnowledgeReview review`
Expected: PASS

- [ ] **Step 2: Update the development log**

```md
### [2026-06-08] Polish - 复习页切回极简阅读优先，并将原文改为抽屉式滚动面板
- `ReviewSessionCard` 改为浅色文档工作台，原文通过抽屉式面板展开，正文容器带独立滚动条。
```

- [ ] **Step 3: Update the conversation log**

```md
### 对话 125：复习页从训练仪表盘回调为极简阅读工作台
- 用户明确要求“页面很丑陋，原文模块应该加上滚动条”，并确认采用“极简阅读优先 + 原文抽屉式”设计。
```

- [ ] **Step 4: Run a final sanity check**

Run: `git diff -- frontend/src/components/review/ReviewSessionCard.tsx frontend/src/pages/KnowledgeReview.tsx frontend/src/components/review/ReviewSessionCard.test.tsx .trae/documents/InkWords_Development_Plan_and_Log.md .trae/documents/InkWords_Conversation_Log.md`
Expected: only the reading-workspace redesign, source drawer, and documentation updates appear in the diff.
