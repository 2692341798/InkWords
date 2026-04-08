# Dashboard Word Cloud Implementation Spec

## 1. 目标与背景
当前用户仪表盘 (Dashboard) 中的“技术栈涉及频率分布”使用的是 Recharts 的 `PieChart`（饼图）。由于技术栈种类繁多，且存在占比极低的长尾项，导致饼图标签拥挤、重叠，严重影响可读性。
目标：将 Dashboard 的饼图替换为动态的**词云图 (Word Cloud)** 组件，并在后端对数据进行 Top 20 过滤，以提供更清晰、专业的数据可视化体验。

## 2. 方案设计
### 2.1 后端接口修改 (Go + Gin)
- **目标文件**: `backend/internal/api/user.go`
- **目标函数**: `GetUserStats`
- **逻辑变更**:
  1. 在遍历汇总 `stackMap` 后，得到完整的 `techStackStats` 切片。
  2. 使用 `sort.Slice` 对 `techStackStats` 根据 `Count` 字段进行降序排序。
  3. 如果 `len(techStackStats) > 20`，则截取前 20 项，过滤掉低频长尾数据。
  4. 原有的 JSON 返回结构保持不变（`tech_stack_stats` 数组），对前端的影响最小化。

### 2.2 前端依赖引入
- **依赖库**: `react-wordcloud` (用于词云渲染), `d3-scale` (用于颜色映射支持，可选/按需)
- **注意**: `react-wordcloud` 需要映射特定的数据格式，即 `{ text: string, value: number }`，而目前后端返回的是 `{ name: string, count: number }`，因此在前端需要做一层数据映射。

### 2.3 前端组件修改 (React)
- **目标文件**: `frontend/src/components/Dashboard.tsx`
- **逻辑变更**:
  1. 移除与 `PieChart`, `Pie`, `Cell`, `Tooltip`, `Legend` 相关的 Recharts 引入。
  2. 引入 `ReactWordcloud` 从 `react-wordcloud` 库中。
  3. 在渲染技术栈的区域，将原有的 `<ResponsiveContainer>` 和 `<PieChart>` 块替换为 `<ReactWordcloud>` 组件。
  4. 在渲染前，使用 `map` 函数将后端的 `tech_stack_stats` (包含 `name`, `count`) 转换为 `react-wordcloud` 所需的 `{ text, value }` 格式。
  5. 配置词云图的 `options` (如 `rotations`, `rotationAngles`, `fontSizes`, 颜色配置等)，使其在界面中展示美观。

## 3. 测试与验证
1. 在终端执行 `npm install react-wordcloud` 安装依赖。
2. 重启后端服务和前端服务 (`docker compose down && docker compose up -d --build`)。
3. 登录进入个人中心，查看图表是否正确渲染为词云图，且数据项是否被限制在最多 20 项以内。
4. 验证悬浮提示是否正常工作（显示具体的博客篇数）。
