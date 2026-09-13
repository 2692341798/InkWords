import { useState } from 'react'
import { Cable, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'
import { textbookService, type ProviderConnectionResult } from '@/services/textbook'

export function ProviderConnectionPanel() {
  const [result, setResult] = useState<ProviderConnectionResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [testing, setTesting] = useState(false)

  const testConnection = async () => {
    setTesting(true)
    setError(null)
    try {
      const response = await textbookService.testConfiguredProviderConnection()
      if (response.code !== 0) throw new Error('模型连接测试失败')
      setResult(response.data)
    } catch {
      setResult(null)
      // The server deliberately keeps provider details private. Keep the UI
      // boundary equally strict so a future transport failure cannot surface
      // an upstream body, credential, or SDK diagnostic.
      setError('模型连接测试失败；请检查本机提供商配置后重试')
    } finally {
      setTesting(false)
    }
  }

  return <Panel className="mt-4 p-5" aria-label="模型连接测试">
    <SectionHeader eyebrow="本机模型配置" title="测试已配置连接" description="仅在你点击后发送固定 JSON 探测，不发送教材、资料或用户文本；密钥和原始错误不会显示或保存。" action={<StatusPill tone={result ? 'success' : 'warning'}>{result ? '连接成功' : error ? '需要处理' : '尚未测试'}</StatusPill>} />
    <div className="mt-4 flex flex-wrap items-center gap-3"><Button type="button" variant="outline" disabled={testing} onClick={() => void testConnection()} className="gap-2" aria-label="测试已配置的模型连接">{testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Cable className="h-4 w-4" />}{testing ? '正在测试连接' : '测试已配置连接'}</Button><span className="text-xs text-muted-foreground">未配置真实提供商时不会发起网络请求。</span></div>
    {result ? <p className="mt-3 text-sm" role="status">已连接：{result.provider} / {result.model} · Token {result.usage_known ? '已返回' : '未知'}{result.request_id ? ` · 请求 ${result.request_id}` : ''}</p> : null}
    {error ? <p className="mt-3 text-sm text-destructive" role="alert">{error}</p> : null}
  </Panel>
}
