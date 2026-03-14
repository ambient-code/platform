"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { FileCheck, RefreshCw, AlertCircle } from "lucide-react";
import { loadPickerApi } from "@/lib/google-picker-loader";
import { usePickerToken } from "@/services/drive-api";

export interface SelectedFile {
  id: string;
  name: string;
  mimeType: string;
  url: string;
  sizeBytes: number | null;
  isFolder: boolean;
}

interface GooglePickerProps {
  projectName: string;
  /** Google API key for the Picker. */
  apiKey: string;
  /** Google Cloud project app ID. */
  appId: string;
  /** Pre-selected file IDs (for the modify flow). */
  existingFileIds?: string[];
  /** Called when user selects files and confirms. */
  onFilesPicked: (files: SelectedFile[]) => void;
  /** Called when user cancels the picker. */
  onCancel?: () => void;
  /** Custom button label. */
  buttonLabel?: string;
  /** Whether the button is disabled. */
  disabled?: boolean;
}

export function GooglePicker({
  projectName,
  apiKey,
  appId,
  onFilesPicked,
  onCancel,
  buttonLabel = "Choose Files",
  disabled = false,
}: GooglePickerProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const pickerTokenQuery = usePickerToken(projectName);

  const openPicker = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch a fresh picker token
      const tokenData = await pickerTokenQuery.refetch();
      if (!tokenData.data?.accessToken) {
        throw new Error("Failed to get picker token. Please re-authenticate.");
      }

      // Load the Picker API
      await loadPickerApi();

      const accessToken = tokenData.data.accessToken;

      // Build the Picker
      const docsView = new window.google.picker.DocsView(
        window.google.picker.ViewId.DOCS
      );
      docsView.setIncludeFolders(true);
      docsView.setSelectFolderEnabled(true);

      const picker = new window.google.picker.PickerBuilder()
        .setOAuthToken(accessToken)
        .setDeveloperKey(apiKey)
        .setAppId(appId)
        .addView(docsView)
        .enableFeature(window.google.picker.Feature.MULTISELECT_ENABLED)
        .enableFeature(window.google.picker.Feature.SUPPORT_DRIVES)
        .setOrigin(window.location.protocol + "//" + window.location.host)
        .setTitle("Select files to share with this platform")
        .setCallback((data: GooglePickerResponse) => {
          if (data.action === window.google.picker.Action.PICKED) {
            if (data.docs.length === 0) {
              setError("Please select at least one file.");
              return;
            }

            const selectedFiles: SelectedFile[] = data.docs.map((doc) => ({
              id: doc.id,
              name: doc.name,
              mimeType: doc.mimeType,
              url: doc.url,
              sizeBytes: doc.sizeBytes || null,
              isFolder: doc.type === "folder",
            }));

            onFilesPicked(selectedFiles);
          } else if (data.action === window.google.picker.Action.CANCEL) {
            onCancel?.();
          }
        })
        .build();

      picker.setVisible(true);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to open file picker.";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [apiKey, appId, onFilesPicked, onCancel, pickerTokenQuery]);

  return (
    <div className="space-y-2">
      <Button
        onClick={openPicker}
        disabled={disabled || loading}
        className="w-full"
      >
        {loading ? (
          <>
            <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
            Opening file picker...
          </>
        ) : (
          <>
            <FileCheck className="mr-2 h-4 w-4" />
            {buttonLabel}
          </>
        )}
      </Button>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="flex items-center justify-between">
            <span>{error}</span>
            <Button
              variant="outline"
              size="sm"
              onClick={openPicker}
              disabled={loading}
            >
              <RefreshCw className="mr-1 h-3 w-3" />
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
