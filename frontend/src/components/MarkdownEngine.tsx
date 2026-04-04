import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeMermaid from 'rehype-mermaid';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { visit } from 'unist-util-visit';

// Remark plugin to strip style and classDef from mermaid blocks
const remarkStripMermaidStyles = () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'code', (node: any) => {
      if (node.lang === 'mermaid' && node.value) {
        // Remove style and classDef lines to ensure a clean default theme
        node.value = node.value
          .split('\n')
          .filter((line: string) => !line.trim().startsWith('style ') && !line.trim().startsWith('classDef ') && !line.trim().startsWith('linkStyle '))
          .join('\n');
      }
    });
  };
};

interface MarkdownEngineProps {
  content: string;
}

export const MarkdownEngine: React.FC<MarkdownEngineProps> = ({ content }) => {
  return (
    <div className="prose prose-sm md:prose-base max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkStripMermaidStyles]}
        rehypePlugins={[
          [
            rehypeMermaid,
            {
              // Config rehype-mermaid to intercept and remove styles for a pure theme
              // This ensures the frontend renderer uses default styles without pollution
              strategy: 'inline-svg',
              mermaidOptions: {
                theme: 'default',
                themeVariables: {
                  // You can override specific variables here if needed
                }
              }
            }
          ]
        ]}
        components={{
          code(props) {
            const { children, className, ...rest } = props;
            const match = /language-(\w+)/.exec(className || '');
            
            // Skip mermaid blocks as they are handled by rehype-mermaid
            if (match && match[1] === 'mermaid') {
              return <code className={className} {...rest}>{children}</code>;
            }

            return match ? (
              <SyntaxHighlighter
                {...rest}
                PreTag="div"
                children={String(children).replace(/\n$/, '')}
                language={match[1]}
                style={oneLight}
              />
            ) : (
              <code {...rest} className={className}>
                {children}
              </code>
            );
          }
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};
