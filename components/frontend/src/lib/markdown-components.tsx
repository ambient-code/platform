import React from "react";
import type { Components } from "react-markdown";

/**
 * Shared ReactMarkdown component overrides used by both message.tsx and tool-message.tsx.
 * All colors use Tailwind theme utilities for theme-awareness -- no hardcoded hex/rgba.
 *
 * Spec: specs/frontend/sessions/messages/markdown-rendering.spec.md
 */
export const sharedMarkdownComponents: Components = {
  // --- Inline code vs block code ---
  code: ({
    inline,
    className,
    children,
    ...props
  }: {
    inline?: boolean;
    className?: string;
    children?: React.ReactNode;
  } & React.HTMLAttributes<HTMLElement>) => {
    const codeContent = String(children || "");
    const isShortCode = codeContent.length <= 50 && !codeContent.includes("\n");

    if (inline || isShortCode) {
      return (
        <code
          className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono"
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      );
    }

    return (
      <pre className="bg-muted text-foreground py-3 rounded text-xs overflow-x-auto border my-2">
        <code
          className={className}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      </pre>
    );
  },

  // --- Paragraph spacing: mb-2 (8px) ---
  p: ({ children }) => (
    <div className="text-muted-foreground leading-relaxed mb-2 text-sm">
      {children}
    </div>
  ),

  // --- Headings ---
  h1: ({ children }) => (
    <h1 className="text-lg font-bold text-foreground mb-2">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-md font-semibold text-foreground mb-2">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-sm font-medium text-foreground mb-1">{children}</h3>
  ),
  // Extended headings h4-h6: progressively smaller scale
  h4: ({ children }) => (
    <h4 className="text-xs font-medium text-foreground mb-1">{children}</h4>
  ),
  h5: ({ children }) => (
    <h5 className="text-xs font-normal text-foreground mb-1">{children}</h5>
  ),
  h6: ({ children }) => (
    <h6 className="text-xs font-light text-foreground mb-1">{children}</h6>
  ),

  // --- Inline formatting ---
  strong: ({ children }) => (
    <strong className="font-semibold text-foreground">{children}</strong>
  ),
  em: ({ children }) => (
    <em className="italic text-foreground">{children}</em>
  ),
  del: ({ children }) => (
    <del className="line-through opacity-60">{children}</del>
  ),

  // --- Lists: harmonized spacing with paragraphs ---
  ul: ({ children }) => (
    <ul className="list-disc list-outside ml-4 mb-2 space-y-1.5 text-muted-foreground text-sm">
      {children}
    </ul>
  ),
  ol: ({ children }) => (
    <ol className="list-decimal list-outside ml-4 mb-2 space-y-1.5 text-muted-foreground text-sm">
      {children}
    </ol>
  ),
  li: ({ children }) => <li className="leading-relaxed">{children}</li>,

  // --- Links ---
  a: ({ href, children }) => (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-primary hover:underline cursor-pointer"
    >
      {children}
    </a>
  ),

  // --- GFM Tables ---
  table: ({ children }) => (
    <div className="overflow-x-auto my-2">
      <table className="border-collapse w-full text-sm">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead>{children}</thead>,
  tbody: ({ children }) => <tbody>{children}</tbody>,
  tr: ({ children }) => <tr className="border-b border-border">{children}</tr>,
  th: ({ children }) => (
    <th className="px-3 py-1.5 text-left font-medium text-foreground bg-muted border border-border">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="px-3 py-1.5 text-muted-foreground border border-border">
      {children}
    </td>
  ),

  // --- Blockquote ---
  blockquote: ({ children }) => (
    <blockquote className="border-l-4 border-border pl-4 py-1 my-2 italic text-muted-foreground">
      {children}
    </blockquote>
  ),

  // --- Horizontal rule ---
  hr: () => <hr className="border-t border-border my-4" />,

  // --- Image ---
  img: ({ src, alt, ...props }) => (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={src}
      alt={alt || ""}
      className="max-w-full rounded"
      {...(props as React.ImgHTMLAttributes<HTMLImageElement>)}
    />
  ),
};
