"use client";

import React from "react";
import { cn } from "@/lib/utils";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";
import { formatTimestamp } from "@/lib/format-timestamp";
import { useLoadingTips } from "@/services/queries/use-loading-tips";
import { DEFAULT_LOADING_TIPS } from "@/lib/loading-tips";

export type MessageRole = "bot" | "user";

export type MessageProps = {
  role: MessageRole;
  content: string;
  isLoading?: boolean;
  avatar?: string;
  name?: string;
  className?: string;
  components?: Components;
  borderless?: boolean;
  actions?: React.ReactNode;
  timestamp?: string;
  streaming?: boolean;
  /** Feedback buttons to show below the message (for bot messages) */
  feedbackButtons?: React.ReactNode;
};

const defaultComponents: Components = {
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
    // Convert children to string to check length
    const codeContent = String(children || '');
    const isShortCode = codeContent.length <= 50 && !codeContent.includes('\n');

    // Treat short code blocks as inline
    if (inline || isShortCode) {
      return (
        <code
          className="bg-code-bg text-code-foreground px-1.5 py-0.5 rounded text-[0.9em] font-mono"
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      );
    }

    // Full code blocks for longer content
    return (
      <pre className="bg-code-bg text-code-foreground py-3 px-3 rounded-lg text-[0.9em] overflow-x-auto border border-border/50 my-2">
        <code
          className={className}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      </pre>
    );
  },
  p: ({ children }) => (
    <div className="text-foreground/80 leading-relaxed mb-1 text-base">{children}</div>
  ),
  h1: ({ children }) => (
    <h1 className="text-xl font-bold text-foreground mb-2">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-lg font-semibold text-foreground mb-2">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-base font-medium text-foreground mb-1">{children}</h3>
  ),
  ul: ({ children }) => (
    <ul className="list-disc list-outside ml-4 mb-2 space-y-1 text-foreground/80 text-base">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="list-decimal list-outside ml-4 mb-2 space-y-1 text-foreground/80 text-base">{children}</ol>
  ),
  li: ({ children }) => (
    <li className="leading-relaxed">{children}</li>
  ),
  a: ({ href, children }) => (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-link hover:text-link-hover hover:underline cursor-pointer"
    >
      {children}
    </a>
  ),
};

/**
 * Parse markdown-style links [text](url) in a string and return React elements
 */
function parseMarkdownLinks(text: string): React.ReactNode {
  const linkRegex = /\[([^\]]+)\]\(([^)]+)\)/g;
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match;

  while ((match = linkRegex.exec(text)) !== null) {
    // Add text before the link
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    // Validate URL scheme to prevent javascript: injection
    const href = match[2];
    const isSafeUrl = href.startsWith('https://') || href.startsWith('http://');
    // Add the link (or plain text if URL is unsafe)
    parts.push(
      isSafeUrl ? (
        <a
          key={match.index}
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-link hover:text-link-hover hover:underline"
        >
          {match[1]}
        </a>
      ) : (
        <span key={match.index}>{match[1]}</span>
      )
    );
    lastIndex = match.index + match[0].length;
  }

  // Add remaining text after last link
  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts.length > 0 ? parts : text;
}

export const LoadingDots = () => {
  const { data } = useLoadingTips();
  const tips = data?.tips ?? DEFAULT_LOADING_TIPS;

  const [messageIndex, setMessageIndex] = React.useState(() =>
    Math.floor(Math.random() * tips.length)
  );

  // Reset index when tips array changes to prevent undefined access
  React.useEffect(() => {
    setMessageIndex((prev) => prev % tips.length);
  }, [tips.length]);

  React.useEffect(() => {
    const intervalId = setInterval(() => {
      setMessageIndex((prevIndex) => (prevIndex + 1) % tips.length);
    }, 8000);
    return () => clearInterval(intervalId);
  }, [tips.length]);

  return (
    <div className="flex items-center mt-2">
      <svg
        width="56"
        height="16"
        viewBox="0 0 56 16"
        xmlns="http://www.w3.org/2000/svg"
        className="mr-2"
      >
        <style>
          {`
            @keyframes loadingDotPulse {
              0%, 60%, 100% {
                opacity: 0.3;
              }
              30% {
                opacity: 1;
              }
            }
            .loading-dot {
              animation: loadingDotPulse 1.4s infinite ease-in-out;
            }
            .loading-dot-1 {
              animation-delay: 0s;
            }
            .loading-dot-2 {
              animation-delay: 0.15s;
            }
            .loading-dot-3 {
              animation-delay: 0.3s;
            }
            .loading-dot-4 {
              animation-delay: 0.45s;
            }
          `}
        </style>
        <circle
          className="loading-dot loading-dot-1"
          cx="8"
          cy="8"
          r="5"
          fill="var(--primary)"
        />
        <circle
          className="loading-dot loading-dot-2"
          cx="22"
          cy="8"
          r="5"
          fill="var(--chart-2)"
        />
        <circle
          className="loading-dot loading-dot-3"
          cx="36"
          cy="8"
          r="5"
          fill="var(--destructive)"
        />
        <circle
          className="loading-dot loading-dot-4"
          cx="50"
          cy="8"
          r="5"
          fill="var(--muted-foreground)"
          opacity="0.4"
        />
      </svg>
      <span className="ml-2 text-sm text-muted-foreground">{parseMarkdownLinks(tips[messageIndex])}</span>
    </div>
  );
};

export const Message = React.forwardRef<HTMLDivElement, MessageProps>(
  (
    { role, content, isLoading, className, components, borderless, actions, timestamp, streaming, feedbackButtons, ...props },
    ref
  ) => {
    const isBot = role === "bot";
    const avatarBg = isBot ? "bg-primary" : "bg-chart-2";
    const avatarText = isBot ? "AI" : "U";
    const formattedTime = formatTimestamp(timestamp);
    const isActivelyStreaming = streaming && isBot;

    const avatar = (
      <div className="flex-shrink-0">
      <div
        className={cn(
          "w-8 h-8 rounded-full flex items-center justify-center ring-2 ring-background",
          avatarBg,
          (isLoading || isActivelyStreaming) && "animate-pulse"
        )}
      >
        <span className="text-white text-sm font-semibold">
          {avatarText}
        </span>
      </div>
    </div>
    )

    return (
      <div ref={ref} className={cn("mb-4", isBot && "mt-2", className)} {...props}>
        <div className={cn("flex space-x-3", isBot ? "items-start" : "items-center justify-end")}>
          {/* Avatar */}
         {isBot ? avatar : null}

          {/* Message Content */}
          <div className={cn("flex-1 min-w-0", !isBot && "max-w-[70%]")}>
            {/* Timestamp */}
            {formattedTime && (
              <div className={cn("text-sm text-muted-foreground/60 mb-1", !isBot && "text-right")}>
                {formattedTime}
              </div>
            )}
            <div className={cn(
              borderless ? "p-0" : "rounded-lg",
              !borderless && (isBot ? "bg-card" : "bg-primary/8 dark:bg-primary/12")
            )}>
              {/* Content */}
              <div className={cn("text-base text-foreground", !isBot && "py-2 px-4")}>
                {isLoading ? (
                  <div>
                    <div className="text-base text-muted-foreground mb-2">{content}</div>
                    <LoadingDots />
                  </div>
                ) : (
                  <div className="inline">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      components={components || defaultComponents}
                    >
                      {content}
                    </ReactMarkdown>
                    {isActivelyStreaming && (
                      <span className="inline-block w-0.5 h-4 bg-primary animate-[cursor-blink_1s_ease-in-out_infinite] rounded-full ml-0.5 align-middle" />
                    )}
                  </div>
                )}
              </div>

              {/* Feedback buttons for bot messages */}
              {isBot && feedbackButtons && !isLoading && !streaming && (
                <div className="mt-2 flex items-center">
                  {feedbackButtons}
                </div>
              )}

              {actions ? (
                <div className={cn(borderless ? "mt-1" : "mt-3 pt-2 border-t")}>{actions}</div>
              ) : null}
            </div>
          </div>

          {isBot ? null : avatar}
        </div>
      </div>
    );
  }
);

Message.displayName = "Message";
