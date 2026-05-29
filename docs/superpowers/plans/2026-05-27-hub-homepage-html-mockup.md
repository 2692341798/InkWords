# Hub Homepage HTML Mockup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone HTML mockup for a process-oriented InkWords hub homepage that helps users choose between blog generation and knowledge review more clearly.

**Architecture:** Use one standalone static HTML file in `frontend/public/` with embedded CSS and lightweight client-side JavaScript. The page keeps one active workflow visible at a time, updates the flow preview and summary panel from in-memory state, and includes compact support/history panels beneath the main workspace.

**Tech Stack:** HTML5, CSS3, vanilla JavaScript

---

### Task 1: Create The Standalone Mockup File

**Files:**
- Create: `frontend/public/hub-homepage-mockup.html`
- Reference: `docs/superpowers/specs/2026-05-27-hub-homepage-process-design.md`

- [ ] **Step 1: Define the HTML document structure**

Create a standalone page with the following high-level sections:

```html
<body>
  <div class="app-shell">
    <header class="hero"></header>
    <section class="decision-center"></section>
    <section class="flow-strip"></section>
    <main class="workspace-grid">
      <section class="workspace-panel"></section>
      <aside class="summary-panel"></aside>
    </main>
    <section class="support-grid"></section>
  </div>
</body>
```

- [ ] **Step 2: Add the design tokens and layout styles**

Define CSS variables and layout rules that keep the page light and process-oriented:

```css
:root {
  --bg: #f6f7fb;
  --surface: #ffffff;
  --surface-soft: #f8f8fc;
  --text: #16181d;
  --muted: #667085;
  --line: #e6e8ef;
  --primary: #5b4bff;
  --primary-soft: #eeebff;
  --success: #1f9d68;
  --shadow: 0 20px 60px rgba(17, 24, 39, 0.08);
}
```

- [ ] **Step 3: Add interactive controls for both workflows**

Implement clickable controls with dataset values for:

```html
<button class="lane-card is-active" data-flow="blog">生成博客</button>
<button class="lane-card" data-flow="review">知识复习</button>

<button class="choice-chip is-active" data-source="github">GitHub 仓库</button>
<button class="choice-chip" data-source="local">本地文档</button>

<button class="choice-chip is-active" data-scenario="guide">小白教程</button>
<button class="choice-chip" data-scenario="ebook">电子书解读</button>

<button class="choice-chip is-active" data-entry="today">今日推荐</button>
<button class="choice-chip" data-mode="light">轻提示复述</button>
```

- [ ] **Step 4: Add the render functions**

Use one state object and render helpers:

```js
const state = {
  activeFlow: "blog",
  blog: { source: "github", scenario: "guide" },
  review: { entry: "today", mode: "light" }
};

function renderFlowSteps() {}
function renderWorkspace() {}
function renderSummary() {}
```

- [ ] **Step 5: Verify the page manually**

Open the HTML file in a browser and verify:
- switching between `生成博客` and `知识复习` changes the main workspace
- the flow strip updates with the selected path
- the right summary panel updates after each choice
- only one workflow is expanded at a time

- [ ] **Step 6: Commit**

```bash
git add docs/superpowers/plans/2026-05-27-hub-homepage-html-mockup.md frontend/public/hub-homepage-mockup.html
git commit -m "feat: add hub homepage html mockup"
```

### Task 2: Validate Visual Hierarchy And Readability

**Files:**
- Modify: `frontend/public/hub-homepage-mockup.html`

- [ ] **Step 1: Check the hierarchy against the spec**

Confirm the page reads top-to-bottom as:

```text
Hero -> Decision Center -> Flow Strip -> Active Workspace -> Support Area
```

- [ ] **Step 2: Check the copy density**

Keep the main labels concise and Chinese-first:

```text
生成博客
知识复习
选择来源
选择写作场景
开始解析并生成
开始本次复习
```

- [ ] **Step 3: Re-test the interaction states**

Verify these states are clear:
- first visit with `生成博客` selected
- switched to `知识复习`
- changed blog source and scenario
- changed review entry and mode

- [ ] **Step 4: Commit**

```bash
git add frontend/public/hub-homepage-mockup.html
git commit -m "refactor: polish hub homepage html mockup hierarchy"
```
