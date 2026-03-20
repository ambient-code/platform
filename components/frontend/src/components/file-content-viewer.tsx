"use client";

import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Download, FileWarning } from "lucide-react";
import { cn } from "@/lib/utils";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

type FileContentViewerProps = {
  fileName: string;
  content: string;
  /** Direct URL to the file for binary content (images, PDFs) */
  fileUrl?: string;
  onDownload?: () => void;
};

function detectFileType(fileName: string, content: string): {
  type: 'image' | 'pdf' | 'html' | 'markdown' | 'binary' | 'text';
  mimeType?: string;
} {
  const ext = fileName.toLowerCase().split('.').pop() || '';

  const imageExts = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico'];
  if (imageExts.includes(ext)) {
    const mimeMap: Record<string, string> = {
      'png': 'image/png', 'jpg': 'image/jpeg', 'jpeg': 'image/jpeg',
      'gif': 'image/gif', 'svg': 'image/svg+xml', 'webp': 'image/webp',
      'bmp': 'image/bmp', 'ico': 'image/x-icon',
    };
    return { type: 'image', mimeType: mimeMap[ext] || 'image/*' };
  }
  if (ext === 'pdf') return { type: 'pdf', mimeType: 'application/pdf' };
  if (ext === 'html' || ext === 'htm') return { type: 'html', mimeType: 'text/html' };
  if (ext === 'md' || ext === 'mdx' || ext === 'markdown') return { type: 'markdown', mimeType: 'text/markdown' };

  const binaryPattern = /[\x00-\x08\x0B\x0C\x0E-\x1F]/;
  if (binaryPattern.test(content.slice(0, 1000))) return { type: 'binary' };

  return { type: 'text' };
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

/**
 * Code block with line numbers
 */
function NumberedCodeBlock({ content, className }: { content: string; className?: string }) {
  const lines = content.split('\n');
  return (
    <div className={cn("flex text-sm font-mono overflow-auto border rounded bg-muted/50", className)}>
      <div
        className="select-none text-right pr-3 pl-3 py-3 text-muted-foreground/50 border-r bg-muted/20 flex-shrink-0"
        aria-hidden
      >
        {lines.map((_, i) => (
          <div key={i} className="leading-6">{i + 1}</div>
        ))}
      </div>
      <pre className="flex-1 py-3 pl-4 pr-4 overflow-x-auto m-0 bg-transparent">
        <code>{content}</code>
      </pre>
    </div>
  );
}

export function FileContentViewer({ fileName, content, fileUrl, onDownload }: FileContentViewerProps) {
  const [imageError, setImageError] = useState(false);

  const fileInfo = useMemo(() => detectFileType(fileName, content), [fileName, content]);
  const fileSize = useMemo(() => new Blob([content]).size, [content]);

  // Image viewer
  if (fileInfo.type === 'image' && !imageError) {
    return (
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">Image • {formatFileSize(fileSize)}</Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <div className="flex-1 bg-muted/50 p-4 rounded border flex items-center justify-center min-h-48">
          {fileUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={fileUrl} alt={fileName} className="max-w-full max-h-full object-contain rounded" onError={() => setImageError(true)} />
          ) : fileInfo.mimeType === 'image/svg+xml' ? (
            <div className="max-w-full overflow-auto" dangerouslySetInnerHTML={{ __html: content }} />
          ) : (
            <div className="text-center text-sm text-muted-foreground">
              <FileWarning className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>Image preview requires a file URL.</p>
              {onDownload && <p className="text-xs mt-1">Download to view.</p>}
            </div>
          )}
        </div>
      </div>
    );
  }

  // PDF viewer
  if (fileInfo.type === 'pdf') {
    return (
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">PDF • {formatFileSize(fileSize)}</Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        {fileUrl ? (
          <div className="flex-1 bg-muted/50 rounded border overflow-hidden min-h-96">
            <object data={fileUrl} type="application/pdf" className="w-full h-full">
              <p className="p-4 text-sm text-muted-foreground text-center">
                PDF cannot be displayed inline.{' '}
                <a href={fileUrl} target="_blank" rel="noopener noreferrer" className="underline">Open PDF</a>
              </p>
            </object>
          </div>
        ) : (
          <div className="bg-muted/50 p-6 rounded border flex flex-col items-center justify-center text-center gap-3">
            <FileWarning className="h-12 w-12 text-muted-foreground opacity-50" />
            <div>
              <p className="text-sm font-medium">PDF File</p>
              <p className="text-xs text-muted-foreground mt-1">Download to view this PDF.</p>
            </div>
          </div>
        )}
      </div>
    );
  }

  // HTML viewer with tabs
  if (fileInfo.type === 'html') {
    return (
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">HTML • {formatFileSize(fileSize)}</Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <Tabs defaultValue="rendered" className="flex flex-col flex-1 min-h-0">
          <TabsList className="w-full justify-start flex-shrink-0">
            <TabsTrigger value="rendered" className="text-xs">Rendered</TabsTrigger>
            <TabsTrigger value="raw" className="text-xs">Raw</TabsTrigger>
          </TabsList>
          <TabsContent value="rendered" className="flex-1 mt-2 min-h-0">
            <div className="bg-muted/50 rounded border overflow-hidden h-full min-h-96">
              <iframe
                {...(fileUrl ? { src: fileUrl } : { srcDoc: content })}
                className="w-full h-full bg-white"
                title={fileName}
                sandbox="allow-scripts allow-same-origin"
              />
            </div>
          </TabsContent>
          <TabsContent value="raw" className="flex-1 mt-2 min-h-0 overflow-auto">
            <NumberedCodeBlock content={content} />
          </TabsContent>
        </Tabs>
      </div>
    );
  }

  // Markdown viewer with tabs
  if (fileInfo.type === 'markdown') {
    return (
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">Markdown • {formatFileSize(fileSize)}</Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <Tabs defaultValue="rendered" className="flex flex-col flex-1 min-h-0">
          <TabsList className="w-full justify-start flex-shrink-0">
            <TabsTrigger value="rendered" className="text-xs">Rendered</TabsTrigger>
            <TabsTrigger value="raw" className="text-xs">Raw</TabsTrigger>
          </TabsList>
          <TabsContent value="rendered" className="flex-1 mt-2 min-h-0 overflow-auto">
            <div className="bg-muted/50 p-4 rounded border prose prose-sm dark:prose-invert max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
                {content}
              </ReactMarkdown>
            </div>
          </TabsContent>
          <TabsContent value="raw" className="flex-1 mt-2 min-h-0 overflow-auto">
            <NumberedCodeBlock content={content} />
          </TabsContent>
        </Tabs>
      </div>
    );
  }

  // Binary file fallback
  if (fileInfo.type === 'binary') {
    const ext = fileName.split('.').pop() || '';
    return (
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">Binary • {ext.toUpperCase()} • {formatFileSize(fileSize)}</Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <div className="bg-muted/50 p-6 rounded border flex flex-col items-center justify-center text-center gap-3">
          <FileWarning className="h-12 w-12 text-muted-foreground opacity-50" />
          <div>
            <p className="text-sm font-medium">Binary File</p>
            <p className="text-xs text-muted-foreground mt-1">Cannot display binary content. Download to view.</p>
          </div>
        </div>
      </div>
    );
  }

  // Text file (default) — with line numbers
  return (
    <div className="flex flex-col h-full gap-2">
      <div className="flex items-center justify-between">
        <Badge variant="secondary" className="text-xs">Text • {formatFileSize(fileSize)}</Badge>
        {onDownload && (
          <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
            <Download className="h-3 w-3" />
          </Button>
        )}
      </div>
      <div className="flex-1 min-h-0 overflow-auto">
        <NumberedCodeBlock content={content} />
      </div>
    </div>
  );
}
