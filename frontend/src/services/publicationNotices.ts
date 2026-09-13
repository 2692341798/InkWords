import type { TextbookBookBuild, TextbookChapter } from './textbook'

export interface PublicationNoticeDraft {
  title: string
  text: string
  source_url: string
  prepared_by: string
  subject_refs: string[]
}

export interface NoticeSubjectOption { value: string; label: string }

export function savedPublicationNotices(build: TextbookBookBuild | null): PublicationNoticeDraft[] {
  const book = build?.manifest_json.book as { publication_notices?: PublicationNoticeDraft[] } | undefined
  return (book?.publication_notices ?? []).map(({ title, text, source_url, prepared_by, subject_refs }) => ({ title, text, source_url, prepared_by, subject_refs: [...subject_refs] }))
}

export function publicationNoticeSubjects(build: TextbookBookBuild | null, chapters: TextbookChapter[]): NoticeSubjectOption[] {
  const result = chapters.filter(chapter => chapter.approved_revision_id).map(chapter => ({ value: `chapter-revision:${chapter.approved_revision_id}`, label: `章节：${chapter.title}` }))
  if (result.length === 0) for (const id of build?.approved_revision_ids ?? []) result.push({ value: `chapter-revision:${id}`, label: `已冻结章节 ${result.length + 1}` })
  const artifacts = build?.manifest_json.code_artifacts as Array<{ id: string }> | undefined
  for (const [index, artifact] of (artifacts ?? []).entries()) result.push({ value: `code-artifact:${artifact.id}`, label: `教学代码：第 ${index + 1} 组` })
  return result
}
