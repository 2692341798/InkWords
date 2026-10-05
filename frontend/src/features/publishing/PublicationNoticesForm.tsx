import { useState } from 'react'
import { Button } from '@/components/ui/button'
import type { NoticeSubjectOption, PublicationNoticeDraft } from '@/services/publicationNotices'

const control = 'w-full rounded-md border border-border bg-background px-3 py-2'

const emptyNotice = (): PublicationNoticeDraft => ({ title: '', text: '', source_url: '', prepared_by: '', subject_refs: [] })

export function PublicationNoticesForm({ notices, subjects, disabled, onChange, onDirty }: { notices: PublicationNoticeDraft[]; subjects: NoticeSubjectOption[]; disabled: boolean; onChange: (notices: PublicationNoticeDraft[]) => void; onDirty: (dirty: boolean) => void }) {
  const [draft, setDraft] = useState(emptyNotice)
  const update = (next: PublicationNoticeDraft) => { setDraft(next); onDirty(Object.values(next).some(value => typeof value === 'string' ? value.length > 0 : value.length > 0)) }
  return <details className="mt-4 rounded-md border border-border p-4">
    <summary className="cursor-pointer text-sm font-medium">随稿分发声明（{notices.length} 项）</summary>
    <p className="my-3 text-sm text-muted-foreground">在此加入需要随稿提供的版权、许可或来源说明。它们会和批准正文一起冻结，并进入各格式及相关代码目录；权利是否就绪仍需单独审阅。</p>
    <ul aria-label="本次随稿声明" className="space-y-2">{notices.map((notice, index) => <li key={index} className="rounded border border-border p-3 text-sm"><span>{notice.title}</span><p className="text-xs text-muted-foreground">{notice.prepared_by} · 关联 {notice.subject_refs.length} 项作品</p><Button type="button" variant="outline" size="sm" disabled={disabled} onClick={() => onChange(notices.filter((_, i) => i !== index))}>从本次构建移除：{notice.title}</Button></li>)}</ul>
    <form className="mt-4" onSubmit={event => { event.preventDefault(); if (disabled || notices.length >= 16 || draft.subject_refs.length === 0) return; onChange([...notices, draft]); update(emptyNotice()) }}>
      <fieldset disabled={disabled || notices.length >= 16} className="space-y-3">
        <label className="block text-sm">声明标题<input className={control} required maxLength={160} value={draft.title} onChange={event => update({ ...draft, title: event.target.value })} /></label>
        <label className="block text-sm">声明来源链接<input className={control} required type="url" pattern="https://.*" value={draft.source_url} onChange={event => update({ ...draft, source_url: event.target.value })} /></label>
        <label className="block text-sm">声明整理者<input className={control} required maxLength={200} value={draft.prepared_by} onChange={event => update({ ...draft, prepared_by: event.target.value })} /></label>
        <fieldset className="space-y-2"><legend className="text-sm">这份声明适用于哪些作品</legend>{subjects.map(subject => <label key={subject.value} className="flex items-center gap-2 text-sm"><input type="checkbox" checked={draft.subject_refs.includes(subject.value)} onChange={event => update({ ...draft, subject_refs: event.target.checked ? [...draft.subject_refs, subject.value] : draft.subject_refs.filter(value => value !== subject.value) })} />{subject.label}</label>)}{subjects.length === 0 ? <p className="text-xs text-muted-foreground">批准章节后可关联正文；已有冻结构建的教学代码可单独关联。</p> : null}</fieldset>
        <label className="block text-sm">完整声明文字<textarea className={control} required rows={8} maxLength={24000} value={draft.text} onChange={event => update({ ...draft, text: event.target.value })} /></label>
        <div className="flex gap-2"><Button type="submit" disabled={draft.subject_refs.length === 0}>添加到本次分发声明</Button><Button type="button" variant="outline" onClick={() => update(emptyNotice())}>清空未添加内容</Button></div>
      </fieldset>
    </form>
  </details>
}
