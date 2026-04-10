"use client";

import { useState, useRef } from "react";
import { Loader2, Link, FileUp, FolderUp } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { InputWithHistory } from "@/components/input-with-history";
import { useInputHistory } from "@/hooks/use-input-history";

// Maximum file size: 10MB for all file types
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB unified limit
// Maximum total folder size: 100MB
const MAX_FOLDER_SIZE = 100 * 1024 * 1024;

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
};

export type UploadFileSource = {
  type: "local" | "url" | "folder";
  file?: File;
  files?: { file: File; relativePath: string }[];
  url?: string;
  filename?: string;
};

type UploadFileModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFile: (source: UploadFileSource) => Promise<void>;
  isLoading?: boolean;
};

export function UploadFileModal({
  open,
  onOpenChange,
  onUploadFile,
  isLoading = false,
}: UploadFileModalProps) {
  const [activeTab, setActiveTab] = useState<"local" | "folder" | "url">("local");
  const [fileUrl, setFileUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [selectedFolderFiles, setSelectedFolderFiles] = useState<{ file: File; relativePath: string }[]>([]);
  const [folderName, setFolderName] = useState<string | null>(null);
  const [isStartingService, setIsStartingService] = useState(false);
  const [fileSizeError, setFileSizeError] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const { addToHistory: addUrlToHistory } = useInputHistory("upload-file:url");

  const handleSubmit = async () => {
    setIsStartingService(false);

    if (activeTab === "local") {
      if (!selectedFile) return;
      try {
        await onUploadFile({ type: "local", file: selectedFile });
      } catch (error) {
        if (error instanceof Error && error.message.includes("starting")) {
          setIsStartingService(true);
        }
        throw error;
      }
    } else if (activeTab === "folder") {
      if (selectedFolderFiles.length === 0) return;
      try {
        await onUploadFile({ type: "folder", files: selectedFolderFiles });
      } catch (error) {
        if (error instanceof Error && error.message.includes("starting")) {
          setIsStartingService(true);
        }
        throw error;
      }
    } else {
      if (!fileUrl.trim()) return;

      // Save URL to history before uploading
      addUrlToHistory(fileUrl.trim());

      // Extract filename from URL
      const urlParts = fileUrl.split("/");
      const filename = urlParts[urlParts.length - 1] || "downloaded-file";

      try {
        await onUploadFile({ type: "url", url: fileUrl.trim(), filename });
      } catch (error) {
        if (error instanceof Error && error.message.includes("starting")) {
          setIsStartingService(true);
        }
        throw error;
      }
    }

    // Reset form on success
    setFileUrl("");
    setSelectedFile(null);
    setSelectedFolderFiles([]);
    setFolderName(null);
    setIsStartingService(false);
    setFileSizeError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    if (folderInputRef.current) {
      folderInputRef.current.value = "";
    }
  };

  const handleCancel = () => {
    setFileUrl("");
    setSelectedFile(null);
    setSelectedFolderFiles([]);
    setFolderName(null);
    setIsStartingService(false);
    setFileSizeError(null);
    setIsValidating(false);
    setActiveTab("local");
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    if (folderInputRef.current) {
      folderInputRef.current.value = "";
    }
    onOpenChange(false);
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Show loading state while validating
    setIsValidating(true);
    setFileSizeError(null);
    setSelectedFile(null);

    // Use setTimeout to allow UI to update with loading state
    setTimeout(() => {
      // Check file size against unified 10MB limit
      if (file.size > MAX_FILE_SIZE) {
        setFileSizeError(
          `File size (${formatFileSize(file.size)}) exceeds maximum allowed size of ${formatFileSize(MAX_FILE_SIZE)}`
        );
        setSelectedFile(null);
        if (fileInputRef.current) {
          fileInputRef.current.value = "";
        }
      } else {
        setFileSizeError(null);
        setSelectedFile(file);
      }
      setIsValidating(false);
    }, 0);
  };

  const handleFolderSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    setIsValidating(true);
    setFileSizeError(null);
    setSelectedFolderFiles([]);
    setFolderName(null);

    setTimeout(() => {
      const folderFiles: { file: File; relativePath: string }[] = [];
      let totalSize = 0;

      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const relativePath = (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;

        if (file.size > MAX_FILE_SIZE) {
          setFileSizeError(
            `File "${relativePath}" (${formatFileSize(file.size)}) exceeds the per-file limit of ${formatFileSize(MAX_FILE_SIZE)}`
          );
          setSelectedFolderFiles([]);
          setFolderName(null);
          if (folderInputRef.current) {
            folderInputRef.current.value = "";
          }
          setIsValidating(false);
          return;
        }

        totalSize += file.size;
        folderFiles.push({ file, relativePath });
      }

      if (totalSize > MAX_FOLDER_SIZE) {
        setFileSizeError(
          `Total folder size (${formatFileSize(totalSize)}) exceeds the maximum allowed size of ${formatFileSize(MAX_FOLDER_SIZE)}`
        );
        setSelectedFolderFiles([]);
        setFolderName(null);
        if (folderInputRef.current) {
          folderInputRef.current.value = "";
        }
      } else {
        setFileSizeError(null);
        setSelectedFolderFiles(folderFiles);
        const firstPath = folderFiles[0]?.relativePath || "";
        const topFolder = firstPath.split("/")[0] || "folder";
        setFolderName(topFolder);
      }
      setIsValidating(false);
    }, 0);
  };

  const isSubmitDisabled = () => {
    if (isLoading || isValidating) return true;
    if (activeTab === "local") return !selectedFile;
    if (activeTab === "folder") return selectedFolderFiles.length === 0;
    if (activeTab === "url") return !fileUrl.trim();
    return true;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Upload File</DialogTitle>
          <DialogDescription>
            Upload files to your workspace from your local machine or a URL. Files will be available in
            the file-uploads folder. Maximum file size: {formatFileSize(MAX_FILE_SIZE)}.
          </DialogDescription>
        </DialogHeader>

        {fileSizeError && (
          <Alert variant="destructive">
            <AlertDescription>{fileSizeError}</AlertDescription>
          </Alert>
        )}

        {isValidating && (
          <Alert>
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertDescription>
              Validating file...
            </AlertDescription>
          </Alert>
        )}

        {isStartingService && (
          <Alert>
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertDescription>
              Content service is starting. This may take a few seconds. Your upload will automatically retry.
            </AlertDescription>
          </Alert>
        )}

        <Tabs
          value={activeTab}
          onValueChange={(v) => {
            setActiveTab(v as "local" | "folder" | "url");
            setFileSizeError(null);
          }}
          className="w-full"
        >
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="local" disabled={isLoading || isValidating}>
              <FileUp className="h-4 w-4 mr-2" />
              File
            </TabsTrigger>
            <TabsTrigger value="folder" disabled={isLoading || isValidating}>
              <FolderUp className="h-4 w-4 mr-2" />
              Folder
            </TabsTrigger>
            <TabsTrigger value="url" disabled={isLoading || isValidating}>
              <Link className="h-4 w-4 mr-2" />
              URL
            </TabsTrigger>
          </TabsList>

          <TabsContent value="local" className="space-y-4">
            <div className="space-y-2">
              <input
                id="file-upload"
                ref={fileInputRef}
                type="file"
                onChange={handleFileSelect}
                disabled={isLoading || isValidating}
                className="sr-only"
                aria-label="Choose File"
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isLoading || isValidating}
                className="w-full border-2 border-dashed border-muted-foreground/25 hover:border-primary/50 rounded-lg p-6 flex flex-col items-center gap-2 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <FileUp className="h-8 w-8 text-muted-foreground/60" />
                <span className="text-sm font-medium">Click to choose a file</span>
                <span className="text-xs text-muted-foreground">
                  Max {formatFileSize(MAX_FILE_SIZE)}
                </span>
              </button>
              {selectedFile && !isValidating && (
                <p className="text-sm text-muted-foreground">
                  Selected: {selectedFile.name} ({(selectedFile.size / 1024).toFixed(1)} KB)
                </p>
              )}
            </div>
          </TabsContent>

          <TabsContent value="folder" className="space-y-4">
            <div className="space-y-2">
              <input
                id="folder-upload"
                ref={folderInputRef}
                type="file"
                // @ts-expect-error webkitdirectory is not in React's InputHTMLAttributes
                webkitdirectory=""
                directory=""
                onChange={handleFolderSelect}
                disabled={isLoading || isValidating}
                className="sr-only"
                aria-label="Choose Folder"
              />
              <button
                type="button"
                onClick={() => folderInputRef.current?.click()}
                disabled={isLoading || isValidating}
                className="w-full border-2 border-dashed border-muted-foreground/25 hover:border-primary/50 rounded-lg p-6 flex flex-col items-center gap-2 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <FolderUp className="h-8 w-8 text-muted-foreground/60" />
                <span className="text-sm font-medium">Click to choose a folder</span>
                <span className="text-xs text-muted-foreground">
                  Max {formatFileSize(MAX_FILE_SIZE)} per file, {formatFileSize(MAX_FOLDER_SIZE)} total
                </span>
              </button>
              {selectedFolderFiles.length > 0 && !isValidating && (
                <div className="text-sm text-muted-foreground space-y-1">
                  <p>
                    Selected: {folderName}/ — {selectedFolderFiles.length} file(s),{" "}
                    {formatFileSize(selectedFolderFiles.reduce((sum, f) => sum + f.file.size, 0))} total
                  </p>
                </div>
              )}
              <p className="text-xs text-muted-foreground">
                All files in the folder will be uploaded preserving directory structure.
              </p>
            </div>
          </TabsContent>

          <TabsContent value="url" className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="file-url">File URL</Label>
              <InputWithHistory
                historyKey="upload-file:url"
                id="file-url"
                type="url"
                placeholder="https://example.com/file.pdf"
                value={fileUrl}
                onChange={(e) => setFileUrl(e.target.value)}
                disabled={isLoading || isValidating}
              />
              <p className="text-sm text-muted-foreground">
                The file will be downloaded and uploaded to your workspace
              </p>
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel} disabled={isLoading || isValidating}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitDisabled()}>
            {isLoading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Uploading...
              </>
            ) : isValidating ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Validating...
              </>
            ) : (
              "Upload"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
