"use client";

import { useState, useRef } from "react";
import { Loader2, Link, FileUp } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription } from "@/components/ui/alert";

// Maximum file sizes based on type
// Documents (text files): 700KB limit - no base64 encoding overhead
// Images: 3MB upload limit - realistic compression to 350KB target
const MAX_DOCUMENT_SIZE = 700 * 1024; // 700KB for documents
const MAX_IMAGE_SIZE = 3 * 1024 * 1024; // 3MB for images (server will compress to 350KB)

// Determine if a file is an image based on MIME type
const isImageFile = (fileType: string): boolean => {
  return fileType.startsWith('image/');
};

// Get the appropriate max file size based on file type
const getMaxFileSize = (fileType: string): number => {
  return isImageFile(fileType) ? MAX_IMAGE_SIZE : MAX_DOCUMENT_SIZE;
};

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
};

type UploadFileModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFile: (source: {
    type: "local" | "url";
    file?: File;
    url?: string;
    filename?: string;
  }) => Promise<void>;
  isLoading?: boolean;
};

export function UploadFileModal({
  open,
  onOpenChange,
  onUploadFile,
  isLoading = false,
}: UploadFileModalProps) {
  const [activeTab, setActiveTab] = useState<"local" | "url">("local");
  const [fileUrl, setFileUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isStartingService, setIsStartingService] = useState(false);
  const [fileSizeError, setFileSizeError] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = async () => {
    setIsStartingService(false);

    if (activeTab === "local") {
      if (!selectedFile) return;
      try {
        await onUploadFile({ type: "local", file: selectedFile });
      } catch (error) {
        // Check if error is about content service starting
        if (error instanceof Error && error.message.includes("starting")) {
          setIsStartingService(true);
        }
        throw error;
      }
    } else {
      if (!fileUrl.trim()) return;

      // Extract filename from URL
      const urlParts = fileUrl.split("/");
      const filename = urlParts[urlParts.length - 1] || "downloaded-file";

      try {
        await onUploadFile({ type: "url", url: fileUrl.trim(), filename });
      } catch (error) {
        // Check if error is about content service starting
        if (error instanceof Error && error.message.includes("starting")) {
          setIsStartingService(true);
        }
        throw error;
      }
    }

    // Reset form on success
    setFileUrl("");
    setSelectedFile(null);
    setIsStartingService(false);
    setFileSizeError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleCancel = () => {
    setFileUrl("");
    setSelectedFile(null);
    setIsStartingService(false);
    setFileSizeError(null);
    setIsValidating(false);
    setActiveTab("local");
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
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
      const fileType = file.type || 'application/octet-stream';
      const maxSize = getMaxFileSize(fileType);
      const fileTypeLabel = isImageFile(fileType) ? 'images' : 'documents';

      // Check file size based on type
      if (file.size > maxSize) {
        setFileSizeError(
          `File size (${formatFileSize(file.size)}) exceeds maximum allowed size of ${formatFileSize(maxSize)} for ${fileTypeLabel}`
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

  const isSubmitDisabled = () => {
    if (isLoading || isValidating) return true;
    if (activeTab === "local") return !selectedFile;
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
            the file-uploads folder. Maximum file size: {formatFileSize(MAX_IMAGE_SIZE)} for images, {formatFileSize(MAX_DOCUMENT_SIZE)} for documents.
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
            setActiveTab(v as "local" | "url");
            setFileSizeError(null); // Clear error when switching tabs
          }}
          className="w-full"
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="local" disabled={isLoading || isValidating}>
              <FileUp className="h-4 w-4 mr-2" />
              Local File
            </TabsTrigger>
            <TabsTrigger value="url" disabled={isLoading || isValidating}>
              <Link className="h-4 w-4 mr-2" />
              From URL
            </TabsTrigger>
          </TabsList>

          <TabsContent value="local" className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="file-upload">Choose File</Label>
              <Input
                id="file-upload"
                ref={fileInputRef}
                type="file"
                onChange={handleFileSelect}
                disabled={isLoading || isValidating}
              />
              {selectedFile && !isValidating && (
                <p className="text-sm text-muted-foreground">
                  Selected: {selectedFile.name} ({(selectedFile.size / 1024).toFixed(1)} KB)
                </p>
              )}
            </div>
          </TabsContent>

          <TabsContent value="url" className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="file-url">File URL</Label>
              <Input
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
