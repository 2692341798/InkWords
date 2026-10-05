import { afterEach, expect, it, vi } from 'vitest'
import { dependencyProjectionService, type DependencyProjectionPreview } from './dependencyProjection'

afterEach(()=>vi.unstubAllGlobals())
it('sends only candidate references and the exact preview confirmation through encoded chapter routes',async()=>{
  const fetch=vi.fn().mockImplementation(async()=>new Response(JSON.stringify({code:0,data:[]}),{status:200,headers:{'Content-Type':'application/json'}}))
  vi.stubGlobal('fetch',fetch)
  await dependencyProjectionService.options('chapter/one')
  expect(fetch.mock.calls[0][0]).toBe('/api/v1/textbook-projects/chapters/chapter%2Fone/dependency-projection/options')
  const request={option_id:'prepared',task_id:'task',revision_id:'revision',content_hash:'hash'}
  await dependencyProjectionService.preview('chapter/one',request)
  expect(JSON.parse(fetch.mock.calls[1][1].body)).toEqual(request)
  await dependencyProjectionService.apply('chapter/one',{request,confirmation_hash:'sha256:exact'} as DependencyProjectionPreview)
  expect(JSON.parse(fetch.mock.calls[2][1].body)).toEqual({...request,confirmed_preview_hash:'sha256:exact'})
})
