"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Save, Loader2, Info, AlertTriangle } from "lucide-react";
import { Plus, Trash2, Eye, EyeOff, ChevronDown, ChevronRight } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useProject, useUpdateProject } from "@/services/queries/use-projects";
import { useSecretsValues, useUpdateSecrets, useIntegrationSecrets, useUpdateIntegrationSecrets } from "@/services/queries/use-secrets";
import { useClusterInfo } from "@/hooks/use-cluster-info";
import { useMemo } from "react";

type SettingsSectionProps = {
  projectName: string;
};

export function SettingsSection({ projectName }: SettingsSectionProps) {
  const [formData, setFormData] = useState({ displayName: "", description: "" });
  const [secrets, setSecrets] = useState<Array<{ key: string; value: string }>>([]);
  const [showValues, setShowValues] = useState<Record<number, boolean>>({});
  const [anthropicApiKey, setAnthropicApiKey] = useState<string>("");
  const [showAnthropicKey, setShowAnthropicKey] = useState<boolean>(false);
  const [storageMode, setStorageMode] = useState<"shared" | "custom">("shared");
  const [s3Endpoint, setS3Endpoint] = useState<string>("");
  const [s3Bucket, setS3Bucket] = useState<string>("");
  const [s3Region, setS3Region] = useState<string>("us-east-1");
  const [s3AccessKey, setS3AccessKey] = useState<string>("");
  const [s3SecretKey, setS3SecretKey] = useState<string>("");
  const [showS3SecretKey, setShowS3SecretKey] = useState<boolean>(false);
  const [anthropicExpanded, setAnthropicExpanded] = useState<boolean>(false);
  const [s3Expanded, setS3Expanded] = useState<boolean>(false);
  const FIXED_KEYS = useMemo(() => ["ANTHROPIC_API_KEY","STORAGE_MODE","S3_ENDPOINT","S3_BUCKET","S3_REGION","S3_ACCESS_KEY","S3_SECRET_KEY"] as const, []);

  // React Query hooks
  const { data: project, isLoading: projectLoading } = useProject(projectName);
  const { data: runnerSecrets } = useSecretsValues(projectName);  // ambient-runner-secrets (ANTHROPIC_API_KEY)
  const { data: integrationSecrets } = useIntegrationSecrets(projectName);  // ambient-non-vertex-integrations (GITHUB_TOKEN, GIT_USER_*, JIRA_*, custom)
  const { vertexEnabled } = useClusterInfo();
  const updateProjectMutation = useUpdateProject();
  const updateSecretsMutation = useUpdateSecrets();
  const updateIntegrationSecretsMutation = useUpdateIntegrationSecrets();

  // Sync project data to form
  useEffect(() => {
    if (project) {
      setFormData({ displayName: project.displayName || "", description: project.description || "" });
    }
  }, [project]);

  // Sync secrets values to state (merge both secrets)
  useEffect(() => {
    const allSecrets = [...(runnerSecrets || []), ...(integrationSecrets || [])];
    if (allSecrets.length > 0) {
      const byKey: Record<string, string> = Object.fromEntries(allSecrets.map(s => [s.key, s.value]));
      setAnthropicApiKey(byKey["ANTHROPIC_API_KEY"] || "");
      // Determine storage mode: "custom" if S3_ENDPOINT is set, otherwise "shared" (default)
      const hasCustomS3 = byKey["STORAGE_MODE"] === "custom" || (byKey["S3_ENDPOINT"] && byKey["S3_ENDPOINT"] !== "");
      setStorageMode(hasCustomS3 ? "custom" : "shared");
      setS3Endpoint(byKey["S3_ENDPOINT"] || "");
      setS3Bucket(byKey["S3_BUCKET"] || "");
      setS3Region(byKey["S3_REGION"] || "us-east-1");
      setS3AccessKey(byKey["S3_ACCESS_KEY"] || "");
      setS3SecretKey(byKey["S3_SECRET_KEY"] || "");
      setSecrets(allSecrets.filter(s => !FIXED_KEYS.includes(s.key as typeof FIXED_KEYS[number])));
    }
  }, [runnerSecrets, integrationSecrets, FIXED_KEYS]);

  const handleSave = () => {
    if (!project) return;
    updateProjectMutation.mutate(
      {
        name: projectName,
        data: {
          displayName: formData.displayName.trim(),
          description: formData.description.trim() || undefined,
          annotations: project.annotations || {},
        },
      },
      {
        onSuccess: () => {
          successToast("Project settings updated successfully!");
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "Failed to update project";
          errorToast(message);
        },
      }
    );
  };

  // Save Anthropic API key separately (ambient-runner-secrets)
  const handleSaveAnthropicKey = () => {
    if (!projectName) return;

    const runnerData: Record<string, string> = {};
    if (anthropicApiKey) runnerData["ANTHROPIC_API_KEY"] = anthropicApiKey;

    if (Object.keys(runnerData).length === 0) {
      errorToast("No Anthropic API key to save");
      return;
    }

    updateSecretsMutation.mutate(
      {
        projectName,
        secrets: Object.entries(runnerData).map(([key, value]) => ({ key, value })),
      },
      {
        onSuccess: () => {
          successToast("Saved to ambient-runner-secrets");
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "Failed to save Anthropic API key";
          errorToast(message);
        },
      }
    );
  };

  // Save integration secrets separately (ambient-non-vertex-integrations)
  const handleSaveIntegrationSecrets = () => {
    if (!projectName) return;

    const integrationData: Record<string, string> = {};

    // NOTE: GIT_USER_* removed - git identity now auto-derived from GitHub/GitLab credentials
    
    // S3 Storage configuration
    integrationData["STORAGE_MODE"] = storageMode;
    if (storageMode === "custom") {
      // Only save custom S3 settings when custom mode is selected
      if (s3Endpoint) integrationData["S3_ENDPOINT"] = s3Endpoint;
      if (s3Bucket) integrationData["S3_BUCKET"] = s3Bucket;
      if (s3Region) integrationData["S3_REGION"] = s3Region;
      if (s3AccessKey) integrationData["S3_ACCESS_KEY"] = s3AccessKey;
      if (s3SecretKey) integrationData["S3_SECRET_KEY"] = s3SecretKey;
    }
    // If shared mode: backend will use operator defaults + minio-credentials secret
    for (const { key, value } of secrets) {
      if (!key) continue;
      if (FIXED_KEYS.includes(key as typeof FIXED_KEYS[number])) continue;
      integrationData[key] = value ?? "";
    }

    if (Object.keys(integrationData).length === 0) {
      errorToast("No integration secrets to save");
      return;
    }

    updateIntegrationSecretsMutation.mutate(
      {
        projectName,
        secrets: Object.entries(integrationData).map(([key, value]) => ({ key, value })),
      },
      {
        onSuccess: () => {
          successToast("Saved to ambient-non-vertex-integrations");
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "Failed to save integration secrets";
          errorToast(message);
        },
      }
    );
  };

  const addSecretRow = () => {
    setSecrets((prev) => [...prev, { key: "", value: "" }]);
  };

  const removeSecretRow = (idx: number) => {
    setSecrets((prev) => prev.filter((_, i) => i !== idx));
  };

  return (
    <div className="flex-1 space-y-6">
      {/* Only show project metadata editor on OpenShift */}
      {project?.isOpenShift ? (
        <Card>
          <CardHeader>
            <CardTitle>General Settings</CardTitle>
            <CardDescription>Basic workspace configuration</CardDescription>
          </CardHeader>
          <Separator />
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="displayName">Display Name</Label>
              <Input
                id="displayName"
                value={formData.displayName}
                onChange={(e) => setFormData((prev) => ({ ...prev, displayName: e.target.value }))}
                placeholder="My Awesome Workspace"
                maxLength={100}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="workspaceName">Workspace Name</Label>
              <Input
                id="workspaceName"
                value={projectName}
                readOnly
                disabled
                className="bg-muted/80 text-muted-foreground"
              />
              <p className="text-sm text-muted-foreground">Workspace name cannot be changed after creation</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="Describe the purpose and goals of this workspace..."
                maxLength={500}
                rows={3}
              />
            </div>
            <div className="pt-2">
              <Button onClick={handleSave} disabled={updateProjectMutation.isPending || projectLoading || !project}>
                {updateProjectMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    Save Changes
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            Running on vanilla Kubernetes. Project display name and description editing is not available.
            The project namespace is: <strong>{projectName}</strong>
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Integration Secrets</CardTitle>
          <CardDescription>
            Configure environment variables for workspace runners. All values are injected into runner pods.
          </CardDescription>
        </CardHeader>
        <Separator />
        <CardContent className="space-y-6">
          {/* Warning about centralized integrations */}
          <Alert variant="warning">
            <AlertTriangle />
            <AlertTitle>Centralized Integrations Recommended</AlertTitle>
            <AlertDescription>
              <p>Cluster-level integrations (Vertex AI, GitHub App, Jira OAuth) are more secure than personal tokens. Only configure these secrets if centralized integrations are unavailable.</p>
            </AlertDescription>
          </Alert>

          {/* Anthropic Section */}
          <div className="border rounded-lg">
            <button
              type="button"
              onClick={() => setAnthropicExpanded(!anthropicExpanded)}
              className="w-full flex items-center justify-between p-3 hover:bg-muted/50 transition-colors rounded-lg"
            >
              <div className="flex items-center gap-2">
                {anthropicExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                <span className="font-semibold">Anthropic</span>
                {anthropicApiKey && <span className="text-xs text-muted-foreground">(configured)</span>}
              </div>
            </button>
            {anthropicExpanded && (
              <div className="px-3 pb-3 space-y-3 border-t pt-3">
                {vertexEnabled && anthropicApiKey && (
                  <Alert variant="warning">
                    <AlertTriangle />
                    <AlertDescription>
                      Vertex AI is enabled for this cluster. The ANTHROPIC_API_KEY will be ignored. Sessions will use Vertex AI instead.
                    </AlertDescription>
                  </Alert>
                )}
                <div className="space-y-2">
                  <Label htmlFor="anthropicApiKey">ANTHROPIC_API_KEY</Label>
                  <div className="text-xs text-muted-foreground">Your Anthropic API key for Claude Code runner (saved to ambient-runner-secrets)</div>
                  <div className="flex items-center gap-2">
                    <Input
                      id="anthropicApiKey"
                      type={showAnthropicKey ? "text" : "password"}
                      placeholder="sk-ant-..."
                      value={anthropicApiKey}
                      onChange={(e) => setAnthropicApiKey(e.target.value)}
                      className="flex-1"
                    />
                    <Button type="button" variant="ghost" size="sm" onClick={() => setShowAnthropicKey((v) => !v)} aria-label={showAnthropicKey ? "Hide key" : "Show key"}>
                      {showAnthropicKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  </div>
                </div>
                <div className="pt-2">
                  <Button onClick={handleSaveAnthropicKey} disabled={updateSecretsMutation.isPending} size="sm">
                    {updateSecretsMutation.isPending ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Saving...
                      </>
                    ) : (
                      <>
                        <Save className="w-4 h-4 mr-2" />
                        Save Anthropic Key
                      </>
                    )}
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* Migration Notice */}
          <div className="border rounded-lg p-4 bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800">
            <h3 className="text-sm font-semibold mb-2 text-blue-900 dark:text-blue-100">Integration Credentials Moved</h3>
            <p className="text-xs text-blue-800 dark:text-blue-200 mb-2">
              GitHub, GitLab, Jira, and Google Drive credentials are now managed at the user level on the{' '}
              <Link href="/integrations" className="underline font-medium">Integrations page</Link>.
              This allows you to use the same credentials across all your workspaces.
            </p>
            <p className="text-xs text-blue-700 dark:text-blue-300">
              Any credentials previously configured here will continue to work as a fallback, but we recommend
              connecting your integrations on the Integrations page for the best experience.
            </p>
          </div>

          {/* S3 Storage Configuration Section */}
          <div className="space-y-3 pt-4 border-t">
            <div
              className="flex items-center justify-between cursor-pointer hover:opacity-80"
              onClick={() => setS3Expanded((v) => !v)}
            >
              <div>
                <Label className="text-base font-semibold cursor-pointer">S3 Storage Configuration</Label>
                <div className="text-xs text-muted-foreground mt-1">Configure S3-compatible storage for session artifacts and state</div>
              </div>
              {s3Expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
            </div>
            {s3Expanded && (
              <div className="space-y-4 pl-1">
                <Alert>
                  <Info className="h-4 w-4" />
                  <AlertTitle>Session State Storage</AlertTitle>
                  <AlertDescription>
                    Session artifacts, uploads, and Claude history are persisted to S3-compatible storage. By default, the cluster provides shared MinIO storage.
                  </AlertDescription>
                </Alert>
                <div className="space-y-3">
                  <Label className="text-sm font-medium">Storage Configuration</Label>
                  <div className="space-y-2">
                    <div className="flex items-center space-x-2">
                      <input
                        id="storage-shared"
                        type="radio"
                        name="storageMode"
                        value="shared"
                        checked={storageMode === "shared"}
                        onChange={() => setStorageMode("shared")}
                        className="h-4 w-4"
                      />
                      <Label htmlFor="storage-shared" className="cursor-pointer font-normal">
                        Use shared cluster storage (default)
                      </Label>
                    </div>
                    <div className="text-xs text-muted-foreground ml-6">
                      Automatically uses in-cluster MinIO. No configuration needed.
                    </div>
                  </div>
                  <div className="space-y-2">
                    <div className="flex items-center space-x-2">
                      <input
                        id="storage-custom"
                        type="radio"
                        name="storageMode"
                        value="custom"
                        checked={storageMode === "custom"}
                        onChange={() => setStorageMode("custom")}
                        className="h-4 w-4"
                      />
                      <Label htmlFor="storage-custom" className="cursor-pointer font-normal">
                        Use custom S3-compatible storage
                      </Label>
                    </div>
                    <div className="text-xs text-muted-foreground ml-6">
                      Configure AWS S3, external MinIO, or other S3-compatible endpoint.
                    </div>
                  </div>
                </div>
                {storageMode === "custom" && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="s3Endpoint">S3_ENDPOINT</Label>
                      <div className="text-xs text-muted-foreground mb-1">S3-compatible endpoint (e.g., https://s3.amazonaws.com, http://minio.local:9000)</div>
                      <Input
                        id="s3Endpoint"
                        type="text"
                        placeholder="https://s3.amazonaws.com"
                        value={s3Endpoint}
                        onChange={(e) => setS3Endpoint(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="s3Bucket">S3_BUCKET</Label>
                      <div className="text-xs text-muted-foreground mb-1">Bucket name for session storage</div>
                      <Input
                        id="s3Bucket"
                        type="text"
                        placeholder="ambient-sessions"
                        value={s3Bucket}
                        onChange={(e) => setS3Bucket(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="s3Region">S3_REGION</Label>
                      <div className="text-xs text-muted-foreground mb-1">AWS region (optional, default: us-east-1)</div>
                      <Input
                        id="s3Region"
                        type="text"
                        placeholder="us-east-1"
                        value={s3Region}
                        onChange={(e) => setS3Region(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="s3AccessKey">S3_ACCESS_KEY</Label>
                      <div className="text-xs text-muted-foreground mb-1">S3 access key ID</div>
                      <Input
                        id="s3AccessKey"
                        type="text"
                        placeholder="AKIAIOSFODNN7EXAMPLE"
                        value={s3AccessKey}
                        onChange={(e) => setS3AccessKey(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="s3SecretKey">S3_SECRET_KEY</Label>
                      <div className="text-xs text-muted-foreground mb-1">S3 secret access key</div>
                      <div className="flex items-center gap-2">
                        <Input
                          id="s3SecretKey"
                          type={showS3SecretKey ? "text" : "password"}
                          placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
                          value={s3SecretKey}
                          onChange={(e) => setS3SecretKey(e.target.value)}
                          className="flex-1"
                        />
                        <Button type="button" variant="ghost" size="sm" onClick={() => setShowS3SecretKey((v) => !v)} aria-label={showS3SecretKey ? "Hide secret" : "Show secret"}>
                          {showS3SecretKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </Button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>

          {/* Custom Environment Variables Section */}
          <div className="space-y-3 pt-2">
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-base font-semibold">Custom Environment Variables</Label>
                <div className="text-xs text-muted-foreground mt-1">Add any additional environment variables for your integrations</div>
              </div>
            </div>
            <div className="space-y-2">
              {secrets.map((item, idx) => (
                <div key={idx} className="flex gap-2 items-center">
                  <Input
                    value={item.key}
                    onChange={(e) =>
                      setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, key: e.target.value } : it)))
                    }
                    placeholder="KEY"
                    className="w-1/3"
                  />
                  <div className="flex-1 flex items-center gap-2">
                    <Input
                      type={showValues[idx] ? "text" : "password"}
                      value={item.value}
                      onChange={(e) =>
                        setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, value: e.target.value } : it)))
                      }
                      placeholder="value"
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowValues((prev) => ({ ...prev, [idx]: !prev[idx] }))}
                      aria-label={showValues[idx] ? "Hide value" : "Show value"}
                    >
                      {showValues[idx] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => removeSecretRow(idx)} aria-label="Remove row">
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              ))}
            </div>
            <Button variant="outline" size="sm" onClick={addSecretRow}>
              <Plus className="w-4 h-4 mr-2" /> Add Environment Variable
            </Button>
          </div>

          {/* Save Button */}
          <div className="pt-4 border-t">
            <Button
              onClick={handleSaveIntegrationSecrets}
              disabled={updateIntegrationSecretsMutation.isPending}
            >
              {updateIntegrationSecretsMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="w-4 h-4 mr-2" />
                  Save Integration Secrets
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

