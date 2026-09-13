import type { AssetFontReview } from '@/services/rightsAmendments'

export interface AssetFontContext {
  asset: { id: string; content_hash: string }
  fonts: Array<{ label: string; file_hash: string; rights_item_id: string; amendment_id: string }>
}

const control = 'w-full rounded-md border border-border bg-background px-3 py-2'

export function AssetFontReviewFields({ context, value, onChange }: {
  context: AssetFontContext
  value?: AssetFontReview
  onChange: (value: AssetFontReview | undefined) => void
}) {
  return <fieldset className="grid gap-3 rounded border border-border p-3">
    <legend>图片内部的文字与字体</legend>
    <p className="text-sm text-muted-foreground">核对整张图片，包括界面、图标中的字形和代码区。这里只记录已核对的字体来源，不授予许可；证据和审阅者使用本次补证记录。选择“尚未核对”会撤下此前版本的字体结论，历史仍保留。</p>
    <label>图片文字核对<select className={control} value={value?.text_status ?? 'not_reviewed'} onChange={(e) => {
      const status = e.target.value
      onChange(status === 'not_reviewed' ? undefined : { contract_version: 'inkwords.asset-font-review.v1', asset_id: context.asset.id, content_hash: context.asset.content_hash, surface: 'raster', text_status: status as AssetFontReview['text_status'], scope: value?.scope ?? '', fonts: [] })
    }}><option value="not_reviewed">尚未核对</option><option value="identified_fonts">有文字，逐项核对字体来源</option><option value="no_text">已检查，整张图片不含文字或字形</option></select></label>
    {value ? <>
      <label>图片字体核对范围<textarea className={control} required minLength={8} maxLength={4000} value={value.scope} onChange={(e) => onChange({ ...value, scope: e.target.value })} /></label>
      {value.text_status === 'identified_fonts' ? <div className="grid gap-2">
        <p className="text-sm">选择已登记且已核对可用的字体。只勾选有证据对应到本图的文件；任何未识别字体都应继续保持尚未核对。</p>
        {context.fonts.length === 0 ? <p role="status">尚无可选字体，请先在权利清单登记并核对字体文件。</p> : context.fonts.map((font) => <label key={font.rights_item_id} className="flex items-start gap-2 text-sm"><input type="checkbox" checked={value.fonts.some((f) => f.rights_item_id === font.rights_item_id)} onChange={(e) => onChange({ ...value, fonts: e.target.checked ? [...value.fonts, { file_hash: font.file_hash, rights_item_id: font.rights_item_id, amendment_id: font.amendment_id }] : value.fonts.filter((f) => f.rights_item_id !== font.rights_item_id) })} />{font.label}</label>)}
      </div> : null}
    </> : null}
  </fieldset>
}
