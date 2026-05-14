"use client";

import React, { useState, useCallback } from "react";
import type { Components } from "react-markdown";
import { Check, Code2, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

/**
 * CopyButton -- appears on hover/focus inside code blocks.
 * Uses navigator.clipboard with a copied-state timeout.
 */
function CopyButton({ text, inline }: { text: string; inline?: boolean }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API may be unavailable in some contexts; fail silently.
    }
  }, [text]);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={handleCopy}
            aria-label={copied ? "Copied" : "Copy code to clipboard"}
            className={
              inline
                ? "cursor-pointer bg-transparent hover:bg-muted border-0 text-muted-foreground hover:text-foreground opacity-0 group-hover/codeblock:opacity-100 focus:opacity-100 transition-opacity"
                : "absolute top-2 right-2 cursor-pointer bg-muted/80 hover:bg-muted border border-border text-muted-foreground hover:text-foreground opacity-0 group-hover/codeblock:opacity-100 focus:opacity-100 transition-opacity"
            }
          >
            {copied ? (
              <Check className="w-3.5 h-3.5 text-green-500" />
            ) : (
              <Copy className="w-3.5 h-3.5" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">
          {copied ? "Copied!" : "Copy"}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

/**
 * Extract plain text from React children for the copy button.
 */
function extractText(node: React.ReactNode): string {
  if (typeof node === "string") return node;
  if (typeof node === "number") return String(node);
  if (node == null || typeof node === "boolean") return "";
  if (Array.isArray(node)) return node.map(extractText).join("");
  if (React.isValidElement(node)) {
    const props = node.props as { children?: React.ReactNode };
    return extractText(props.children);
  }
  return "";
}

/**
 * Extract the language name from a <code> element's className.
 * rehype-highlight adds classes like "language-python" or "hljs language-typescript".
 * Returns the capitalised language name, or null when no language tag is present.
 */
function extractLanguage(children: React.ReactNode): string | null {
  const child = React.Children.toArray(children)[0];
  if (!React.isValidElement(child)) return null;

  const className = (child.props as { className?: string }).className;
  if (!className) return null;

  const match = className.match(/language-(\S+)/);
  if (!match) return null;

  const lang = match[1];
  // Capitalise first letter for display
  return lang.charAt(0).toUpperCase() + lang.slice(1);
}

/**
 * Shared ReactMarkdown component overrides used by both message.tsx and tool-message.tsx.
 * All colors use Tailwind theme utilities for theme-awareness -- no hardcoded hex/rgba.
 *
 * Syntax highlighting is provided by rehype-highlight (highlight.js) which must be
 * passed as a rehypePlugin on each ReactMarkdown instance. Theme adaptation between
 * light/dark mode is handled by SyntaxThemeProvider + syntax-highlighting.css.
 *
 * Spec: specs/frontend/sessions/messages/markdown-rendering.spec.md
 */
export const sharedMarkdownComponents: Components = {
  // --- Block code container (fenced code blocks) ---
  // rehype-highlight adds hljs classes to <code> inside <pre>.
  // We wrap with a group for hover-visible copy button.
  pre: ({ children, ...props }: React.HTMLAttributes<HTMLPreElement>) => {
    const text = extractText(children);
    const language = extractLanguage(children);
    return (
      <div className="relative group/codeblock my-2">
        {language && (
          <div className="flex items-center gap-1.5 bg-muted/80 text-muted-foreground text-xs px-3 py-1.5 rounded-t border border-b-0 font-mono">
            <Code2 className="w-3.5 h-3.5" />
            <span className="flex-1">{language}</span>
            <CopyButton text={text} inline />
          </div>
        )}
        <pre
          className={`bg-muted text-foreground text-xs overflow-x-auto border ${language ? "rounded-b" : "rounded"}`}
          {...props}
        >
          {children}
        </pre>
        {!language && <CopyButton text={text} />}
      </div>
    );
  },

  // --- Inline code ---
  // Fenced code blocks are handled by the `pre` override above;
  // the `code` override only styles inline code snippets.
  code: ({
    className,
    children,
    ...props
  }: {
    className?: string;
    children?: React.ReactNode;
  } & React.HTMLAttributes<HTMLElement>) => {
    // When inside a <pre> (fenced block), className contains "hljs" or "language-*".
    // Let rehype-highlight's classes pass through untouched.
    const isHighlighted =
      className && (className.includes("hljs") || className.includes("language-"));

    if (isHighlighted) {
      return (
        <code
          className={className}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      );
    }

    // Inline code styling
    return (
      <code
        className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono"
        {...(props as React.HTMLAttributes<HTMLElement>)}
      >
        {children}
      </code>
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
