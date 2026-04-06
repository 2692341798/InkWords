# Dogfood Report: localhost

| Field | Value |
|-------|-------|
| **Date** | 2026-04-07 |
| **App URL** | http://localhost |
| **Session** | localhost |
| **Scope** | 全站探索性测试 |

## Summary

| Severity | Count |
|----------|-------|
| 🛑 **Critical** | 0 |
| 🟢 **High** | 0 |
| 🟡 **Medium** | 1 |
| 🔵 **Low / UI** | 0 |

### Key Findings
1. **停止生成后大纲状态异常**：在生成过程中点击“停止生成”按钮后，由于大纲仍处于折叠状态，且“展开大纲”按钮没有正确重置或工作，导致用户无法查看大纲。

---

## Detailed Findings

### ISSUE-001: 停止生成后大纲无法正常展开

- **Severity:** 🟡 Medium
- **Category:** UX/UI Logic
- **Repro Video:** N/A

**Description:** 
当开始生成博客时，大纲会自动折叠（预期行为）。但如果在生成过程中点击“停止生成”按钮，生成过程虽然停止了，大纲仍然处于折叠状态，这不符合预期。更严重的是，此时点击“展开大纲”按钮，UI 状态可能未正确同步，导致大纲无法展开或状态混乱。

**Repro Steps:**
1. 输入 Git 仓库链接并解析生成大纲。
2. 点击“开始生成”按钮，大纲自动折叠，显示“生成中...”。
3. 在生成过程中，点击旁边的“停止生成”按钮。
4. 观察大纲状态，发现其仍然折叠，且此时页面上显示了“展开大纲”按钮，但大纲本应该自动展开。

**Screenshot Evidence:**
![issue-001](./screenshots/issue-001.png)

---

## Wrap up
本次针对新增的“大纲折叠”与“停止生成”功能进行了探索性测试，通过前端入口 `http://localhost` 正常访问。大纲能够正常折叠，停止生成功能能够立刻中断流。

但发现一个小瑕疵：停止生成后，大纲面板的“折叠”状态未能自动恢复为“展开”状态。目前已记录为 `ISSUE-001`，后续可在 `Generator.tsx` 中修复 `isOutlineExpanded` 状态同步。
