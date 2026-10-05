// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DependencyProjectionPanel } from './DependencyProjectionPanel'
import { dependencyProjectionService, type DependencyProjectionPreview } from '@/services/dependencyProjection'
import type { ChapterRevision } from '@/services/textbook'

vi.mock('@/services/dependencyProjection',()=>({dependencyProjectionService:{options:vi.fn(),preview:vi.fn(),apply:vi.fn()}}))
const revision:ChapterRevision={id:'revision',chapter_id:'chapter',revision_number:1,kind:'candidate',created_by:'generation',generation_task_id:'task',markdown:'saved',content_hash:'original',document_json:{},created_at:''}
const selection={workspace_id:'workspace',project_id:'project',chapter_id:'chapter',module:'github.com/gin-gonic/gin',version:'v1.12.0',toolchain:'go1.26.8',dependency_manifest_hash:'sha256:bundle',snapshot:{id:'snapshot',locator:'https://github.com/gin-gonic/gin',resolved_version:'fixed-commit',content_hash:'sha256:source'}}
const option={id:'option',title:'Gin 固定依赖包',selection_hash:'sha256:selection',selection}
const preview:DependencyProjectionPreview={contract:'inkwords.dependency-projection.v1',request:{option_id:option.id,task_id:'task',revision_id:revision.id,content_hash:revision.content_hash},artifact_id:'artifact',selection,selection_hash:option.selection_hash,book_contract_hash:'sha256:book',style_sheet_hash:'sha256:style',commands:[{kind:'go_test'},{kind:'browser_page',browser_path:'/orders'}],file_count:20,total_bytes:2048,files_hash:'sha256:files',confirmation_hash:'sha256:confirmed'}

describe('DependencyProjectionPanel',()=>{
  beforeEach(()=>{vi.resetAllMocks();vi.mocked(dependencyProjectionService.options).mockResolvedValue({code:0,data:[option]});vi.mocked(dependencyProjectionService.preview).mockResolvedValue({code:0,data:preview})})
  afterEach(cleanup)
  const prepare=async()=>{
    await screen.findByRole('option',{name:option.title})
    fireEvent.change(screen.getByLabelText('本地依赖包'),{target:{value:option.id}})
    fireEvent.change(screen.getByLabelText('待投影候选'),{target:{value:revision.id}})
    fireEvent.click(screen.getByRole('button',{name:'预览依赖工件'}))
    await screen.findByText(/确认指纹：sha256:confirmed/)
  }
  it('requires explicit preview confirmation, forwards its exact hash, and never automatically retries failure',async()=>{
    const refreshed=vi.fn()
    vi.mocked(dependencyProjectionService.apply).mockRejectedValue(new Error('响应未确认，请先刷新核对'))
    render(<DependencyProjectionPanel chapterId="chapter" revisions={[revision]} disabled={false} onRegistered={refreshed}/>)
    await prepare()
    expect(dependencyProjectionService.preview).toHaveBeenCalledWith('chapter',preview.request)
    expect(screen.getByRole('button',{name:'登记未验证工件'}).matches(':disabled')).toBe(true)
    expect(dependencyProjectionService.apply).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(screen.getByRole('button',{name:'登记未验证工件'}))
    await screen.findByRole('alert')
    expect(dependencyProjectionService.apply).toHaveBeenCalledExactlyOnceWith('chapter',preview)
    expect(refreshed).not.toHaveBeenCalled()
  })
  it('invalidates confirmation when the candidate bytes change',async()=>{
    const {rerender}=render(<DependencyProjectionPanel chapterId="chapter" revisions={[revision]} disabled={false}/>)
    await prepare();fireEvent.click(screen.getByRole('checkbox'))
    rerender(<DependencyProjectionPanel chapterId="chapter" revisions={[{...revision,content_hash:'changed'}]} disabled={false}/>)
    expect(screen.queryByRole('button',{name:'登记未验证工件'})).toBeNull()
    expect(dependencyProjectionService.apply).not.toHaveBeenCalled()
  })
  it('shows a prepared package for an empty chapter without inventing a candidate or starting generation',async()=>{
    render(<DependencyProjectionPanel chapterId="chapter" revisions={[]} disabled={false}/>)
    await waitFor(()=>expect(screen.getByText(/尚无生成候选/)).toBeTruthy())
    expect(screen.getByRole('button',{name:'预览依赖工件'}).matches(':disabled')).toBe(true)
    expect(dependencyProjectionService.preview).not.toHaveBeenCalled()
    expect(dependencyProjectionService.apply).not.toHaveBeenCalled()
  })
})
