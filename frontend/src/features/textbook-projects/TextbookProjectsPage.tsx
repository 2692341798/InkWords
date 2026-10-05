import { useEffect, useState } from 'react'
import { ArrowLeft, ArrowRight, BookOpen, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { PageHeader, PageShell, Panel, StatusPill } from '@/components/ui/workspace'
import { useTextbookStore } from '@/store/textbookStore'
import { textbookService, type AudienceLevel, type SourceKind } from '@/services/textbook'
import { TextbookWorkspacePanel } from '@/features/textbook-projects/TextbookWorkspacePanel'
import { ChapterEditorPanel } from '@/features/textbook-projects/ChapterEditorPanel'
import { GenerationContractsPanel } from '@/features/generation/GenerationContractsPanel'
import { BlueprintEditor } from '@/features/generation/BlueprintEditor'
import { TextbookProgressStages } from '@/features/textbook-projects/TextbookProgressStages'
import { BookBuildPanel } from '@/features/publishing/BookBuildPanel'
import { EditorialWorkspace } from '@/features/publishing/EditorialWorkspace'
import { ProviderConnectionPanel } from '@/features/generation/ProviderConnectionPanel'
import { GenerationStagePanel } from '@/features/generation/GenerationStagePanel'
import { ManuscriptStagePanel } from '@/features/manuscript/ManuscriptStagePanel'
import { SourceLibraryPanel } from '@/features/source-library/SourceLibraryPanel'
import { TodayQueue } from '@/components/learning/TodayQueue'
import { MasterySession } from '@/components/learning/MasterySession'
import { useDueMasteryTasks } from '@/hooks/useDueMasteryTasks'
import { masteryService, type DueMasteryTask } from '@/services/mastery'

const draftKey = 'inkwords:textbook-project-draft:v1'
const wizardSteps = ['目标读者', '主资料', '官方补充', 'BookContract', 'StyleSheet']
const initialDraft = { title: 'Gin 从零自学教材', source: 'https://github.com/gin-gonic/gin', audience: 'foundation' as AudienceLevel, kind: 'git_repository' as SourceKind }

function loadDraft(): typeof initialDraft {
  try {
    return { ...initialDraft, ...(JSON.parse(localStorage.getItem(draftKey) ?? '') as Partial<typeof initialDraft>) }
  } catch {
    return initialDraft
  }
}

export function TextbookProjectsPage() {
  const recordDelegatedPublicationReview = useTextbookStore((state) => state.recordDelegatedPublicationReview)
  const appendRightsAmendment = useTextbookStore((state) => state.appendRightsAmendment)
  const { projects, selectedWorkspace, selectedProjectProgress, sourceLibrary, sourceEvidence, selectedChapterWorkspace, latestSampleTask, latestSourceImportTask, sourceRetrieval, latestBookBuild, editorialWorkspace, isLoading, error, load, create, select, addOfficialSource, loadGinFixture, importSourceFile, importOfficialWeb, retrieveSourceEvidence, createChapter, createBookContract, createStyleSheet, approveBookContract, approveStyleSheet, createBlueprint, approveBlueprint, openChapter, closeChapter, acquireChapterLock, saveChapterDraft, applyCandidate, rejectCandidate, prepareSampleGeneration, generateSample, uploadVisualAsset, createBookBuild, addPublicationRight, completePublicationReview, promoteBookBuild } = useTextbookStore()
  const [draft, setDraft] = useState(loadDraft)
  const [wizardStep, setWizardStep] = useState(0)
  const [wizardError, setWizardError] = useState<string | null>(null)
  const { tasks: dueTasks, loading: isLoadingDueTasks, refresh: refreshDueTasks } = useDueMasteryTasks()
  const [activeMasteryTask, setActiveMasteryTask] = useState<DueMasteryTask | null>(null)

  useEffect(() => { void load() }, [load])
  useEffect(() => { localStorage.setItem(draftKey, JSON.stringify(draft)) }, [draft])

  const advanceToSource = () => {
    if (!draft.title.trim()) {
      setWizardError('请先填写教材名称；内容已保留在本机草稿中。')
      return
    }
    setWizardError(null)
    setWizardStep(1)
  }
  const createProject = async () => {
    setWizardError(null)
    try {
      await create({ title: draft.title.trim(), audience_level: draft.audience, primary_source: { kind: draft.kind, locator: draft.source.trim() } })
      setWizardStep(2)
    } catch {
      // Store-owned error gives the recovery action; this draft remains intact.
    }
  }
  const startApprovedLearning = async () => {
    if (!selectedChapterWorkspace) throw new Error('请先打开一个章节。')
    const response = await textbookService.getApprovedChapterProjections(selectedChapterWorkspace.chapter.id)
    const objective = response.data.learning.objectives[0]
    if (!objective) throw new Error('批准稿没有可创建的学习目标。')
    const stages = response.data.learning.learning_arc.stages
    const practice = response.data.learning.practice_set
    if (response.data.learning.format !== 'inkwords.learning-projection.v2' || practice?.version !== 'inkwords.practice-set.v1') throw new Error('批准稿尚无具体六维题目，请先补齐并审阅教材练习。')
    const criteria = practice.tasks.flatMap((task) => task.rubric.map((criterion) => criterion.description))
    const created = await masteryService.createObjective({
      chapter_id: `approved-revision:${response.data.learning.chapter_id}:${response.data.learning.revision_id}`,
      title: objective.text,
      behavior: objective.text,
      skills: objective.required_modes,
      rubric: criteria,
      key_points: criteria,
      prerequisites: stages.flatMap((stage) => stage.prerequisites ?? []),
      evidence_refs: [`approved-revision:${response.data.learning.revision_id}`, ...new Set(practice.tasks.flatMap((task) => task.evidence_ids))],
    })
    const tasks = await refreshDueTasks()
    const dueTask = tasks.find((task) => task.objective_id === created.id)
    if (!dueTask) throw new Error('学习目标已创建，但首个练习尚未到期；请稍后主动刷新。')
    setActiveMasteryTask(dueTask)
  }

  return <PageShell wide>
    <PageHeader title="教材项目" description="一份母稿，派生自学教材、博客和视频教案。所有内容先从可追溯资料开始。" meta={<StatusPill tone="brand">本地工作区</StatusPill>} />

    <Panel className="mt-6 p-5"><TodayQueue tasks={dueTasks} loading={isLoadingDueTasks} onStartTask={setActiveMasteryTask} /></Panel>
    {activeMasteryTask ? <MasterySession key={`${activeMasteryTask.objective_id}:${activeMasteryTask.skill}`} task={activeMasteryTask} onLoad={masteryService.getWorkspace} onHelp={masteryService.revealHelp} onRecord={masteryService.recordAttempt} onRecorded={() => void refreshDueTasks()} onClose={() => setActiveMasteryTask(null)} /> : null}

    {!selectedWorkspace ? <Panel className="mt-6 p-6">
      <ol className="mb-6 grid gap-2 sm:grid-cols-5" aria-label="教材项目向导">
        {wizardSteps.map((stage, index) => {
          const state = index < wizardStep ? 'complete' : index === wizardStep ? 'current' : 'upcoming'
          return <li key={stage} aria-current={state === 'current' ? 'step' : undefined} className={`rounded-md border px-3 py-2 text-xs ${state === 'complete' ? 'border-[var(--brand)] bg-[var(--brand)]/5 text-foreground' : state === 'current' ? 'border-[var(--brand)] bg-[var(--brand)]/10 font-medium text-foreground' : 'border-border text-muted-foreground'}`}>{index + 1}. {stage}</li>
        })}
      </ol>

      <form className="grid gap-4" onSubmit={(event) => { event.preventDefault(); void createProject() }}>
        {wizardStep === 0 ? <>
          <div><h2 className="text-base font-semibold">先确定你要为谁写</h2><p className="mt-1 text-sm text-muted-foreground">这会决定教材的默认语言、脚手架和后续质量门禁；可在创建后以 revision 继续细化。</p></div>
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_220px]">
            <label className="text-sm"><span className="mb-1 block text-muted-foreground">教材名称</span><input required className="w-full rounded-md border border-border bg-background px-3 py-2" value={draft.title} onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))} /></label>
            <label className="text-sm"><span className="mb-1 block text-muted-foreground">读者版本</span><select className="w-full rounded-md border border-border bg-background px-3 py-2" value={draft.audience} onChange={(event) => setDraft((current) => ({ ...current, audience: event.target.value as AudienceLevel }))}><option value="foundation">零基础</option><option value="programming">有编程基础</option><option value="stack_familiar">熟悉技术栈</option></select></label>
          </div>
          <div className="flex flex-wrap items-center gap-3"><Button type="button" onClick={advanceToSource} className="gap-2">下一步：主资料 <ArrowRight className="h-4 w-4" /></Button><span className="text-xs text-muted-foreground">表单内容会保存在本机。</span></div>
        </> : <>
          <div><h2 className="text-base font-semibold">登记主资料</h2><p className="mt-1 text-sm text-muted-foreground">创建后会立即进入该项目。官方补充资料、教学契约和写作规范均在该项目内继续保存和审批。</p></div>
          <div className="grid gap-3 md:grid-cols-[220px_minmax(0,1fr)]">
            <label className="text-sm"><span className="mb-1 block text-muted-foreground">主资料类型</span><select className="w-full rounded-md border border-border bg-background px-3 py-2" value={draft.kind} onChange={(event) => setDraft((current) => ({ ...current, kind: event.target.value as SourceKind }))}><option value="git_repository">Git 仓库</option><option value="official_web">官网页面</option><option value="pdf">PDF</option><option value="docx">DOCX</option><option value="markdown">Markdown</option><option value="text">TXT</option><option value="zip">ZIP</option></select></label>
            <label className="text-sm"><span className="mb-1 block text-muted-foreground">主资料定位</span><input required type={draft.kind === 'git_repository' || draft.kind === 'official_web' ? 'url' : 'text'} placeholder={draft.kind === 'git_repository' || draft.kind === 'official_web' ? 'https://…' : '本地路径，仅用作来源标识；创建后选择文件导入'} className="w-full rounded-md border border-border bg-background px-3 py-2" value={draft.source} onChange={(event) => setDraft((current) => ({ ...current, source: event.target.value }))} /></label>
          </div>
          <div className="flex flex-wrap items-center gap-3"><Button type="button" variant="outline" onClick={() => setWizardStep(0)} className="gap-2"><ArrowLeft className="h-4 w-4" />返回</Button><Button type="submit" disabled={isLoading} className="gap-2"><Plus className="h-4 w-4" />创建并进入资料工作台</Button><span className="text-xs text-muted-foreground">暂不需要 API Key；失败时保留草稿与输入，修正后可直接重试。</span></div>
        </>}
      </form>
    </Panel> : null}

    {wizardError || error ? <p role="alert" className="mt-4 text-sm text-destructive">{wizardError ?? error}。已保留当前表单内容；请检查提示后重试。</p> : null}
    <TextbookProgressStages progress={selectedProjectProgress} />

    {selectedWorkspace ? <>
      <Panel className="mt-6 p-5"><p className="text-xs text-muted-foreground">当前工作台项目</p><h2 className="mt-1 text-lg font-semibold">{selectedWorkspace.project.title}</h2><p className="mt-2 text-sm text-muted-foreground">服务端状态：{selectedWorkspace.project.status} · 项目版本 {selectedWorkspace.project.revision_version}。章节内容会以服务端 revision 版本执行锁定和 CAS。</p></Panel>
	  <section className="mt-4 grid gap-4 xl:grid-cols-3" aria-label="教材工作台阶段说明"><SourceLibraryPanel /><GenerationStagePanel /><ManuscriptStagePanel /></section>
	  <ProviderConnectionPanel />
      <TextbookWorkspacePanel workspace={selectedWorkspace} isBusy={isLoading} onAddOfficialSource={addOfficialSource} onLoadGinFixture={loadGinFixture} onImportSourceFile={importSourceFile} onImportOfficialWeb={importOfficialWeb} latestSourceImportTask={latestSourceImportTask} onSourceImportSucceeded={() => void select(selectedWorkspace.project.id)} sourceRetrieval={sourceRetrieval} onRetrieveSourceEvidence={retrieveSourceEvidence} onCreateChapter={createChapter} onOpenChapter={openChapter} />
      <GenerationContractsPanel key={selectedWorkspace.project.id} project={selectedWorkspace.project} bookContract={selectedWorkspace.book_contract} styleSheet={selectedWorkspace.style_sheet} isBusy={isLoading} onCreateBookContract={createBookContract} onCreateStyleSheet={createStyleSheet} onApproveBookContract={approveBookContract} onApproveStyleSheet={approveStyleSheet} />
      <BlueprintEditor key={`${selectedWorkspace.project.id}:${selectedWorkspace.blueprint?.id ?? 'new'}`} project={selectedWorkspace.project} workspace={selectedWorkspace} evidence={sourceEvidence} isBusy={isLoading} onCreate={createBlueprint} onApprove={approveBlueprint} onRetrieveEvidence={retrieveSourceEvidence} />
	  <BookBuildPanel key={`${selectedWorkspace.project.id}:${latestBookBuild?.id ?? 'new'}`} build={latestBookBuild} chapters={selectedWorkspace.chapters} disabled={isLoading} onCreate={createBookBuild} />
      {editorialWorkspace ? <EditorialWorkspace workspace={editorialWorkspace} disabled={isLoading} onAddRight={addPublicationRight} onAmendRight={appendRightsAmendment} onCompleteReview={completePublicationReview} onRecordDelegatedReview={recordDelegatedPublicationReview} onPromote={promoteBookBuild} /> : null}
      <Panel className="mt-4 p-5"><div className="flex items-center justify-between gap-3"><div><p className="text-xs text-muted-foreground">资料库</p><h2 className="mt-1 text-lg font-semibold">已解析的可引用资料</h2></div><StatusPill tone={sourceLibrary.length > 0 ? 'success' : 'warning'}>{sourceLibrary.length} 份文档</StatusPill></div>{sourceLibrary.length === 0 ? <p className="mt-4 rounded-md border border-dashed border-border px-3 py-4 text-sm text-muted-foreground">资料已登记，但尚未生成固定快照和结构化片段；因此不会被当作生成证据。</p> : <ul className="mt-4 space-y-2" aria-label="已解析资料库">{sourceLibrary.map((document) => <li key={document.id} className="rounded-md border border-border px-3 py-2"><div className="flex flex-wrap items-center justify-between gap-2"><span className="font-medium text-sm">{document.title}</span><span className="text-xs text-muted-foreground">{document.media_type} · {document.chunk_count} 个片段</span></div><p className="mt-1 break-all text-xs text-muted-foreground">{document.artifact_path || document.canonical_locator}</p></li>)}</ul>}</Panel>
      {selectedChapterWorkspace ? <ChapterEditorPanel key={`${selectedChapterWorkspace.chapter.id}:${selectedChapterWorkspace.chapter.revision_version}:${selectedChapterWorkspace.lock?.version ?? 0}`} workspace={selectedChapterWorkspace} isBusy={isLoading} onAcquireLock={acquireChapterLock} onSaveDraft={saveChapterDraft} onApplyCandidate={applyCandidate} onRejectCandidate={rejectCandidate} onPrepareSampleGeneration={prepareSampleGeneration} onGenerateSample={generateSample} onUploadVisualAsset={uploadVisualAsset} onStartLearning={startApprovedLearning} onDependencyRegistered={() => openChapter(selectedChapterWorkspace.chapter.id)} latestSampleTaskId={latestSampleTask?.task_id ?? null} onClose={closeChapter} /> : null}
    </> : null}

    <section className="mt-6 grid gap-4 md:grid-cols-2">{projects.map((project) => <Panel key={project.id} className="p-5"><div className="flex items-start gap-3"><BookOpen className="mt-1 h-5 w-5 text-[var(--brand)]" /><div className="min-w-0 flex-1"><h2 className="truncate font-semibold">{project.title}</h2><p className="mt-1 text-sm text-muted-foreground">{project.audience_level === 'foundation' ? '零基础' : project.audience_level === 'programming' ? '有编程基础' : '熟悉技术栈'} · {project.status}</p><p className="mt-3 text-xs text-muted-foreground">版本 {project.revision_version} · {new Date(project.updated_at).toLocaleString()}</p><Button className="mt-4" variant="outline" onClick={() => { setWizardStep(2); void select(project.id) }}>打开工作台</Button></div></div></Panel>)}</section>
    {!isLoading && projects.length === 0 ? <Panel className="mt-6 p-8 text-center text-sm text-muted-foreground">还没有教材项目。先用上方两步创建一份主资料明确的教材母稿。</Panel> : null}
  </PageShell>
}
