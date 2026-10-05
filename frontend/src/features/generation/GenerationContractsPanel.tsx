import { useState } from 'react'
import { CheckCircle2, FileText, Save } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import type { AudienceLevel, ChapterProfile, GenerationContractRevision, TextbookProject } from '@/services/textbook'

interface GenerationContractsPanelProps {
  project: TextbookProject
  bookContract?: GenerationContractRevision
  styleSheet?: GenerationContractRevision
  isBusy: boolean
  onCreateBookContract: (input: { reader: { audience: AudienceLevel; known_knowledge: string[]; forbidden_assumptions: string[]; learning_outcomes: string[] }; promise: string; chapter_profiles: ChapterProfile[]; terminology_version: string; publication_profile: string }) => Promise<void>
  onCreateStyleSheet: (input: { language: string; terminology_rules: string[]; code_rules: string[]; visual_rules: string[]; citation_rules: string[]; forbidden_phrases: string[] }) => Promise<void>
  onApproveBookContract: (revisionID: string) => Promise<void>
  onApproveStyleSheet: (revisionID: string) => Promise<void>
}

const lines = (value: string) => value.split('\n').map((item) => item.trim()).filter(Boolean)

const readerDefaults: Record<AudienceLevel, { known: string; forbidden: string }> = {
  foundation: { known: '', forbidden: '不假设读者知道命令行、包管理、HTTP、并发或框架术语。' },
  programming: { known: '变量、函数、条件和循环', forbidden: '不假设读者已经使用过本技术栈。' },
  stack_familiar: { known: '本技术栈的基本项目结构与调试方式', forbidden: '不假设读者熟悉本项目的具体实现与取舍。' },
}

export function GenerationContractsPanel({ project, bookContract, styleSheet, isBusy, onCreateBookContract, onCreateStyleSheet, onApproveBookContract, onApproveStyleSheet }: GenerationContractsPanelProps) {
  const defaults = readerDefaults[project.audience_level]
  const [book, setBook] = useState({ promise: `帮助读者从真实问题出发，独立理解并运用 ${project.title}。`, outcomes: '能用自己的话解释核心原理\n能补全最小示例代码\n能定位常见故障', terminology: 'terms-v1', publication: 'personal_learning', known: defaults.known, forbidden: defaults.forbidden })
  const [style, setStyle] = useState({ terminology: '术语首次出现时先用白话解释，再给出专业名\n每个类比都要写清对应关系和失效边界', code: '代码块标注来源或教学工件身份\n未经验证的代码必须标记验证状态', visual: '运行结果只能使用已验证的文字输出或截图\n图示说明观察对象与限制', citation: '关键事实必须带证据引用\n只使用主资料和已确认官方资料', forbidden: '显然\n很简单\n留给读者\n顾名思义' })
  const bookApproved = Boolean(bookContract && bookContract.id === project.approved_book_contract_revision_id)
  const styleApproved = Boolean(styleSheet && styleSheet.id === project.approved_style_sheet_revision_id)

  return <section className="mt-4 grid gap-4 xl:grid-cols-2">
    <Panel className="p-5">
      <SectionHeader eyebrow="生成约束 1/2" title="BookContract" description="先固定读者起点、学习成果与出版目标，再允许生成候选稿。" action={<StatusPill tone={bookApproved ? 'success' : bookContract ? 'warning' : 'default'}>{bookApproved ? '已批准' : bookContract ? '草稿待审' : '未创建'}</StatusPill>} />
      {bookContract ? <p className="mt-3 rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">当前草稿 r{bookContract.revision_number} · {bookApproved ? '它就是当前生成依据。' : '它尚未成为生成依据。保存新草稿会创建新 revision。'}</p> : null}
      <form className="mt-4 grid gap-3" onSubmit={(event) => { event.preventDefault(); void onCreateBookContract({ reader: { audience: project.audience_level, known_knowledge: lines(book.known), forbidden_assumptions: lines(book.forbidden), learning_outcomes: lines(book.outcomes) }, promise: book.promise.trim(), chapter_profiles: ['concept', 'hands_on'], terminology_version: book.terminology.trim(), publication_profile: book.publication.trim() }).catch(() => undefined) }}>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">教学承诺</span><textarea required className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" value={book.promise} onChange={(event) => setBook((current) => ({ ...current, promise: event.target.value }))} /></label>
        <label className="text-sm"><span className="mb-1 block text-muted-foreground">学习成果（每行一项）</span><textarea required className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" value={book.outcomes} onChange={(event) => setBook((current) => ({ ...current, outcomes: event.target.value }))} /></label>
        <details className="rounded-md border border-border px-3 py-2 text-sm"><summary className="cursor-pointer text-muted-foreground">读者已知与不可假设知识</summary><div className="mt-3 grid gap-3"><label><span className="mb-1 block text-xs text-muted-foreground">已知知识（每行一项）</span><textarea className="min-h-16 w-full rounded-md border border-border bg-background px-3 py-2" value={book.known} onChange={(event) => setBook((current) => ({ ...current, known: event.target.value }))} /></label><label><span className="mb-1 block text-xs text-muted-foreground">不可假设知识（每行一项）</span><textarea className="min-h-16 w-full rounded-md border border-border bg-background px-3 py-2" value={book.forbidden} onChange={(event) => setBook((current) => ({ ...current, forbidden: event.target.value }))} /></label></div></details>
        <div className="grid gap-3 sm:grid-cols-2"><label className="text-sm"><span className="mb-1 block text-muted-foreground">术语表版本</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={book.terminology} onChange={(event) => setBook((current) => ({ ...current, terminology: event.target.value }))} /></label><label className="text-sm"><span className="mb-1 block text-muted-foreground">出版目标</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={book.publication} onChange={(event) => setBook((current) => ({ ...current, publication: event.target.value }))} /></label></div>
        <div className="flex flex-wrap gap-2"><Button type="submit" variant="outline" disabled={isBusy} className="gap-2"><Save className="h-4 w-4" />保存草稿</Button>{bookContract && !bookApproved ? <Button type="button" disabled={isBusy} className="gap-2" onClick={() => void onApproveBookContract(bookContract.id).catch(() => undefined)}><CheckCircle2 className="h-4 w-4" />我已审阅并批准</Button> : null}</div>
      </form>
    </Panel>

    <Panel className="p-5">
      <SectionHeader eyebrow="生成约束 2/2" title="StyleSheet" description="固定通俗表达、证据、代码与运行结果的写法，避免每章重新猜测标准。" action={<StatusPill tone={styleApproved ? 'success' : styleSheet ? 'warning' : 'default'}>{styleApproved ? '已批准' : styleSheet ? '草稿待审' : '未创建'}</StatusPill>} />
      {styleSheet ? <p className="mt-3 rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">当前草稿 r{styleSheet.revision_number} · {styleApproved ? '它就是当前生成依据。' : '它尚未成为生成依据。'}</p> : null}
      <form className="mt-4 grid gap-3" onSubmit={(event) => { event.preventDefault(); void onCreateStyleSheet({ language: 'zh-CN', terminology_rules: lines(style.terminology), code_rules: lines(style.code), visual_rules: lines(style.visual), citation_rules: lines(style.citation), forbidden_phrases: lines(style.forbidden) }).catch(() => undefined) }}>
        {([['术语规则', 'terminology'], ['代码与验证规则', 'code'], ['图示与运行结果规则', 'visual'], ['证据引用规则', 'citation'], ['禁止表达', 'forbidden']] as const).map(([label, field]) => <label key={field} className="text-sm"><span className="mb-1 block text-muted-foreground">{label}（每行一项）</span><textarea required={field !== 'visual' && field !== 'forbidden'} className="min-h-16 w-full rounded-md border border-border bg-background px-3 py-2" value={style[field]} onChange={(event) => setStyle((current) => ({ ...current, [field]: event.target.value }))} /></label>)}
        <div className="flex flex-wrap gap-2"><Button type="submit" variant="outline" disabled={isBusy} className="gap-2"><FileText className="h-4 w-4" />保存草稿</Button>{styleSheet && !styleApproved ? <Button type="button" disabled={isBusy} className="gap-2" onClick={() => void onApproveStyleSheet(styleSheet.id).catch(() => undefined)}><CheckCircle2 className="h-4 w-4" />我已审阅并批准</Button> : null}</div>
      </form>
    </Panel>
  </section>
}
