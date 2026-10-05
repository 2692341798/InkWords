import { create } from 'zustand'
import { appendRightsAmendment, type RightsAmendmentInput } from '@/services/rightsAmendments'
import type { DelegatedPublicationReviewInput, HumanPublicationReviewInput } from '@/services/textbook'
import { loadBlueprintEvidence } from '@/services/blueprintEvidence'
import { textbookService, type AudienceLevel, type BlueprintVolumeInput, type ChapterProfile, type ChapterWorkspace, type EditorialWorkspace, type RightsStatus, type RightsWorkType, type SampleGenerationPreflight, type SourceKind, type SourceLibraryDocument, type SourceLibraryEvidence, type SourceRetrievalPlan, type TextbookBookBuild, type TextbookGenerationTask, type TextbookProject, type TextbookProjectProgress, type TextbookSourceImportTask, type TextbookWorkspace } from '@/services/textbook'

const sampleTaskStorageKey = (chapterId: string) => `inkwords:textbook:sample-task:${chapterId}`

type TextbookGenerationTaskRef = Pick<TextbookGenerationTask, 'task_id'>

const savedSampleTask = (chapterId: string): TextbookGenerationTaskRef | null => {
  if (typeof window === 'undefined') return null
  const taskId = window.sessionStorage.getItem(sampleTaskStorageKey(chapterId))
  return taskId ? { task_id: taskId } : null
}

interface TextbookState {
  projects: TextbookProject[]
  selectedProject: TextbookProject | null
  selectedWorkspace: TextbookWorkspace | null
  selectedProjectProgress: TextbookProjectProgress | null
  sourceLibrary: SourceLibraryDocument[]
  sourceEvidence: SourceLibraryEvidence[]
  selectedChapterWorkspace: ChapterWorkspace | null
	latestSampleTask: TextbookGenerationTaskRef | null
	latestSourceImportTask: TextbookSourceImportTask | null
	sourceRetrieval: SourceRetrievalPlan | null
	latestBookBuild: TextbookBookBuild | null
	editorialWorkspace: EditorialWorkspace | null
  isLoading: boolean
  error: string | null
  load: () => Promise<void>
  create: (input: { title: string; audience_level: AudienceLevel; primary_source: { kind: SourceKind; locator: string } }) => Promise<void>
  select: (projectId: string) => Promise<void>
  addOfficialSource: (input: { kind: SourceKind; locator: string }) => Promise<void>
	loadGinFixture: () => Promise<void>
	importSourceFile: (sourceId: string, file: File, resolvedVersion?: string) => Promise<void>
	importOfficialWeb: (sourceId: string, allowedPathPrefix: string) => Promise<void>
	retrieveSourceEvidence: (query: string) => Promise<SourceRetrievalPlan>
  createChapter: (input: { title: string; chapterProfile: ChapterProfile }) => Promise<void>
  createBookContract: (input: { reader: { audience: AudienceLevel; known_knowledge: string[]; forbidden_assumptions: string[]; learning_outcomes: string[] }; promise: string; chapter_profiles: ChapterProfile[]; terminology_version: string; publication_profile: string }) => Promise<void>
  createStyleSheet: (input: { language: string; terminology_rules: string[]; code_rules: string[]; visual_rules: string[]; citation_rules: string[]; forbidden_phrases: string[] }) => Promise<void>
  approveBookContract: (revisionId: string) => Promise<void>
  approveStyleSheet: (revisionId: string) => Promise<void>
  createBlueprint: (input: { volumes: BlueprintVolumeInput[] }) => Promise<void>
  approveBlueprint: (revisionId: string) => Promise<void>
  openChapter: (chapterId: string) => Promise<void>
  closeChapter: () => void
  acquireChapterLock: (ownerId: string) => Promise<void>
  saveChapterDraft: (input: { markdown: string; documentJson: Record<string, unknown>; contentHash: string; ownerId: string }) => Promise<void>
  applyCandidate: (input: { candidateRevisionId: string; ownerId: string; reviewNote: string; reviewerKind?: "human" | "delegated_ai"; delegationNote?: string; dimensionScores: Array<{ dimension: string; score: number }> }) => Promise<void>
	rejectCandidate: (input: { candidateRevisionId: string; ownerId: string; reason: string }) => Promise<void>
	prepareSampleGeneration: () => Promise<SampleGenerationPreflight>
	generateSample: (confirmedInputHash: string) => Promise<void>
	uploadVisualAsset: (input: { revisionId: string; evidenceId: string; file: File; altText: string; source: string; generationMethod: string; visualPurpose: 'layout' | 'memory_map' | 'call_stack' | 'network_flow' | 'rendered_ui'; rightsStatus: 'pending' | 'ready' | 'blocked' }) => Promise<void>
	createBookBuild: (notices?: import('@/services/publicationNotices').PublicationNoticeDraft[]) => Promise<void>
	addPublicationRight: (input: { subjectRef: string; workType: RightsWorkType; rightsBasis: string; allowedUse: string; attribution: string; publicationStatus: RightsStatus }) => Promise<void>
	completePublicationReview: (input: HumanPublicationReviewInput) => Promise<void>
	recordDelegatedPublicationReview: (input: DelegatedPublicationReviewInput) => Promise<void>
	appendRightsAmendment: (input: RightsAmendmentInput) => Promise<void>
	promoteBookBuild: () => Promise<void>
}

export const useTextbookStore = create<TextbookState>((set, get) => ({
  projects: [], selectedProject: null, selectedWorkspace: null, selectedProjectProgress: null, sourceLibrary: [], sourceEvidence: [], selectedChapterWorkspace: null, latestSampleTask: null, latestSourceImportTask: null, sourceRetrieval: null, latestBookBuild: null, editorialWorkspace: null, isLoading: false, error: null,
  load: async () => { set({ isLoading: true, error: null }); try { const response = await textbookService.list(); set({ projects: response.data, isLoading: false }) } catch (error) { set({ isLoading: false, error: error instanceof Error ? error.message : '加载教材项目失败' }) } },
  create: async (input) => {
    set({ isLoading: true, error: null })
    try {
      const response = await textbookService.create(input)
      set({ projects: [response.data, ...get().projects] })
      await get().select(response.data.id)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '创建教材项目失败' })
      throw error
    }
  },
  select: async (projectId) => { set({ isLoading: true, error: null, latestBookBuild: null, editorialWorkspace: null }); try { const [workspace, library, evidence, progress] = await Promise.all([textbookService.getWorkspace(projectId), textbookService.getSourceLibrary(projectId), textbookService.getSourceEvidence(projectId), textbookService.getProgress(projectId)]); const normalizedWorkspace = { ...workspace.data, sources: workspace.data.sources ?? [], chapters: workspace.data.chapters ?? [] }; const sourceEvidence = await loadBlueprintEvidence(evidence.data ?? [], normalizedWorkspace.blueprint?.document_json, async (ids) => (await textbookService.getSourceEvidence(projectId, ids)).data ?? []); const latestBookBuild = normalizedWorkspace.latest_book_build ?? null; const editorialWorkspace = latestBookBuild ? (await textbookService.getEditorialWorkspace(latestBookBuild.id)).data : null; set({ selectedProject: normalizedWorkspace.project, selectedWorkspace: normalizedWorkspace, selectedProjectProgress: progress.data, sourceLibrary: library.data ?? [], sourceEvidence, latestBookBuild, editorialWorkspace, isLoading: false }) } catch (error) { set({ isLoading: false, error: error instanceof Error ? error.message : '加载教材工作台失败' }) } },
  addOfficialSource: async (input) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.addSource(projectId, { ...input, role: 'official_supporting', official_confirmed: true })
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '添加官方资料失败' })
      throw error
    }
  },
	loadGinFixture: async () => {
		const projectId = get().selectedProject?.id
		if (!projectId) throw new Error('请先打开教材项目')
		set({ isLoading: true, error: null })
		try {
			await textbookService.loadGinFixture(projectId)
			await get().select(projectId)
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '载入固定 Gin 样章资料失败' })
			throw error
		}
	},
	importSourceFile: async (sourceId, file, resolvedVersion) => {
		const projectId = get().selectedProject?.id
		if (!projectId) throw new Error('请先打开教材项目')
		if (file.size < 1) throw new Error('不能导入空文件')
		if (file.size > 888 * 1024 * 1024) throw new Error('单个资料文件不能超过 888 MiB')
		set({ isLoading: true, error: null, latestSourceImportTask: null })
		try {
			const response = await textbookService.createSourceImport(projectId, { source_id: sourceId, file, resolved_version: resolvedVersion })
			set({ latestSourceImportTask: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '创建资料导入任务失败' })
			throw error
		}
	},
	importOfficialWeb: async (sourceId, allowedPathPrefix) => {
		const projectId = get().selectedProject?.id
		if (!projectId) throw new Error('请先打开教材项目')
		const normalizedPrefix = allowedPathPrefix.trim()
		if (!normalizedPrefix.startsWith('/')) throw new Error('抓取路径必须以 / 开头')
		set({ isLoading: true, error: null, latestSourceImportTask: null })
		try {
			const response = await textbookService.createOfficialWebImport(projectId, { source_id: sourceId, allowed_path_prefixes: [normalizedPrefix] })
			set({ latestSourceImportTask: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '创建官网资料导入任务失败' })
			throw error
		}
	},
	retrieveSourceEvidence: async (query) => {
		const projectId = get().selectedProject?.id
		if (!projectId) throw new Error('请先打开教材项目')
		set({ isLoading: true, error: null, sourceRetrieval: null })
		try {
			const response = await textbookService.retrieveSourceEvidence(projectId, { query: query.trim(), limit: 8 })
			set({ sourceRetrieval: response.data, isLoading: false })
			return response.data
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '检索资料失败' })
			throw error
		}
	},
  createChapter: async (input) => {
    const workspace = get().selectedWorkspace
    if (!workspace) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      const lastSortOrder = workspace.chapters.length === 0 ? 0 : Math.max(...workspace.chapters.map((chapter) => chapter.sort_order))
      await textbookService.createChapter(workspace.project.id, { title: input.title, chapter_profile: input.chapterProfile, sort_order: lastSortOrder + 1 })
      await get().select(workspace.project.id)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '创建章节失败' })
      throw error
    }
  },
  createBookContract: async (input) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.createBookContract(projectId, input)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '保存 BookContract 草稿失败' })
      throw error
    }
  },
  createStyleSheet: async (input) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.createStyleSheet(projectId, input)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '保存 StyleSheet 草稿失败' })
      throw error
    }
  },
  approveBookContract: async (revisionId) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.approveBookContract(projectId, revisionId)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '批准 BookContract 失败' })
      throw error
    }
  },
  approveStyleSheet: async (revisionId) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.approveStyleSheet(projectId, revisionId)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '批准 StyleSheet 失败' })
      throw error
    }
  },
  createBlueprint: async (input) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.createBlueprint(projectId, input)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '保存教学蓝图草稿失败' })
      throw error
    }
  },
  approveBlueprint: async (revisionId) => {
    const projectId = get().selectedProject?.id
    if (!projectId) throw new Error('请先打开教材项目')
    set({ isLoading: true, error: null })
    try {
      await textbookService.approveBlueprint(projectId, revisionId)
      await get().select(projectId)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '批准教学蓝图失败' })
      throw error
    }
  },
  openChapter: async (chapterId) => {
    set({ isLoading: true, error: null })
    try {
      const response = await textbookService.getChapterWorkspace(chapterId)
      const latestSampleTask = response.data.latest_sample_task ?? savedSampleTask(chapterId)
      if (latestSampleTask && typeof window !== 'undefined') {
        window.sessionStorage.setItem(sampleTaskStorageKey(chapterId), latestSampleTask.task_id)
      }
      set({ selectedChapterWorkspace: response.data, latestSampleTask, isLoading: false })
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '加载章节编辑器失败' })
    }
  },
  closeChapter: () => set({ selectedChapterWorkspace: null }),
  acquireChapterLock: async (ownerId) => {
    const chapter = get().selectedChapterWorkspace?.chapter
    if (!chapter) throw new Error('请先打开章节')
    set({ isLoading: true, error: null })
    try {
      await textbookService.acquireChapterLock(chapter.id, { owner_id: ownerId, expected_version: chapter.revision_version, lease_seconds: 900 })
      await get().openChapter(chapter.id)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '获取编辑锁失败' })
      throw error
    }
  },
  saveChapterDraft: async (input) => {
    const chapterWorkspace = get().selectedChapterWorkspace
    if (!chapterWorkspace?.lock || chapterWorkspace.lock.owner_id !== input.ownerId) throw new Error('请先获取当前章节的编辑锁')
    set({ isLoading: true, error: null })
    try {
      await textbookService.appendDraftRevision(chapterWorkspace.chapter.id, { expected_version: chapterWorkspace.chapter.revision_version, markdown: input.markdown, document_json: input.documentJson, content_hash: input.contentHash, lock_owner_id: input.ownerId, lock_version: chapterWorkspace.lock.version })
      await get().openChapter(chapterWorkspace.chapter.id)
      if (get().selectedProject) await get().select(get().selectedProject!.id)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '保存章节草稿失败' })
      throw error
    }
  },
  applyCandidate: async (input) => {
    const chapterWorkspace = get().selectedChapterWorkspace
    if (!chapterWorkspace?.lock || chapterWorkspace.lock.owner_id !== input.ownerId) throw new Error('请先获取当前章节的编辑锁')
    set({ isLoading: true, error: null })
    try {
      await textbookService.applyCandidate(chapterWorkspace.chapter.id, input.candidateRevisionId, { expected_version: chapterWorkspace.chapter.revision_version, lock_owner_id: input.ownerId, lock_version: chapterWorkspace.lock.version, review_note: input.reviewNote, dimension_scores: input.dimensionScores, ...(input.reviewerKind ? {reviewer_kind: input.reviewerKind, delegation_note: input.delegationNote} : {}) })
      await get().openChapter(chapterWorkspace.chapter.id)
      if (get().selectedProject) await get().select(get().selectedProject!.id)
    } catch (error) {
      set({ isLoading: false, error: error instanceof Error ? error.message : '应用候选稿失败' })
      throw error
    }
  },
	rejectCandidate: async (input) => {
		const chapterWorkspace = get().selectedChapterWorkspace
		if (!chapterWorkspace?.lock || chapterWorkspace.lock.owner_id !== input.ownerId) throw new Error('请先获取当前章节的编辑锁')
		set({ isLoading: true, error: null })
		try {
			await textbookService.rejectCandidate(chapterWorkspace.chapter.id, input.candidateRevisionId, { expected_version: chapterWorkspace.chapter.revision_version, lock_owner_id: input.ownerId, lock_version: chapterWorkspace.lock.version, reason: input.reason.trim() })
			await get().openChapter(chapterWorkspace.chapter.id)
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '驳回候选稿失败' })
			throw error
		}
	},
	prepareSampleGeneration: async () => {
		const project = get().selectedProject
		const chapter = get().selectedChapterWorkspace?.chapter
		if (!project || !chapter) throw new Error('请先打开要生成的章节')
		set({ isLoading: true, error: null })
		try {
			const response = await textbookService.prepareSampleGeneration(project.id, chapter.id)
			set({ isLoading: false })
			return response.data
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '生成前预检失败' })
			throw error
		}
	},
	generateSample: async (confirmedInputHash) => {
		const project = get().selectedProject
		const chapter = get().selectedChapterWorkspace?.chapter
		if (!project || !chapter) throw new Error('请先打开要生成的章节')
		set({ isLoading: true, error: null, latestSampleTask: null })
		try {
			const response = await textbookService.generateSample(project.id, chapter.id, confirmedInputHash)
			window.sessionStorage.setItem(sampleTaskStorageKey(chapter.id), response.data.task_id)
			set({ latestSampleTask: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '创建样章生成任务失败' })
			throw error
		}
	},
	uploadVisualAsset: async (input) => {
		const chapter = get().selectedChapterWorkspace?.chapter
		if (!chapter) throw new Error('请先打开要登记截图的章节')
		set({ isLoading: true, error: null })
		try {
			await textbookService.uploadVisualAsset(chapter.id, input)
			await get().openChapter(chapter.id)
			set({ isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '登记截图资产失败' })
			throw error
		}
	},
	createBookBuild: async (notices = []) => {
		const projectId = get().selectedProject?.id
		if (!projectId) throw new Error('请先打开教材项目')
		set({ isLoading: true, error: null })
		try {
			const response = await textbookService.createBookBuild(projectId, notices)
			await get().select(projectId)
			set({ latestBookBuild: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '冻结整书构建失败' })
			throw error
		}
	},
	addPublicationRight: async (input) => {
		const build = get().latestBookBuild
		if (!build) throw new Error('请先冻结待审构建')
		set({ isLoading: true, error: null })
		try {
			await textbookService.addPublicationRight(build.id, { subject_ref: input.subjectRef.trim(), work_type: input.workType, rights_basis: input.rightsBasis.trim(), allowed_use: input.allowedUse.trim(), attribution: input.attribution.trim(), publication_status: input.publicationStatus })
			const response = await textbookService.getEditorialWorkspace(build.id)
			set({ editorialWorkspace: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '登记权利项失败' })
			throw error
		}
	},
	completePublicationReview: async (input) => {
		const build = get().latestBookBuild
		if (!build) throw new Error('请先冻结待审构建')
		set({ isLoading: true, error: null })
		try {
			await textbookService.completePublicationReview(build.id, input)
			const response = await textbookService.getEditorialWorkspace(build.id)
			set({ editorialWorkspace: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '记录人工审校失败' })
			throw error
		}
	},
	recordDelegatedPublicationReview: async (input) => {
		const build = get().latestBookBuild
		if (!build) throw new Error('请先冻结待审构建')
		set({ isLoading: true, error: null })
		try {
			await textbookService.recordDelegatedPublicationReview(build.id, input)
			const response = await textbookService.getEditorialWorkspace(build.id)
			set({ editorialWorkspace: response.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '记录委托 AI 审阅失败' })
			throw error
		}
	},
	appendRightsAmendment: async (input) => {
		const build = get().latestBookBuild
		if (!build) throw new Error('请先冻结待审构建')
		set({ isLoading: true, error: null })
		try {
			await appendRightsAmendment(build.id, input)
			const response = await textbookService.getEditorialWorkspace(build.id)
			if (get().latestBookBuild?.id === build.id) set({ editorialWorkspace: response.data })
		} catch (error) {
			set({ error: error instanceof Error ? error.message : '保存补证失败' })
			throw error
		} finally { set({ isLoading: false }) }
	},
	promoteBookBuild: async () => {
		const build = get().latestBookBuild
		if (!build) throw new Error('请先冻结待审构建')
		set({ isLoading: true, error: null })
		try {
			const promoted = await textbookService.promoteBookBuild(build.id)
			const editorial = await textbookService.getEditorialWorkspace(build.id)
			set({ latestBookBuild: promoted.data, editorialWorkspace: editorial.data, isLoading: false })
		} catch (error) {
			set({ isLoading: false, error: error instanceof Error ? error.message : '标记出版候选失败' })
			throw error
		}
	},
}))
