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
  suppressErrorRendering: true, // Prevent "Syntax error in text" from rendering on the page
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

// Rehype plugin to add source line numbers to elements
const rehypeSourceLine = () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'element', (node: any) => {
      if (node.position?.start?.line) {
        node.properties = node.properties || {};
        node.properties['data-source-line'] = node.position.start.line;
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
          // Validate diagram type to prevent rendering errors
          const trimmedChart = chart.trim();
          if (!trimmedChart || trimmedChart === 'undefined') {
            throw new Error('Empty or undefined diagram text');
          }
          
          // Use suppressErrors to prevent mermaid from throwing errors directly to console/UI
          mermaid.mermaidAPI.getConfig().suppressErrorRendering = true;
          
          const { svg } = await mermaid.render(id, trimmedChart);
          if (containerRef.current) {
            containerRef.current.innerHTML = svg;
          }
        } catch {
          // Do not log the error to the console during streaming as it's expected
          // that incomplete mermaid syntax will throw errors until fully generated.
          // Hide errors completely to avoid cluttering the UI with "Syntax error in text"
          if (containerRef.current) {
            containerRef.current.innerHTML = '';
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
        rehypePlugins={[rehypeSourceLine]}
        components={{
          code(props) {
            const { children, className, ref, ...rest } = props;
            const match = /language-(\w+)/.exec(className || '');
            
            // Render Mermaid blocks using our custom component
            if (match && match[1] === 'mermaid') {
              const diagramText = String(children).replace(/\n$/, '');
              if (!diagramText || diagramText === 'undefined') {
                return <code className={className} {...rest}>{children}</code>;
              }
              return <MermaidBlock chart={diagramText} />;
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
