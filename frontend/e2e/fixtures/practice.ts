import type { PracticeTask } from '../../src/services/textbook'

export const practiceSet = {
  version: 'inkwords.practice-set.v1' as const,
  tasks: (['explain', 'complete', 'reproduce', 'transfer', 'diagnose', 'retain'] as const).map((mode): PracticeTask => ({
    id: `gin-${mode}`, mode,
    prompt: mode === 'explain' ? '说明登记路由与请求查找的两条调用链，并指出方法和路径的作用。' : `${mode}：改变 /orders 的方法或前缀，写出你的实现和判断依据。`,
    variation: '请求缺少 /api 前缀时，处理函数会被调用吗？',
    expected_answer: '登记阶段写入方法与完整路径，请求阶段按方法选择树、按路径查找处理函数。',
    rubric: (mode === 'diagnose' ? ['hypothesis', 'localization', 'evidence', 'root_cause', 'fix_verification'] : ['complete', 'reproduce', 'transfer'].includes(mode) ? ['correctness', 'completeness', 'runtime', 'tests', 'design'] : ['accuracy', 'completeness', 'causality', 'boundaries', 'clarity']).map((id) => ({ id, description: `${id}：解释完整路径、方法与处理函数的关系。`, requires_runtime: ['runtime', 'tests', 'fix_verification'].includes(id) })),
    hints: [{ level: 1, text: '先区分服务启动前后两个时间点。' }, { level: 2, text: '比较方法树和路径分段。' }, { level: 3, text: '沿登记链与查找链检查共享的数据。' }],
    evidence_ids: ['evidence:gin-routing'], min_delay_hours: mode === 'retain' ? 24 : 0,
  })),
}
