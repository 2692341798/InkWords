import React, { useEffect, useRef, useState } from 'react'
import mermaid from 'mermaid'

mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'strict',
  suppressErrorRendering: true,
  darkMode: false,
  themeVariables: {
    background: '#ffffff',
  },
})

const sanitizeChart = (chart: string): string => {
  let trimmed = chart.trim()
  trimmed = trimmed.replace(/\[([^"\]]+)\]/g, (_match, p1) => {
    if (p1.includes('(') || p1.includes(')') || p1.includes('^')) {
      return `["${p1}"]`
    }
    return _match
  })
  return trimmed
}

const ErrorFallback: React.FC<{ chart: string }> = ({ chart }) => (
  <div className="w-full">
    <div className="text-xs text-red-500 mb-2 font-semibold flex items-center gap-1">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      Mermaid 渲染失败: 语法错误
    </div>
    <pre className="text-xs text-zinc-600 overflow-auto p-4 bg-zinc-50 rounded-lg border border-zinc-200">
      <code>{chart}</code>
    </pre>
  </div>
)

const MermaidBlock: React.FC<{ chart: string }> = ({ chart }) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svgContent, setSvgContent] = useState<string | null>(null)
  const [renderError, setRenderError] = useState<boolean>(false)

  useEffect(() => {
    let cancelled = false

    const renderChart = async () => {
      try {
        const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`
        const trimmedChart = sanitizeChart(chart)

        if (!trimmedChart || trimmedChart === 'undefined') {
          throw new Error('Empty or undefined diagram text')
        }

        const { svg } = await mermaid.render(id, trimmedChart)
        if (!cancelled) {
          setSvgContent(svg)
          setRenderError(false)
        }
      } catch (err: unknown) {
        if (!cancelled) {
          console.debug('Mermaid rendering failed:', err)
          setSvgContent(null)
          setRenderError(true)
        }
      }
    }

    renderChart()

    return () => {
      cancelled = true
    }
  }, [chart])

  return (
    <div
      className="mermaid-container flex justify-center bg-white p-6 rounded-xl border border-zinc-200 shadow-sm overflow-x-auto"
      ref={containerRef}
    >
      {svgContent ? (
        <div dangerouslySetInnerHTML={{ __html: svgContent }} />
      ) : renderError ? (
        <ErrorFallback chart={chart} />
      ) : null}
    </div>
  )
}

export default MermaidBlock
