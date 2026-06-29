import React, { Suspense } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { visit } from 'unist-util-visit'

const LazyMermaidBlock = React.lazy(() => import('./markdown/MermaidBlock'))
const LazyCodeBlock = React.lazy(() => import('./markdown/CodeBlock'))

const MermaidSkeleton = () => (
  <div className="mermaid-container flex justify-center bg-white p-6 rounded-xl border border-zinc-200 shadow-sm overflow-x-auto">
    <div className="flex items-center gap-2 text-sm text-zinc-400">
      <svg className="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      加载图表中...
    </div>
  </div>
)

const CodeSkeleton = ({ language }: { language: string }) => (
  <pre className="rounded-xl border border-zinc-200 shadow-sm p-4 text-sm text-zinc-400 bg-zinc-50">
    <code>{language ? `加载 ${language} 代码中...` : '加载代码中...'}</code>
  </pre>
)

const remarkStripMermaidStyles = () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'code', (node: any) => {
      if (node.lang === 'mermaid' && node.value) {
        node.value = node.value
          .split('\n')
          .filter(
            (line: string) =>
              !line.trim().startsWith('style ') &&
              !line.trim().startsWith('classDef ') &&
              !line.trim().startsWith('linkStyle '),
          )
          .join('\n')
      }
    })
  }
}

const rehypeSourceLine = () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'element', (node: any) => {
      if (node.position?.start?.line) {
        node.properties = node.properties || {}
        node.properties['data-source-line'] = node.position.start.line
      }
    })
  }
}

interface MarkdownEngineProps {
  content: string
}

export const MarkdownEngine: React.FC<MarkdownEngineProps> = ({ content }) => {
  const processedContent = content.replace(/(!?\[[^\]]*\])\(([^)]+)\)/g, (_match, p1, p2) => {
    const trimmedUrl = p2.trim()
    if (trimmedUrl.startsWith('http') || trimmedUrl.startsWith('/') || trimmedUrl.startsWith('./')) {
      return `${p1}(${p2.replace(/\s*\n\s*/g, '')})`
    }
    return _match
  })

  return (
    <div className="prose prose-sm md:prose-base max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkStripMermaidStyles]}
        rehypePlugins={[rehypeSourceLine]}
        components={{
          pre(props) {
            // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
            const { node, className, children, ...rest } = props as any
            return (
              <div className={`not-prose my-6 ${className || ''}`} {...rest}>
                {children}
              </div>
            )
          },
          code(props) {
            // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
            const { children, className, ref, node, ...rest } = props as any
            const match = /language-(\w+)/.exec(className || '')
            const isBlock = String(children).includes('\n')

            if (match && match[1] === 'mermaid') {
              const diagramText = String(children).replace(/\n$/, '')
              if (!diagramText || diagramText === 'undefined') {
                return <code className={className} {...rest}>{children}</code>
              }
              return (
                <Suspense fallback={<MermaidSkeleton />}>
                  <LazyMermaidBlock chart={diagramText} />
                </Suspense>
              )
            }

            if (match || isBlock) {
              return (
                <Suspense fallback={<CodeSkeleton language={match ? match[1] : ''} />}>
                  <LazyCodeBlock
                    language={match ? match[1] : 'text'}
                    code={String(children).replace(/\n$/, '')}
                  />
                </Suspense>
              )
            }

            return (
              <code ref={ref} {...rest} className={className}>
                {children}
              </code>
            )
          },
        }}
      >
        {processedContent}
      </ReactMarkdown>
    </div>
  )
}
