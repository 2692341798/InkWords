import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import type { ChapterRevision } from '@/services/textbook'
import { dependencyProjectionService, type DependencyOption, type DependencyProjectionPreview } from '@/services/dependencyProjection'

export function DependencyProjectionPanel({chapterId,revisions,disabled,onRegistered}:{chapterId:string;revisions:ChapterRevision[];disabled:boolean;onRegistered?:()=>Promise<void>}) {
  const [options,setOptions] = useState<DependencyOption[]>([])
  const [loaded,setLoaded] = useState(false)
  const [optionId,setOptionId] = useState('')
  const [revisionId,setRevisionId] = useState('')
  const [preview,setPreview] = useState<DependencyProjectionPreview|null>(null)
  const [confirmed,setConfirmed] = useState(false)
  const [busy,setBusy] = useState(false)
  const [error,setError] = useState('')
  const [message,setMessage] = useState('')
  const [reload,setReload] = useState(0)
  const mutation = useRef(false)
  const mounted = useRef(true)
  useEffect(()=>{mounted.current=true;return()=>{mounted.current=false}},[])
  const candidates = revisions.filter(r=>r.kind==='candidate' && r.created_by==='generation' && r.generation_task_id)
  const revision = candidates.find(r=>r.id===revisionId)
  const option = options.find(o=>o.id===optionId)
  const currentPreview = preview && revision && preview.request.option_id===optionId && preview.request.revision_id===revision.id && preview.request.content_hash===revision.content_hash ? preview : null
  useEffect(()=>{
    let disposed=false
    void dependencyProjectionService.options(chapterId).then(response=>{
      if(!disposed){setOptions(response.data);setLoaded(true);setError('')}
    }).catch(reason=>{if(!disposed){setError(reason instanceof Error?reason.message:'依赖目录读取失败');setLoaded(true)}})
    return ()=>{disposed=true}
  },[chapterId,reload])
  const resetPreview=()=>{setPreview(null);setConfirmed(false);setMessage('')}
  const prepare=async()=>{
    if(mutation.current || !revision?.generation_task_id || !option)return
    mutation.current=true;setBusy(true);setError('');resetPreview()
    try {
      const response=await dependencyProjectionService.preview(chapterId,{option_id:option.id,task_id:revision.generation_task_id,revision_id:revision.id,content_hash:revision.content_hash})
      if(response.data.contract!=='inkwords.dependency-projection.v1')throw new Error('工件预览版本不受支持')
      if(mounted.current)setPreview(response.data)
    }catch(reason){setError(reason instanceof Error?reason.message:'预览失败，未登记工件')}
    finally{mutation.current=false;setBusy(false)}
  }
  const apply=async()=>{
    if(mutation.current || !confirmed || !currentPreview)return
    mutation.current=true;setBusy(true);setError('')
    try {
      await dependencyProjectionService.apply(chapterId,currentPreview)
      resetPreview();setMessage('工件已登记，仍为未验证。请在教学代码区单独启动验证。')
      if(mounted.current && onRegistered)await onRegistered()
    }catch(reason){setError(reason instanceof Error?reason.message:'登记响应未确认；请先刷新章节核对工件，不会自动重试')}
    finally{mutation.current=false;setBusy(false)}
  }
  return <section className="mt-6 border-t border-border pt-5" aria-label="离线教学依赖">
    <h3 className="font-medium">为候选准备离线依赖</h3>
    <p className="mt-2 text-sm text-muted-foreground">从本机已准备、绑定本章来源的依赖包生成教学工件。原候选不变，登记不会运行代码，也不代表依赖已获出版许可。</p>
    {!loaded?<p className="mt-2 text-sm" role="status">正在核对本地依赖目录…</p>:options.length===0?<p className="mt-2 text-sm">本章尚无可选的本地依赖包，请先按依赖准备流程配置。</p>:<fieldset disabled={disabled||busy} className="mt-3 grid gap-3">
      <label className="text-sm">本地依赖包<select className="mt-1 w-full rounded border border-border bg-background p-2" value={optionId} onChange={e=>{setOptionId(e.target.value);resetPreview()}}><option value="">请选择已准备的包</option>{options.map(o=><option key={o.id} value={o.id}>{o.title}</option>)}</select></label>
      {option?<div className="break-all text-xs text-muted-foreground"><p>{option.selection.module} {option.selection.version} · {option.selection.toolchain}</p><p>固定源码：{option.selection.snapshot.resolved_version}</p><p>依赖清单：{option.selection.dependency_manifest_hash}</p></div>:null}
      <label className="text-sm">待投影候选<select className="mt-1 w-full rounded border border-border bg-background p-2" value={revisionId} onChange={e=>{setRevisionId(e.target.value);resetPreview()}}><option value="">请选择生成候选</option>{candidates.map(r=><option key={r.id} value={r.id}>候选 r{r.revision_number}</option>)}</select></label>
      {candidates.length===0?<p className="text-sm text-muted-foreground">尚无生成候选。依赖包可以先核对；生成候选后才能预览工件。</p>:null}
      <Button type="button" variant="outline" disabled={!option||!revision} onClick={()=>void prepare()}>预览依赖工件</Button>
      {currentPreview?<div className="rounded border border-border p-3 text-sm"><p>将登记 {currentPreview.file_count} 个文件，共 {currentPreview.total_bytes.toLocaleString()} 字节。</p><p>待验证步骤：{currentPreview.commands.map(c=>c.kind==='browser_page'?`浏览器观察 ${c.browser_path}`:'Go 测试').join('、')}</p><p className="mt-2 break-all font-mono text-xs">确认指纹：{currentPreview.confirmation_hash}</p><label className="mt-3 flex items-start gap-2"><input type="checkbox" checked={confirmed} onChange={e=>setConfirmed(e.target.checked)}/>确认使用上述候选、固定依赖和验证步骤登记未验证工件</label><Button type="button" className="mt-3" disabled={!confirmed} onClick={()=>void apply()}>登记未验证工件</Button></div>:null}
    </fieldset>}
    <Button type="button" variant="ghost" size="sm" className="mt-2" disabled={busy||disabled} onClick={()=>{resetPreview();setOptions([]);setLoaded(false);setReload(n=>n+1)}}>重新核对依赖目录</Button>
    {message?<p role="status" className="mt-2 text-sm">{message}</p>:null}{error?<p role="alert" className="mt-2 text-sm text-destructive">{error}</p>:null}
  </section>
}
