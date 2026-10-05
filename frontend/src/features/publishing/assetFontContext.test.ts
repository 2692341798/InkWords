import { describe, expect, it } from 'vitest'
import type { EditorialWorkspace, PublicationRightsItem } from '@/services/textbook'
import { assetFontContext } from './assetFontContext'

describe('asset font editing snapshot', () => {
  it('selects only this build’s ready font rights and never follows a later revision implicitly', () => {
    const item = { id: 'asset-rights', subject_ref: 'asset:image-1', work_type: 'screenshot', build_id: 'build', project_id: 'project' } as PublicationRightsItem
    const font = { ...item, id: 'font', subject_ref: `font-file:sha256:${'b'.repeat(64)}`, work_type: 'font', publication_status: 'ready', attribution: '字体来源' }
    const workspace = { build: { manifest_json: { assets: [{ id: 'image-1', kind: 'screenshot', content_hash: `sha256:${'a'.repeat(64)}` }] } }, rights_ledger: { effective_items: [font, { ...font, id: 'pending', publication_status: 'pending' }, { ...font, id: 'foreign', build_id: 'other' }], amendments: [{ id: 'font-v2', base_item_id: 'font', revision: 2 }, { id: 'font-v1', base_item_id: 'font', revision: 1 }] } } as unknown as EditorialWorkspace
    const snapshot = assetFontContext(workspace, item)
    expect(snapshot?.fonts).toHaveLength(1)
    expect(snapshot?.fonts[0].amendment_id).toBe('font-v2')
    workspace.rights_ledger!.amendments[0].id = 'changed'
    expect(snapshot?.fonts[0].amendment_id).toBe('font-v2')
    expect(assetFontContext(workspace, { ...item, subject_ref: 'asset:absent' })).toBeUndefined()
    workspace.build.manifest_json.assets = [{ id: 'image-1', kind: 'recording', content_hash: `sha256:${'a'.repeat(64)}` }]
    expect(assetFontContext(workspace, item)).toBeUndefined()
  })
})
