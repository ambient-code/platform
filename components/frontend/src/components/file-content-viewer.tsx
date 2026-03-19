"use client";

import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Download, FileWarning } from "lucide-react";
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

/**
 * Detect file type based on extension and content
 */
function detectFileType(fileName: string, content: string): {
  type: 'image' | 'pdf' | 'html' | 'markdown' | 'binary' | 'text';
  mimeType?: string;
} {
  const ext = fileName.toLowerCase().split('.').pop() || '';

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

  if (ext === 'pdf') {
    return { type: 'pdf', mimeType: 'application/pdf' };
  }

  if (ext === 'html' || ext === 'htm') {
    return { type: 'html', mimeType: 'text/html' };
  }

  if (ext === 'md' || ext === 'mdx' || ext === 'markdown') {
    return { type: 'markdown', mimeType: 'text/markdown' };
  }

  const binaryPattern = /[\x00-\x08\x0B\x0C\x0E-\x1F]/;
  if (binaryPattern.test(content.slice(0, 1000))) {
    return { type: 'binary' };
  }

  return { type: 'text' };
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

export function FileContentViewer({ fileName, content, fileUrl, onDownload }: FileContentViewerProps) {
  const [imageError, setImageError] = useState(false);

  const fileInfo = useMemo(() => detectFileType(fileName, content), [fileName, content]);
  const fileSize = useMemo(() => new Blob([content]).size, [content]);

  // Image viewer
  if (fileInfo.type === 'image' && !imageError) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            Image • {formatFileSize(fileSize)}
          </Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <div className="bg-muted/50 p-4 rounded border flex items-center justify-center">
          {fileUrl ? (
            // Use direct URL for binary images
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={fileUrl}
              alt={fileName}
              className="max-w-full max-h-96 object-contain rounded"
              onError={() => setImageError(true)}
            />
          ) : (
            // Fallback: SVG and text-based images can use inline content
            fileInfo.mimeType === 'image/svg+xml' ? (
              <div
                className="max-w-full max-h-96 overflow-auto"
                dangerouslySetInnerHTML={{ __html: content }}
              />
            ) : (
              <div className="text-center text-sm text-muted-foreground">
                <FileWarning className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>Image preview requires a file URL.</p>
                {onDownload && <p className="text-xs mt-1">Download to view.</p>}
              </div>
            )
          )}
        </div>
      </div>
    );
  }

  // PDF viewer
  if (fileInfo.type === 'pdf') {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            PDF • {formatFileSize(fileSize)}
          </Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        {fileUrl ? (
          <div className="bg-muted/50 rounded border overflow-hidden">
            <iframe src={fileUrl} className="w-full h-96" title={fileName} />
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
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            HTML • {formatFileSize(fileSize)}
          </Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <Tabs defaultValue="raw" className="w-full">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="raw" className="text-xs">Raw</TabsTrigger>
            <TabsTrigger value="rendered" className="text-xs">Rendered</TabsTrigger>
          </TabsList>
          <TabsContent value="raw" className="mt-2">
            <div className="text-xs">
              <pre className="bg-muted/50 p-3 rounded overflow-x-auto max-h-96 overflow-y-auto border">
                <code>{content}</code>
              </pre>
            </div>
          </TabsContent>
          <TabsContent value="rendered" className="mt-2">
            <div className="bg-muted/50 rounded border overflow-hidden">
              <iframe
                srcDoc={content}
                className="w-full h-96 bg-white"
                title={fileName}
                sandbox="allow-scripts"
              />
            </div>
          </TabsContent>
        </Tabs>
      </div>
    );
  }

  // Markdown viewer with tabs
  if (fileInfo.type === 'markdown') {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Badge variant="secondary" className="text-xs">
            Markdown • {formatFileSize(fileSize)}
          </Badge>
          {onDownload && (
            <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
              <Download className="h-3 w-3" />
            </Button>
          )}
        </div>
        <Tabs defaultValue="rendered" className="w-full">
          <TabsList className="w-full justify-start">
            <TabsTrigger value="rendered" className="text-xs">Rendered</TabsTrigger>
            <TabsTrigger value="raw" className="text-xs">Raw</TabsTrigger>
          </TabsList>
          <TabsContent value="rendered" className="mt-2">
            <div className="bg-muted/50 p-4 rounded border overflow-auto max-h-96 prose prose-sm dark:prose-invert max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
                {content}
              </ReactMarkdown>
            </div>
          </TabsContent>
          <TabsContent value="raw" className="mt-2">
            <div className="text-xs">
              <pre className="bg-muted/50 p-3 rounded overflow-x-auto max-h-96 overflow-y-auto border">
                <code>{content}</code>
              </pre>
            </div>
          </TabsContent>
        </Tabs>
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

  // Text file (default)
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Badge variant="secondary" className="text-xs">
          Text • {formatFileSize(fileSize)}
        </Badge>
        {onDownload && (
          <Button variant="ghost" size="sm" onClick={onDownload} className="h-7 px-2" title="Download file">
            <Download className="h-3 w-3" />
          </Button>
        )}
      </div>
      <div className="text-xs">
        <pre className="bg-muted/50 p-3 rounded overflow-x-auto max-h-96 overflow-y-auto border">
          <code>{content}</code>
        </pre>
      </div>
    </div>
  );
}
