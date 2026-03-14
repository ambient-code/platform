"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Shield, ExternalLink, FileCheck } from "lucide-react";
import {
  useInitDriveSetup,
  useDriveCallback,
  useUpdateFileGrants,
} from "@/services/drive-api";
import {
  GooglePicker,
  type SelectedFile,
} from "@/components/google-picker/google-picker";
import { FileSelectionSummary } from "@/components/google-picker/file-selection-summary";

interface GoogleDriveSetupPageProps {
  projectName: string;
  googleApiKey: string;
  googleAppId: string;
}

type SetupStep =
  | "consent"
  | "authenticating"
  | "authenticated"
  | "selecting"
  | "confirming"
  | "complete";

export default function GoogleDriveSetupPage({
  projectName,
  googleApiKey,
  googleAppId,
}: GoogleDriveSetupPageProps) {
  const [step, setStep] = useState<SetupStep>("consent");
  const [selectedFiles, setSelectedFiles] = useState<SelectedFile[]>([]);
  const [error, setError] = useState<string | null>(null);

  const searchParams = useSearchParams();
  const initSetup = useInitDriveSetup();
  const driveCallback = useDriveCallback();
  const updateFileGrants = useUpdateFileGrants();

  // Handle OAuth callback redirect
  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");

    if (code && state) {
      setStep("authenticating");
      driveCallback.mutate(
        { projectName, code, state },
        {
          onSuccess: () => {
            setStep("authenticated");
          },
          onError: (err) => {
            setError(
              err instanceof Error
                ? err.message
                : "OAuth callback failed. Please try again."
            );
            setStep("consent");
          },
        }
      );
    }
  }, [searchParams, projectName]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleConnectDrive = useCallback(() => {
    setError(null);
    const redirectUri = `${window.location.origin}${window.location.pathname}`;

    initSetup.mutate(
      {
        projectName,
        permissionScope: "granular",
        redirectUri,
      },
      {
        onSuccess: (data) => {
          window.location.href = data.authUrl;
        },
        onError: (err) => {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to initiate setup. Please try again."
          );
        },
      }
    );
  }, [projectName, initSetup]);

  const handleFilesPicked = useCallback(
    (files: SelectedFile[]) => {
      if (files.length === 0) {
        setError("Please select at least one file.");
        return;
      }

      setSelectedFiles(files);
      setStep("confirming");
    },
    []
  );

  const handleConfirmSelection = useCallback(() => {
    setError(null);
    updateFileGrants.mutate(
      {
        projectName,
        files: selectedFiles.map((f) => ({
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
          setStep("complete");
        },
        onError: (err) => {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to save file selection. Please try again."
          );
        },
      }
    );
  }, [projectName, selectedFiles, updateFileGrants]);

  const handlePickerCancel = useCallback(() => {
    setStep("authenticated");
  }, []);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Connect Google Drive
          </CardTitle>
          <CardDescription>
            Grant access to only the specific Google Drive files you choose.
            Your other files remain private and inaccessible.
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Permission explanation */}
          <Alert>
            <Shield className="h-4 w-4" />
            <AlertDescription>
              <strong>Only the specific files you select</strong> will be
              accessible to this platform. We use Google&apos;s file picker so
              you choose exactly which files to share — we never see your full
              Drive contents.
            </AlertDescription>
          </Alert>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* Step: Consent — show Connect button */}
          {step === "consent" && (
            <div className="space-y-4">
              <div className="text-sm text-muted-foreground space-y-2">
                <p>
                  When you connect, Google will ask you to authorize access. You
                  will see the text:
                </p>
                <blockquote className="border-l-2 border-muted pl-4 italic">
                  &ldquo;See, edit, create, and delete only the specific Google
                  Drive files you use with this app&rdquo;
                </blockquote>
              </div>

              <Button
                onClick={handleConnectDrive}
                disabled={initSetup.isPending}
                className="w-full"
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                {initSetup.isPending ? "Connecting..." : "Connect Google Drive"}
              </Button>
            </div>
          )}

          {/* Step: Authenticating — show loading */}
          {step === "authenticating" && (
            <div className="text-center py-4 text-muted-foreground">
              Completing authentication...
            </div>
          )}

          {/* Step: Authenticated — show Google Picker */}
          {step === "authenticated" && (
            <div className="space-y-4">
              <Alert>
                <FileCheck className="h-4 w-4" />
                <AlertDescription>
                  Google Drive connected successfully. Now choose the files you
                  want to share with this platform.
                </AlertDescription>
              </Alert>

              <GooglePicker
                projectName={projectName}
                apiKey={googleApiKey}
                appId={googleAppId}
                onFilesPicked={handleFilesPicked}
                onCancel={handlePickerCancel}
                buttonLabel="Choose Files"
              />
            </div>
          )}

          {/* Step: Confirming — show selected files and confirm button */}
          {step === "confirming" && (
            <div className="space-y-4">
              <FileSelectionSummary
                files={selectedFiles}
                title="Files to share"
                description="These files will be accessible to this platform."
              />

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => setStep("authenticated")}
                >
                  Change Selection
                </Button>
                <Button
                  className="flex-1"
                  onClick={handleConfirmSelection}
                  disabled={updateFileGrants.isPending}
                >
                  {updateFileGrants.isPending
                    ? "Saving..."
                    : `Confirm ${selectedFiles.length} file${selectedFiles.length !== 1 ? "s" : ""}`}
                </Button>
              </div>
            </div>
          )}

          {/* Step: Complete — show success */}
          {step === "complete" && (
            <div className="space-y-4">
              <Alert>
                <FileCheck className="h-4 w-4" />
                <AlertDescription>
                  Google Drive integration is active. Your selected files are now
                  accessible to this platform.
                </AlertDescription>
              </Alert>

              <FileSelectionSummary
                files={selectedFiles}
                title="Shared files"
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
