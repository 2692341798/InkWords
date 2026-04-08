# Dashboard 技术栈分布饼图改造设计文档

## 1. 背景与目标
目前 Dashboard 页面 (`src/components/Dashboard.tsx`) 的“技术栈涉及频率分布”模块使用的是 `react-wordcloud` 词云组件。由于技术栈类别较多，词云显示过于杂乱，无法直观反映核心数据的比例分布。
**目标**：将词云组件替换为基于 `recharts` 的饼图组件（Donut 环形图形式），并限制最高只展示 15 个类别（前 14 个技术栈 + “其它”），以优化视觉体验和数据可读性。

## 2. 架构与数据处理
- **数据源**：`stats.tech_stack_stats` (类型为 `{ name: string, count: number }[]`)
- **数据转换逻辑**：
  1. 将数组按照 `count` 字段进行降序排序。
  2. 若类别总数 $\le$ 15，则直接全量展示。
  3. 若类别总数 $>$ 15，则截取前 14 项保留原名；将第 15 项及之后的所有分类的 `count` 累加，生成一条新的记录 `{ name: "其它", count: sum }`，作为第 15 项附加到数组末尾。
- **渲染组件**：引入项目中已安装的 `recharts` 库，使用 `<ResponsiveContainer>`, `<PieChart>`, `<Pie>`, `<Cell>`, `<Tooltip>` 和 `<Legend>` 构建图表。

## 3. UI 细节与设计
- **图表类型**：Donut 环形饼图，通过设置 `innerRadius` (例如 60) 和 `outerRadius` (例如 100) 实现。
- **颜色方案**：沿用当前 `Dashboard.tsx` 中预设的 `COLORS` 数组，为每个 Pie Slice (扇区) 分配固定颜色（使用 `<Cell fill={COLORS[index % COLORS.length]} />`）。
- **交互**：
  - Hover 扇区时，通过 `<Tooltip />` 悬浮提示框显示分类名称及具体的统计数量。
  - `<Legend />` 提供图例说明，放置于图表下方或右侧，增强信息辨识度。
- **降级处理**：当 `stats.tech_stack_stats` 为空时，依然展示“暂无技术栈数据”的占位提示。

## 4. 依赖管理
- 项目中已经安装了 `recharts`，可以直接导入使用。
- 改造后将移除 `react-wordcloud` 组件的引用，清理不必要的代码。

## 5. 预期结果与测试
- 图表将清晰地呈现排名前列的技术栈及其比例关系。
- 测试验证数据超过 15 条时的“其它”分类计算是否正确，页面加载时是否报错，以及悬浮提示是否正常渲染。
