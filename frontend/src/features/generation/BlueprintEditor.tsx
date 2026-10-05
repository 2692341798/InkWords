import { CheckCircle2, Map, Save, Search } from 'lucide-react'
import { useState } from 'react'
import { blueprintEvidenceOptions } from './blueprintEvidenceOptions'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import type { BlueprintVolumeInput, SourceLibraryEvidence, SourceRetrievalPlan, TextbookProject, TextbookWorkspace } from '@/services/textbook'

interface BlueprintEditorProps {
  project: TextbookProject
  workspace: TextbookWorkspace
  evidence: SourceLibraryEvidence[]
  isBusy: boolean
  onCreate: (input: { volumes: BlueprintVolumeInput[] }) => Promise<void>
  onApprove: (revisionID: string) => Promise<void>
	onRetrieveEvidence: (query: string) => Promise<SourceRetrievalPlan>
}

interface HydratedBlueprintState {
  volumeTitle: string
  evidenceByChapter: Record<string, string[]>
  criticalClaimsByChapter: Record<string, string>
}

const emptyBlueprintState = (): HydratedBlueprintState => ({
  volumeTitle: '核心学习路径',
  evidenceByChapter: {},
  criticalClaimsByChapter: {},
})

// document_json crosses an HTTP boundary, so restore only the small set of
// author-editable fields after checking their runtime shape. Identity, hashes,
// contract bindings, and approval state continue to come from the server row.
function hydrateBlueprintState(document: Record<string, unknown> | undefined): HydratedBlueprintState {
  if (!document || !Array.isArray(document.volumes) || document.volumes.length === 0) return emptyBlueprintState()
  const firstVolume = document.volumes[0]
  if (!firstVolume || typeof firstVolume !== 'object' || Array.isArray(firstVolume)) return emptyBlueprintState()

  const volume = firstVolume as Record<string, unknown>
  const state = emptyBlueprintState()
  if (typeof volume.title === 'string' && volume.title.trim()) state.volumeTitle = volume.title
  if (!Array.isArray(volume.chapters)) return state

  for (const rawChapter of volume.chapters) {
    if (!rawChapter || typeof rawChapter !== 'object' || Array.isArray(rawChapter)) continue
    const chapter = rawChapter as Record<string, unknown>
    if (typeof chapter.id !== 'string' || !chapter.id.trim()) continue
    state.evidenceByChapter[chapter.id] = Array.isArray(chapter.evidence_ids)
      ? chapter.evidence_ids.filter((id): id is string => typeof id === 'string' && Boolean(id.trim()))
      : []
    state.criticalClaimsByChapter[chapter.id] = Array.isArray(chapter.critical_claims)
      ? chapter.critical_claims
          .map((claim) => claim && typeof claim === 'object' && !Array.isArray(claim) ? (claim as Record<string, unknown>).label : undefined)
          .filter((label): label is string => typeof label === 'string' && Boolean(label.trim()))
          .join('\n')
      : ''
  }
  return state
}

const profileLabels: Record<string, string> = {
  concept: '概念讲透', hands_on: '动手实践', source_walkthrough: '源码导读', project_iteration: '项目迭代', troubleshooting: '排错演练', integration_review: '综合复盘', reference: '参考手册',
}

// BlueprintEditor requires explicit evidence mapping for every existing chapter.
// It never infers an evidence relationship from the chapter title or source order.
export function BlueprintEditor({ project, workspace, evidence: initialEvidence, isBusy, onCreate, onApprove, onRetrieveEvidence }: BlueprintEditorProps) {
	const initialBlueprintState = hydrateBlueprintState(workspace.blueprint?.document_json)
  const [volumeTitle, setVolumeTitle] = useState(initialBlueprintState.volumeTitle)
  const [evidenceByChapter, setEvidenceByChapter] = useState<Record<string, string[]>>(initialBlueprintState.evidenceByChapter)
	const [criticalClaimsByChapter, setCriticalClaimsByChapter] = useState<Record<string, string>>(initialBlueprintState.criticalClaimsByChapter)
	const [candidatesByChapter, setCandidatesByChapter] = useState<Record<string, SourceRetrievalPlan[]>>({})
	const [findingCandidatesFor, setFindingCandidatesFor] = useState<string | null>(null)
	const evidence = blueprintEvidenceOptions(initialEvidence, Object.values(candidatesByChapter).flat(), Object.values(evidenceByChapter).flat())
  const contractsApproved = Boolean(project.approved_book_contract_revision_id && project.approved_style_sheet_revision_id)
  const blueprint = workspace.blueprint
  const approved = Boolean(blueprint && blueprint.id === project.approved_blueprint_revision_id)
  const canDraft = contractsApproved && workspace.chapters.length > 0 && evidence.length > 0
	const claimLines = (chapterID: string) => (criticalClaimsByChapter[chapterID] ?? '').split('\n').map((line) => line.trim()).filter(Boolean)
	const canSubmit = canDraft && workspace.chapters.every((chapter) => (evidenceByChapter[chapter.id]?.length ?? 0) > 0 && claimLines(chapter.id).length > 0)

  const create = () => {
    const volume: BlueprintVolumeInput = {
      id: `volume-${project.id}`, title: volumeTitle.trim(), sort: 1,
		chapters: workspace.chapters.map((chapter) => {
			const evidenceIDs = evidenceByChapter[chapter.id] ?? []
			return {
				id: chapter.id, title: chapter.title, sort: chapter.sort_order, profile: chapter.chapter_profile, evidence_ids: evidenceIDs,
				critical_claims: claimLines(chapter.id).map((label, index) => ({ id: `${chapter.id}-claim-${index + 1}`, label, evidence_ids: evidenceIDs })),
			}
		}),
    }
    void onCreate({ volumes: [volume] }).catch(() => undefined)
  }

	const findCandidates = async (chapterID: string) => {
		const claims = claimLines(chapterID)
		if (claims.length === 0) return
		setFindingCandidatesFor(chapterID)
		try {
			const plans = await Promise.all(claims.map((claim) => onRetrieveEvidence(claim)))
			setCandidatesByChapter((current) => ({ ...current, [chapterID]: plans }))
		} finally {
			setFindingCandidatesFor(null)
		}
	}

  return <Panel className="mt-4 p-5">
    <SectionHeader eyebrow="生成约束 3/3" title="教学蓝图" description="为每一章明确教学类型和证据片段；创建时由服务端绑定当前已批准的教学契约与写作规范。" action={<StatusPill tone={approved ? 'success' : blueprint ? 'warning' : 'default'}>{approved ? '已批准' : blueprint ? '草稿待审' : '未创建'}</StatusPill>} />
    {blueprint ? <p className="mt-3 rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">当前蓝图 r{blueprint.revision_number} · {approved ? '它是样章生成的当前前提。' : '请核对每章证据和学习顺序后再批准。'}</p> : null}
    {!contractsApproved ? <p className="mt-4 rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">请先分别批准 BookContract 与 StyleSheet；蓝图不会绑定未审阅的写作规则。</p> : null}
    {contractsApproved && workspace.chapters.length === 0 ? <p className="mt-4 rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">请先创建至少一章，再决定它需要什么证据与教学方式。</p> : null}
    {contractsApproved && workspace.chapters.length > 0 && evidence.length === 0 ? <p className="mt-4 rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">资料尚未解析为可引用片段，因此不能创建看似有依据、实际不可审计的蓝图。</p> : null}
    {canDraft ? <div className="mt-4 grid gap-3">
      <label className="text-sm"><span className="mb-1 block text-muted-foreground">单元标题</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={volumeTitle} onChange={(event) => setVolumeTitle(event.target.value)} /></label>
      <div className="space-y-2" aria-label="蓝图章节证据映射">{workspace.chapters.map((chapter) => <div key={chapter.id} className="grid gap-3 rounded-md border border-border px-3 py-3 md:grid-cols-[minmax(0,1fr)_minmax(260px,0.9fr)]"><div><p className="text-sm font-medium">{chapter.sort_order}. {chapter.title}</p><p className="mt-1 text-xs text-muted-foreground">{profileLabels[chapter.chapter_profile]} · 至少选择一段；涉及多个步骤时可多选。</p><label className="mt-3 block text-sm"><span className="mb-1 block text-xs text-muted-foreground">本章必须讲清的关键事实（每行一条）</span><textarea aria-label={`${chapter.title} 的关键事实`} className="min-h-20 w-full rounded-md border border-border bg-background px-3 py-2" value={criticalClaimsByChapter[chapter.id] ?? ''} onChange={(event) => setCriticalClaimsByChapter((current) => ({ ...current, [chapter.id]: event.target.value }))} placeholder="例如：GET 会把方法、路径和处理函数交给路由登记流程" /></label><Button type="button" size="sm" variant="outline" className="mt-2 gap-2" disabled={isBusy || claimLines(chapter.id).length === 0 || findingCandidatesFor === chapter.id} onClick={() => void findCandidates(chapter.id)}><Search className="h-3.5 w-3.5" />{findingCandidatesFor === chapter.id ? '正在查找资料…' : '按关键事实查找候选资料'}</Button>{candidatesByChapter[chapter.id]?.length ? <ul className="mt-3 space-y-2 rounded-md border border-dashed border-border p-2 text-xs" aria-label={`${chapter.title} 的候选资料`}>{candidatesByChapter[chapter.id].flatMap((plan) => plan.selected.map((item) => <li key={`${plan.id}-${item.chunk_id}`}><span className="font-medium">{item.document_title} · #{item.ordinal}</span><span className="block text-muted-foreground">{item.source_role === 'primary' ? '主资料' : '已确认官方'} · {item.reasons.join('；')}</span></li>))}{candidatesByChapter[chapter.id].filter((plan) => plan.selected.length === 0).map((plan) => <li key={`${plan.id}-missing`} className="text-muted-foreground">“{plan.query}” 暂无可引用资料：请补充主资料或已确认官方资料后重新检索。</li>)}</ul> : null}</div><fieldset><legend className="sr-only">{chapter.title} 的证据片段</legend><div className="grid gap-1">{evidence.map((item) => { const selected = evidenceByChapter[chapter.id]?.includes(item.id) ?? false; return <label key={item.id} className="flex cursor-pointer items-start gap-2 rounded px-2 py-1 text-sm hover:bg-muted"><input type="checkbox" checked={selected} onChange={() => setEvidenceByChapter((current) => { const selectedIDs = current[chapter.id] ?? []; const nextIDs = selected ? selectedIDs.filter((id) => id !== item.id) : [...selectedIDs, item.id]; return { ...current, [chapter.id]: nextIDs } })} /><span>{item.label}</span></label> })}</div></fieldset></div>)}</div>
      <div className="flex flex-wrap items-center gap-2"><Button type="button" variant="outline" disabled={isBusy || !canSubmit || !volumeTitle.trim()} className="gap-2" onClick={create}><Save className="h-4 w-4" />保存蓝图草稿</Button>{blueprint && !approved ? <Button type="button" disabled={isBusy} className="gap-2" onClick={() => void onApprove(blueprint.id).catch(() => undefined)}><CheckCircle2 className="h-4 w-4" />我已审阅并批准</Button> : null}</div>
      {!canSubmit ? <p className="text-xs text-muted-foreground">每一章都需要选择真实资料片段，并写出至少一条必须讲清的关键事实后才能保存。</p> : null}
    </div> : null}
    <p className="mt-4 flex items-center gap-2 text-xs text-muted-foreground"><Map className="h-4 w-4" />证据选择只记录可审计的片段 ID；正文内容仍在候选稿生成和人工审阅阶段产生。</p>
  </Panel>
}
