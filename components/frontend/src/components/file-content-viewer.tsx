"use client";

import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Download, ExternalLink, FileWarning } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

type FileContentViewerProps = {
  fileName: string;
  content: string;
  onDownload?: () => void;
};

/**
 * Detect file type based on extension and content
 */
function detectFileType(fileName: string, content: string): {
  type: 'image' | 'pdf' | 'html' | 'markdown' | 'binary' | 'text';
  mimeType?: string;
} {
  const ext = fileName.toLowerCase().split('.').pop() || '';

  // Image files
  const imageExts = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico'];
  if (imageExts.includes(ext)) {
    const mimeMap: Record<string, string> = {
      'png': 'image/png',
      'jpg': 'image/jpeg',
      'jpeg': 'image/jpeg',
      'gif': 'image/gif',
      'svg': 'image/svg+xml',
      'webp': 'image/webp',
      'bmp': 'image/bmp',
      'ico': 'image/x-icon',
    };
    return { type: 'image', mimeType: mimeMap[ext] || 'image/*' };
  }

  // PDF files
  if (ext === 'pdf') {
    return { type: 'pdf', mimeType: 'application/pdf' };
  }

  // HTML files
  if (ext === 'html' || ext === 'htm') {
    return { type: 'html', mimeType: 'text/html' };
  }

  // Markdown files
  if (ext === 'md' || ext === 'mdx' || ext === 'markdown') {
    return { type: 'markdown', mimeType: 'text/markdown' };
  }

  // Check for binary content (non-printable characters)
  const binaryPattern = /[\x00-\x08\x0B\x0C\x0E-\x1F]/;
  if (binaryPattern.test(content.slice(0, 1000))) {
    return { type: 'binary' };
  }

  return { type: 'text' };
}

/**
 * Format file size in human-readable format
 */
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

/**
 * Open content in a new browser tab
 */
function openInNewTab(fileName: string, content: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const newWindow = window.open(url, '_blank', 'noopener,noreferrer');

  // Clean up the URL after the window loads or after a delay
  setTimeout(() => URL.revokeObjectURL(url), newWindow ? 1000 : 5000);
}

export function FileContentViewer({ fileName, content, onDownload }: FileContentViewerProps) {
  const [imageError, setImageError] = useState(false);

  const fileInfo = useMemo(() => detectFileType(fileName, content), [fileName, content]);
  const fileSize = useMemo(() => new Blob([content]).size, [content]);

  const handleOpenInTab = () => {
    if (fileInfo.mimeType) {
      openInNewTab(fileName, content, fileInfo.mimeType);
    }
  };

  // Image viewer
  if (fileInfo.type === 'image' && !imageError) {
    // Convert content to data URL for display
    const dataUrl = `data:${fileInfo.mimeType};base64,${btoa(content)}`;

    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            Image • {formatFileSize(fileSize)}
          </Badge>
          <div className="flex gap-1">
            {onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onDownload}
                className="h-7 px-2"
                title="Download file"
              >
                <Download className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
        <div className="bg-muted/50 p-4 rounded border flex items-center justify-center">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={dataUrl}
            alt={fileName}
            className="max-w-full max-h-96 object-contain rounded"
            onError={() => setImageError(true)}
          />
        </div>
      </div>
    );
  }

  // PDF viewer
  if (fileInfo.type === 'pdf') {
    const dataUrl = `data:application/pdf;base64,${btoa(content)}`;

    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            PDF • {formatFileSize(fileSize)}
          </Badge>
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleOpenInTab}
              className="h-7 px-2"
              title="Open in new tab"
            >
              <ExternalLink className="h-3 w-3 mr-1" />
              <span className="text-xs">Open</span>
            </Button>
            {onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onDownload}
                className="h-7 px-2"
                title="Download file"
              >
                <Download className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
        <div className="bg-muted/50 rounded border overflow-hidden">
          <iframe
            src={dataUrl}
            className="w-full h-96"
            title={fileName}
          />
        </div>
      </div>
    );
  }

  // HTML viewer
  if (fileInfo.type === 'html') {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            HTML • {formatFileSize(fileSize)}
          </Badge>
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleOpenInTab}
              className="h-7 px-2"
              title="Render in new tab"
            >
              <ExternalLink className="h-3 w-3 mr-1" />
              <span className="text-xs">Render</span>
            </Button>
            {onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onDownload}
                className="h-7 px-2"
                title="Download file"
              >
                <Download className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
        <div className="text-xs">
          <pre className="bg-muted/50 p-3 rounded overflow-x-auto max-h-96 overflow-y-auto border">
            <code>{content}</code>
          </pre>
        </div>
      </div>
    );
  }

  // Markdown viewer
  if (fileInfo.type === 'markdown') {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            Markdown • {formatFileSize(fileSize)}
          </Badge>
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleOpenInTab}
              className="h-7 px-2"
              title="Open rendered view in new tab"
            >
              <ExternalLink className="h-3 w-3 mr-1" />
              <span className="text-xs">Visualize</span>
            </Button>
            {onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onDownload}
                className="h-7 px-2"
                title="Download file"
              >
                <Download className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
        <div className="bg-muted/50 p-4 rounded border overflow-auto max-h-96 prose prose-sm dark:prose-invert max-w-none">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeHighlight]}
          >
            {content}
          </ReactMarkdown>
        </div>
      </div>
    );
  }

  // Binary file fallback
  if (fileInfo.type === 'binary') {
    const ext = fileName.split('.').pop() || '';

    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            Binary • {ext.toUpperCase()} • {formatFileSize(fileSize)}
          </Badge>
          <div className="flex gap-1">
            {onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onDownload}
                className="h-7 px-2"
                title="Download file"
              >
                <Download className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
        <div className="bg-muted/50 p-6 rounded border flex flex-col items-center justify-center text-center gap-3">
          <FileWarning className="h-12 w-12 text-muted-foreground opacity-50" />
          <div>
            <p className="text-sm font-medium">Binary File</p>
            <p className="text-xs text-muted-foreground mt-1">
              Cannot display binary content. Download to view.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Text file (default)
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Badge variant="secondary" className="text-xs">
          Text • {formatFileSize(fileSize)}
        </Badge>
        <div className="flex gap-1">
          {onDownload && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onDownload}
              className="h-7 px-2"
              title="Download file"
            >
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
      </div>
      <div className="text-xs">
        <pre className="bg-muted/50 p-3 rounded overflow-x-auto max-h-96 overflow-y-auto border">
          <code>{content}</code>
        </pre>
      </div>
    </div>
  );
}
