"use client";

import React from "react";
import { X, FileIcon, ImageIcon, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

export type PendingAttachment = {
  id: string;
  file: File;
  preview?: string;
  uploading?: boolean;
  error?: string;
};

export type AttachmentPreviewProps = {
  attachments: PendingAttachment[];
  onRemove: (attachmentId: string) => void;
};

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

export const AttachmentPreview: React.FC<AttachmentPreviewProps> = ({
  attachments,
  onRemove,
}) => {
  if (attachments.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 p-2 border-t bg-muted/30">
      {attachments.map((attachment) => {
        const isImage = attachment.file.type.startsWith("image/");

        return (
          <div
            key={attachment.id}
            className="relative group flex items-center gap-2 bg-background border rounded-md p-2 pr-8 max-w-[200px]"
          >
            {isImage && attachment.preview ? (
              <img
                src={attachment.preview}
                alt={attachment.file.name}
                className="w-10 h-10 object-cover rounded"
              />
            ) : isImage ? (
              <ImageIcon className="w-10 h-10 text-muted-foreground p-2" />
            ) : (
              <FileIcon className="w-10 h-10 text-muted-foreground p-2" />
            )}

            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium truncate" title={attachment.file.name}>
                {attachment.file.name}
              </p>
              <p className="text-[10px] text-muted-foreground">
                {formatFileSize(attachment.file.size)}
              </p>
              {attachment.error && (
                <p className="text-[10px] text-destructive truncate">{attachment.error}</p>
              )}
            </div>

            {attachment.uploading ? (
              <div className="absolute top-1 right-1">
                <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
              </div>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onRemove(attachment.id)}
                className="absolute top-0 right-0 h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-destructive/10 hover:text-destructive"
              >
                <X className="h-3 w-3" />
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default AttachmentPreview;
