"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Settings,
  Unplug,
  FileCheck,
  AlertCircle,
  RefreshCw,
} from "lucide-react";
import {
  useDriveIntegration,
  useFileGrants,
  useUpdateFileGrants,
  useDisconnectDriveIntegration,
} from "@/services/drive-api";
import {
  GooglePicker,
  type SelectedFile,
} from "@/components/google-picker/google-picker";
import { FileSelectionSummary } from "@/components/google-picker/file-selection-summary";

interface GoogleDriveSettingsPageProps {
  projectName: string;
  googleApiKey: string;
  googleAppId: string;
}

const statusLabels: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  active: { label: "Active", variant: "default" },
  disconnected: { label: "Disconnected", variant: "destructive" },
  expired: { label: "Expired", variant: "destructive" },
  error: { label: "Error", variant: "destructive" },
};

export default function GoogleDriveSettingsPage({
  projectName,
  googleApiKey,
  googleAppId,
}: GoogleDriveSettingsPageProps) {
  const router = useRouter();
  const [isModifying, setIsModifying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const integrationQuery = useDriveIntegration(projectName);
  const fileGrantsQuery = useFileGrants(projectName);
  const updateFileGrants = useUpdateFileGrants();
  const disconnectIntegration = useDisconnectDriveIntegration();

  const integration = integrationQuery.data;
  const fileGrants = fileGrantsQuery.data?.files ?? [];
  const hasUnavailableFiles = fileGrants.some(
    (f) => f.status === "unavailable"
  );

  const handleModifyFiles = useCallback(
    (files: SelectedFile[]) => {
      if (files.length === 0) {
        setError("Please select at least one file.");
        return;
      }

      setError(null);
      updateFileGrants.mutate(
        {
          projectName,
          files: files.map((f) => ({
            id: f.id,
            name: f.name,
            mimeType: f.mimeType,
            url: f.url,
            sizeBytes: f.sizeBytes,
            isFolder: f.isFolder,
          })),
        },
        {
          onSuccess: () => {
            setIsModifying(false);
          },
          onError: (err) => {
            setError(
              err instanceof Error
                ? err.message
                : "Failed to update file selection."
            );
          },
        }
      );
    },
    [projectName, updateFileGrants]
  );

  const handleDisconnect = useCallback(() => {
    disconnectIntegration.mutate(
      { projectName },
      {
        onSuccess: () => {
          router.push(`/projects/${projectName}/integrations`);
        },
        onError: (err) => {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to disconnect integration."
          );
        },
      }
    );
  }, [projectName, disconnectIntegration, router]);

  if (integrationQuery.isLoading) {
    return (
      <div className="max-w-2xl mx-auto py-8 text-center text-muted-foreground">
        Loading integration settings...
      </div>
    );
  }

  if (!integration) {
    return (
      <div className="max-w-2xl mx-auto py-8">
        <Alert>
          <AlertDescription>
            No Google Drive integration found for this project.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const statusInfo = statusLabels[integration.status] ?? {
    label: integration.status,
    variant: "secondary" as const,
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Integration Status Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Google Drive Integration
            </span>
            <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
          </CardTitle>
          <CardDescription>
            Permission scope:{" "}
            {integration.permissionScope === "granular"
              ? "File-level (only selected files)"
              : "Full Drive access (legacy)"}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {integration.status === "disconnected" && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                This integration has been disconnected. Please re-connect to
                restore access.
              </AlertDescription>
            </Alert>
          )}

          {hasUnavailableFiles && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Some files are no longer available on Google Drive. Consider
                updating your file selection.
              </AlertDescription>
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* File Grants Card */}
      <FileSelectionSummary
        files={fileGrants.map((g) => ({
          id: g.googleFileId,
          name: g.fileName,
          mimeType: g.mimeType,
          sizeBytes: g.sizeBytes,
          isFolder: g.isFolder,
          status: g.status,
        }))}
        title="Shared Files"
        description="These files are accessible to the platform."
      />

      {/* Action Buttons */}
      <div className="flex gap-3">
        {isModifying ? (
          <div className="flex-1 space-y-3">
            <GooglePicker
              projectName={projectName}
              apiKey={googleApiKey}
              appId={googleAppId}
              existingFileIds={fileGrants.map((g) => g.googleFileId)}
              onFilesPicked={handleModifyFiles}
              onCancel={() => setIsModifying(false)}
              buttonLabel="Select Files"
              disabled={updateFileGrants.isPending}
            />
            {updateFileGrants.isPending && (
              <div className="text-center text-sm text-muted-foreground flex items-center justify-center gap-2">
                <RefreshCw className="h-3 w-3 animate-spin" />
                Updating file selection...
              </div>
            )}
          </div>
        ) : (
          <>
            <Button
              variant="outline"
              className="flex-1"
              onClick={() => setIsModifying(true)}
              disabled={integration.status !== "active"}
            >
              <FileCheck className="mr-2 h-4 w-4" />
              Modify Files
            </Button>

            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  variant="destructive"
                  className="flex-1"
                  disabled={disconnectIntegration.isPending}
                >
                  <Unplug className="mr-2 h-4 w-4" />
                  {disconnectIntegration.isPending
                    ? "Disconnecting..."
                    : "Disconnect"}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Disconnect Google Drive?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will revoke all access tokens and remove all file
                    grants. The platform will no longer be able to access any of
                    your Google Drive files. You can reconnect later.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction onClick={handleDisconnect}>
                    Disconnect
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </>
        )}
      </div>
    </div>
  );
}
