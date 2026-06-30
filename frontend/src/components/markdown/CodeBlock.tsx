import React from 'react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'

interface CodeBlockProps {
  language: string
  code: string
}

const CodeBlock: React.FC<CodeBlockProps> = ({ language, code }) => (
  <SyntaxHighlighter
    PreTag="div"
    children={code}
    language={language}
    style={oneLight}
    className="rounded-xl border border-zinc-200 shadow-sm text-sm !my-0"
  />
)

export default CodeBlock
