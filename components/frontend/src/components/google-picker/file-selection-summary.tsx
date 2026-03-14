"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  File,
  FileText,
  FileSpreadsheet,
  FileImage,
  Folder,
  FileVideo,
  FileAudio,
} from "lucide-react";

interface FileItem {
  id: string;
  name: string;
  mimeType: string;
  sizeBytes?: number | null;
  isFolder?: boolean;
  status?: string;
}

interface FileSelectionSummaryProps {
  files: FileItem[];
  title?: string;
  description?: string;
}

function formatFileSize(bytes: number | null | undefined): string {
  if (bytes == null || bytes === 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let unitIndex = 0;
  let size = bytes;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(unitIndex > 0 ? 1 : 0)} ${units[unitIndex]}`;
}

function getFileIcon(mimeType: string, isFolder?: boolean) {
  if (isFolder) return <Folder className="h-4 w-4 text-blue-500" />;
  if (mimeType.includes("spreadsheet") || mimeType.includes("excel"))
    return <FileSpreadsheet className="h-4 w-4 text-green-600" />;
  if (mimeType.includes("image"))
    return <FileImage className="h-4 w-4 text-purple-500" />;
  if (mimeType.includes("video"))
    return <FileVideo className="h-4 w-4 text-red-500" />;
  if (mimeType.includes("audio"))
    return <FileAudio className="h-4 w-4 text-orange-500" />;
  if (
    mimeType.includes("document") ||
    mimeType.includes("text") ||
    mimeType.includes("pdf")
  )
    return <FileText className="h-4 w-4 text-blue-600" />;
  return <File className="h-4 w-4 text-gray-500" />;
}

export function FileSelectionSummary({
  files,
  title = "Selected Files",
  description,
}: FileSelectionSummaryProps) {
  if (files.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground">
          No files selected.
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center justify-between">
          {title}
          <Badge variant="secondary">{files.length} file{files.length !== 1 ? "s" : ""}</Badge>
        </CardTitle>
        {description && (
          <CardDescription>{description}</CardDescription>
        )}
      </CardHeader>
      <CardContent>
        <ul className="space-y-2">
          {files.map((file) => (
            <li
              key={file.id}
              className="flex items-center gap-3 py-1.5 px-2 rounded-md hover:bg-muted/50"
            >
              {getFileIcon(file.mimeType, file.isFolder)}
              <span className="flex-1 text-sm truncate">{file.name}</span>
              {file.status === "unavailable" && (
                <Badge variant="destructive" className="text-xs">
                  Unavailable
                </Badge>
              )}
              {file.sizeBytes != null && file.sizeBytes > 0 && (
                <span className="text-xs text-muted-foreground">
                  {formatFileSize(file.sizeBytes)}
                </span>
              )}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
