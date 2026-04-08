# Dashboard Pie Chart Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing Word Cloud in the Dashboard with a Recharts Donut Pie Chart that displays the top 14 categories and groups the rest into an "其它" (Others) category.

**Architecture:** Modifying `src/components/Dashboard.tsx` to process the `stats.tech_stack_stats` data, removing the `react-wordcloud` import, and rendering a `<PieChart>` from `recharts`.

**Tech Stack:** React 18, Recharts, Tailwind CSS.

---

### Task 1: Replace Word Cloud with Pie Chart in Dashboard

**Files:**
- Modify: `frontend/src/components/Dashboard.tsx`

- [x] **Step 1: Import Recharts components and remove react-wordcloud**

Replace `import ReactWordcloud from 'react-wordcloud'` with Recharts imports.

```tsx
import { useEffect, useState, useMemo } from 'react'
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Coins, FileText, Hash, User, Loader2, Upload, BookOpen } from 'lucide-react'
```

- [x] **Step 2: Add data processing logic using useMemo**

Inside the `Dashboard` component, add a `useMemo` hook to process the data before the `return` statement.

```tsx
  const processedChartData = useMemo(() => {
    if (!stats?.tech_stack_stats || stats.tech_stack_stats.length === 0) {
      return [];
    }

    // Sort descending by count
    const sortedStats = [...stats.tech_stack_stats].sort((a, b) => b.count - a.count);

    if (sortedStats.length <= 15) {
      return sortedStats;
    }

    // Take top 14
    const top14 = sortedStats.slice(0, 14);
    
    // Sum the rest
    const othersCount = sortedStats.slice(14).reduce((sum, item) => sum + item.count, 0);
    
    return [
      ...top14,
      { name: '其它', count: othersCount }
    ];
  }, [stats?.tech_stack_stats]);
```

- [x] **Step 3: Update the Charts Section JSX**

Replace the `<ReactWordcloud ... />` block with the Recharts implementation.

```tsx
        {/* Charts Section */}
        <div className="bg-white p-6 rounded-xl border border-zinc-200 shadow-sm">
          <h2 className="text-lg font-semibold text-zinc-800 mb-6">技术栈涉及频率分布</h2>
          
          <div className="h-[400px] w-full">
            {processedChartData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={processedChartData}
                    cx="50%"
                    cy="50%"
                    innerRadius={80}
                    outerRadius={130}
                    paddingAngle={2}
                    dataKey="count"
                    nameKey="name"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  >
                    {processedChartData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    formatter={(value: number, name: string) => [value, name]}
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e4e4e7', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                  />
                  <Legend verticalAlign="bottom" height={36} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="w-full h-full flex items-center justify-center text-zinc-400 text-sm">
                暂无技术栈数据
              </div>
            )}
          </div>
        </div>
```

- [x] **Step 4: Commit changes**

```bash
cd frontend && git add src/components/Dashboard.tsx
git commit -m "feat: replace word cloud with donut pie chart for top 15 tech stacks"
```

### Task 2: Remove react-wordcloud dependency

**Files:**
- Modify: `frontend/package.json`

- [x] **Step 1: Uninstall react-wordcloud**

```bash
cd frontend && npm uninstall react-wordcloud
```

- [x] **Step 2: Commit changes**

```bash
cd frontend && git add package.json package-lock.json
git commit -m "chore: remove react-wordcloud dependency"
```
