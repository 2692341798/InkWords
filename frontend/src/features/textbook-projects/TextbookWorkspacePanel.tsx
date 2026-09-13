import { useState } from 'react'
import { DatabaseZap, FilePlus2, Link, ListPlus, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import type { ChapterProfile, SourceKind, SourceRetrievalPlan, TextbookSourceImportTask, TextbookWorkspace } from '@/services/textbook'
import { SourceImportTaskStatusPanel } from './SourceImportTaskStatusPanel'

interface TextbookWorkspacePanelProps {
  workspace: TextbookWorkspace
  isBusy: boolean
	onAddOfficialSource: (input: { kind: SourceKind; locator: string }) => Promise<void>
	onLoadGinFixture: () => Promise<void>
	onImportSourceFile: (sourceId: string, file: File, resolvedVersion?: string) => Promise<void>
	onImportOfficialWeb: (sourceId: string, allowedPathPrefix: string) => Promise<void>
	latestSourceImportTask: TextbookSourceImportTask | null
	onSourceImportSucceeded: () => void
	sourceRetrieval: SourceRetrievalPlan | null
	onRetrieveSourceEvidence: (query: string) => Promise<unknown>
  onCreateChapter: (input: { title: string; chapterProfile: ChapterProfile }) => Promise<void>
  onOpenChapter: (chapterId: string) => Promise<void>
}

const chapterProfiles: Array<{ value: ChapterProfile; label: string }> = [
  { value: 'concept', label: '概念讲透' },
  { value: 'hands_on', label: '动手实践' },
  { value: 'source_walkthrough', label: '源码导读' },
  { value: 'project_iteration', label: '项目迭代' },
  { value: 'troubleshooting', label: '排错演练' },
  { value: 'integration_review', label: '综合复盘' },
  { value: 'reference', label: '参考手册' },
]

const sourceKindLabels: Record<SourceKind, string> = {
  git_repository: 'Git 仓库',
  official_web: '官网页面',
  pdf: 'PDF',
  docx: 'DOCX',
  markdown: 'Markdown',
  text: '文本',
  zip: 'ZIP',
}

const chapterProfileLabel = (profile: ChapterProfile) => chapterProfiles.find((item) => item.value === profile)?.label ?? profile

export function TextbookWorkspacePanel({ workspace, isBusy, onAddOfficialSource, onLoadGinFixture, onImportSourceFile, onImportOfficialWeb, latestSourceImportTask, onSourceImportSucceeded, sourceRetrieval, onRetrieveSourceEvidence, onCreateChapter, onOpenChapter }: TextbookWorkspacePanelProps) {
  const [source, setSource] = useState({ kind: 'official_web' as SourceKind, locator: '' })
	const [gitCommits, setGitCommits] = useState<Record<string, string>>({})
	const [officialPathPrefixes, setOfficialPathPrefixes] = useState<Record<string, string>>({})
  const [retrievalQuery, setRetrievalQuery] = useState('')
  const [chapter, setChapter] = useState({ title: '', chapterProfile: 'concept' as ChapterProfile })
	const hasGinPrimary = workspace.sources.some((source) => source.role === 'primary' && source.kind === 'git_repository' && source.locator.replace(/\/$/, '') === 'https://github.com/gin-gonic/gin')

  return (
    <section className="mt-6 grid gap-4 xl:grid-cols-2">
      <Panel className="p-5">
        <SectionHeader
          eyebrow="资料阶段"
          title="主资料与官方补充"
          description="每一项都由当前本地工作区保存。只有你确认的官方资料才会进入未来的生成证据集。"
          action={<StatusPill tone={workspace.sources.length > 0 ? 'success' : 'warning'}>{workspace.sources.length} 项已登记</StatusPill>}
        />
        <ul className="mt-4 space-y-2" aria-label="已登记资料">
          {workspace.sources.map((item) => (
            <li key={item.id} className="rounded-md border border-border px-3 py-2">
              <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                <span className="font-medium">{item.role === 'primary' ? '主资料' : '官方补充'} · {sourceKindLabels[item.kind]}</span>
                <StatusPill tone={item.role === 'official_supporting' ? 'success' : 'brand'}>{item.role === 'official_supporting' ? '已确认官方' : '主资料'}</StatusPill>
              </div>
              <p className="mt-1 break-all text-xs text-muted-foreground">{item.locator}</p>
				{item.kind === 'git_repository' ? <label className="mt-3 block text-xs"><span className="mb-1 block text-muted-foreground">固定 commit SHA（40 位；文件必须来自此版本）</span><input className="w-full rounded-md border border-border bg-background px-2 py-1 font-mono" aria-label={`${item.locator} 的固定 commit SHA`} value={gitCommits[item.id] ?? ''} onChange={(event) => setGitCommits((current) => ({ ...current, [item.id]: event.target.value.trim() }))} pattern="[0-9a-fA-F]{40}" placeholder="例如：73726dc606796a025971fe451f0aa6f1b9b847f6" /></label> : null}
				{isLocalFileSource(item.kind) ? <label className="mt-3 inline-flex cursor-pointer items-center gap-2 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium hover:bg-muted"><Upload className="h-3.5 w-3.5" />导入固定文件<input className="sr-only" type="file" accept={acceptForSourceKind(item.kind)} disabled={isBusy || (item.kind === 'git_repository' && !/^[0-9a-fA-F]{40}$/.test(gitCommits[item.id] ?? ''))} aria-label={`导入 ${item.locator}`} onChange={(event) => { const file = event.currentTarget.files?.[0]; if (file) { const importTask = item.kind === 'git_repository' ? onImportSourceFile(item.id, file, gitCommits[item.id]) : onImportSourceFile(item.id, file); void importTask.catch(() => undefined) }; event.currentTarget.value = '' }} /></label> : item.kind === 'official_web' && item.role === 'official_supporting' ? <div className="mt-3 grid gap-2"><label className="text-xs"><span className="mb-1 block text-muted-foreground">允许抓取的同站路径边界</span><input className="w-full rounded-md border border-border bg-background px-2 py-1 font-mono" aria-label={`${item.locator} 的允许抓取路径`} value={officialPathPrefixes[item.id] ?? defaultOfficialPathPrefix(item.locator)} onChange={(event) => setOfficialPathPrefixes((current) => ({ ...current, [item.id]: event.target.value }))} /></label><div className="flex flex-wrap items-center gap-2"><Button type="button" size="sm" variant="outline" disabled={isBusy} onClick={() => void onImportOfficialWeb(item.id, officialPathPrefixes[item.id] ?? defaultOfficialPathPrefix(item.locator)).catch(() => undefined)}>抓取并导入官网资料</Button><span className="text-xs text-muted-foreground">只访问 HTTPS 同站、此路径及其子路径；任务完成后才形成不可变快照。</span></div></div> : <p className="mt-3 text-xs text-muted-foreground">官网页面需要先明确确认官方来源，之后才可由受限抓取流程处理。</p>}
            </li>
          ))}
        </ul>
        {hasGinPrimary ? <div className="mt-4 rounded-md border border-[var(--brand)]/30 bg-[var(--brand)]/5 p-3"><p className="text-sm font-medium">Gin 样章离线资料</p><p className="mt-1 text-xs text-muted-foreground">载入同一固定 commit 的路由注册与请求查找源码摘录：GET、handle、addRoute、ServeHTTP、handleHTTPRequest、getValue。每个文件形成独立不可变快照；它只用于首个样章，不替代后续通用资料导入。</p><Button type="button" size="sm" className="mt-3 gap-2" disabled={isBusy} onClick={() => void onLoadGinFixture().catch(() => undefined)}><DatabaseZap className="h-4 w-4" />载入固定 Gin 样章资料</Button></div> : null}
		{latestSourceImportTask ? <SourceImportTaskStatusPanel taskID={latestSourceImportTask.id} onSucceeded={onSourceImportSucceeded} /> : null}
		<form className="mt-5 grid gap-3 border-t border-border pt-5" onSubmit={(event) => { event.preventDefault(); void onRetrieveSourceEvidence(retrievalQuery).catch(() => undefined) }}>
			<p className="text-sm font-medium">检索可引用资料</p>
			<label className="text-sm"><span className="sr-only">检索关键词</span><input required minLength={2} className="w-full rounded-md border border-border bg-background px-3 py-2" value={retrievalQuery} onChange={(event) => setRetrievalQuery(event.target.value)} placeholder="例如：路由 如何匹配处理函数" /></label>
			<div className="flex flex-wrap items-center gap-3"><Button type="submit" variant="outline" disabled={isBusy || workspace.sources.length === 0}>检索并说明理由</Button><span className="text-xs text-muted-foreground">仅匹配已固定快照；最多返回 8 段，不把全文送入模型或浏览器。</span></div>
		</form>
		{sourceRetrieval ? <div className="mt-4 rounded-md border border-border p-3" aria-label="资料检索结果"><div className="flex flex-wrap items-center justify-between gap-2"><p className="text-sm font-medium">“{sourceRetrieval.query}” 的建议证据</p><StatusPill tone="brand">{sourceRetrieval.selected.length} 段</StatusPill></div><ul className="mt-3 space-y-2">{sourceRetrieval.selected.map((item) => <li key={item.chunk_id} className="rounded-md bg-muted/50 px-3 py-2 text-xs"><p className="font-medium">{item.document_title} · 片段 {item.ordinal} · {item.source_role === 'primary' ? '主资料' : '已确认官方'}</p><p className="mt-1 break-all text-muted-foreground">{item.artifact_path || item.canonical_locator}</p><p className="mt-1 text-muted-foreground">匹配理由：{item.reasons.join('；')}（分数 {item.score}）</p></li>)}</ul><p className="mt-3 text-xs text-muted-foreground">检索记录已保存；请在蓝图中人工选择片段，不会自动写入教材。</p></div> : null}
        <form
          className="mt-5 grid gap-3 border-t border-border pt-5"
          onSubmit={(event) => {
            event.preventDefault()
            void onAddOfficialSource({ kind: source.kind, locator: source.locator.trim() })
              .then(() => setSource((current) => ({ ...current, locator: '' })))
              .catch(() => undefined)
          }}
        >
          <p className="text-sm font-medium">登记官方补充资料</p>
          <div className="grid gap-3 md:grid-cols-[160px_minmax(0,1fr)]">
            <label className="text-sm"><span className="sr-only">资料格式</span><select className="w-full rounded-md border border-border bg-background px-3 py-2" value={source.kind} onChange={(event) => setSource((current) => ({ ...current, kind: event.target.value as SourceKind }))}>{(Object.entries(sourceKindLabels) as Array<[SourceKind, string]>).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
            <label className="text-sm"><span className="sr-only">官方资料地址或本地路径</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" placeholder="官网 URL 或本地资料路径" value={source.locator} onChange={(event) => setSource((current) => ({ ...current, locator: event.target.value }))} /></label>
          </div>
          <div className="flex flex-wrap items-center gap-3"><Button type="submit" variant="outline" disabled={isBusy} className="gap-2"><Link className="h-4 w-4" />确认并登记</Button><span className="text-xs text-muted-foreground">解析、抓取与快照固定版本将在下一阶段执行。</span></div>
        </form>
      </Panel>

      <Panel className="p-5">
        <SectionHeader
          eyebrow="章节阶段"
          title="先搭出可自学的学习路径"
          description="章节只保存标题、顺序和教学类型；正文、证据、锁定、差异与候选稿在章节编辑阶段再逐步启用。"
          action={<StatusPill tone={workspace.chapters.length > 0 ? 'success' : 'warning'}>{workspace.chapters.length} 章已建立</StatusPill>}
        />
        <ol className="mt-4 space-y-2" aria-label="教材章节">
          {workspace.chapters.map((item) => (
            <li key={item.id} className="flex items-center gap-3 rounded-md border border-border px-3 py-2 text-sm"><span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-muted text-xs">{item.sort_order}</span><span className="min-w-0 flex-1 truncate font-medium">{item.title}</span><span className="text-xs text-muted-foreground">{chapterProfileLabel(item.chapter_profile)} · r{item.revision_version}</span><Button type="button" size="sm" variant="ghost" disabled={isBusy} onClick={() => void onOpenChapter(item.id)}>编辑</Button></li>
          ))}
          {workspace.chapters.length === 0 ? <li className="rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">尚无章节。先用“概念讲透”把为什么需要这项技术讲明白，再安排实践与排错。</li> : null}
        </ol>
        <form
          className="mt-5 grid gap-3 border-t border-border pt-5"
          onSubmit={(event) => {
            event.preventDefault()
            void onCreateChapter({ title: chapter.title.trim(), chapterProfile: chapter.chapterProfile })
              .then(() => setChapter((current) => ({ ...current, title: '' })))
              .catch(() => undefined)
          }}
        >
          <p className="text-sm font-medium">创建下一章</p>
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px]"><label className="text-sm"><span className="sr-only">章节标题</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" placeholder="例如：一次请求在 Gin 中如何流动" value={chapter.title} onChange={(event) => setChapter((current) => ({ ...current, title: event.target.value }))} /></label><label className="text-sm"><span className="sr-only">教学类型</span><select className="w-full rounded-md border border-border bg-background px-3 py-2" value={chapter.chapterProfile} onChange={(event) => setChapter((current) => ({ ...current, chapterProfile: event.target.value as ChapterProfile }))}>{chapterProfiles.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label></div>
          <div className="flex flex-wrap items-center gap-3"><Button type="submit" variant="outline" disabled={isBusy} className="gap-2"><ListPlus className="h-4 w-4" />创建章节</Button><span className="text-xs text-muted-foreground">顺序会按当前最后一章自动追加；创建后可在后续蓝图阶段调整。</span></div>
        </form>
      </Panel>

      <Panel className="p-5 xl:col-span-2" tone="soft">
          <div className="flex gap-3"><FilePlus2 className="mt-1 h-5 w-5 shrink-0 text-[var(--brand)]" /><div><p className="font-medium">下一步：先固定资料版本，再批准蓝图</p><p className="mt-1 text-sm text-muted-foreground">Gin 首个样章可载入固定离线摘录；其余资料仍必须等待通用导入形成可引用快照。BookContract、StyleSheet、蓝图、Markdown 修订、候选稿差异与人工批准均由服务端版本控制，不会虚报“已验证”。</p></div></div>
      </Panel>
    </section>
  )
}

function isLocalFileSource(kind: SourceKind): boolean {
	return kind === 'git_repository' || kind === 'pdf' || kind === 'docx' || kind === 'markdown' || kind === 'text' || kind === 'zip'
}

function acceptForSourceKind(kind: SourceKind): string {
	switch (kind) {
		case 'git_repository': return '.md,.markdown,.txt,.c,.cpp,.go,.h,.hpp,.java,.js,.jsx,.json,.py,.rs,.sh,.sql,.ts,.tsx,.yaml,.yml,text/plain,text/x-go'
		case 'pdf': return '.pdf,application/pdf'
		case 'docx': return '.docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document'
		case 'markdown': return '.md,.markdown,text/markdown'
		case 'text': return '.txt,text/plain'
		case 'zip': return '.zip,application/zip'
		default: return ''
	}
}

function defaultOfficialPathPrefix(locator: string): string {
	try {
		const path = new URL(locator).pathname
		return path && path !== '/' ? path.replace(/\/$/, '') : '/'
	} catch {
		return '/'
	}
}
