import React, { useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import mermaid from 'mermaid';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { visit } from 'unist-util-visit';

// Initialize mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'loose',
});

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

const MermaidBlock: React.FC<{ chart: string }> = ({ chart }) => {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (containerRef.current && chart) {
      const renderChart = async () => {
        try {
          // Add a unique ID for each chart to avoid conflicts
          const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`;
          const { svg } = await mermaid.render(id, chart);
          if (containerRef.current) {
            containerRef.current.innerHTML = svg;
          }
        } catch (error) {
          console.error('Failed to render mermaid chart', error);
          if (containerRef.current) {
            containerRef.current.innerHTML = `<div class="text-red-500 text-sm p-4 border border-red-200 rounded">Failed to render diagram</div>`;
          }
        }
      };
      renderChart();
    }
  }, [chart]);

  return <div className="mermaid-container flex justify-center my-6" ref={containerRef} />;
};

interface MarkdownEngineProps {
  content: string;
}

export const MarkdownEngine: React.FC<MarkdownEngineProps> = ({ content }) => {
  return (
    <div className="prose prose-sm md:prose-base max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkStripMermaidStyles]}
        components={{
          code(props) {
            const { children, className, ref, ...rest } = props;
            const match = /language-(\w+)/.exec(className || '');
            
            // Render Mermaid blocks using our custom component
            if (match && match[1] === 'mermaid') {
              return <MermaidBlock chart={String(children).replace(/\n$/, '')} />;
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
              <code ref={ref} {...rest} className={className}>
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
