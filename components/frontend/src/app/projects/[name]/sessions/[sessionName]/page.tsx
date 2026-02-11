"use client";

import { useState, useEffect, useMemo, useRef, useCallback } from "react";
import {
  Loader2,
  FolderTree,
  GitBranch,
  Folder,
  Sparkles,
  CloudUpload,
  Cloud,
  FolderSync,
  Download,
  SlidersHorizontal,
  ArrowLeft,
  AlertTriangle,
  X,
  MoreVertical,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
} from "lucide-react";
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "@/components/ui/resizable";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

// Custom components
import { CopilotSessionProvider, CopilotChatView } from "@/components/session/CopilotChatPanel";
import type { InputEnhancementsCtxType } from "@/components/session/session-contexts";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { SessionHeader } from "./session-header";
import { getPhaseColor } from "@/utils/session-helpers";

// Extracted components
import { AddContextModal } from "./components/modals/add-context-modal";
import { UploadFileModal } from "./components/modals/upload-file-modal";
import { CustomWorkflowDialog } from "./components/modals/custom-workflow-dialog";
import { ManageRemoteDialog } from "./components/modals/manage-remote-dialog";
import { WorkflowsAccordion } from "./components/accordions/workflows-accordion";
import { RepositoriesAccordion } from "./components/accordions/repositories-accordion";
import { ArtifactsAccordion } from "./components/accordions/artifacts-accordion";
import { McpServersAccordion, IntegrationsAccordion } from "./components/accordions/mcp-integrations-accordion";
import { WelcomeExperience } from "./components/welcome-experience";
// Extracted hooks and utilities
import { useGitOperations } from "./hooks/use-git-operations";
import { useWorkflowManagement } from "./hooks/use-workflow-management";
import { useFileOperations } from "./hooks/use-file-operations";
import { useSessionQueue } from "@/hooks/use-session-queue";
import type { DirectoryOption, DirectoryRemote } from "./lib/types";

import type { ReconciledRepo, SessionRepo, AgenticSession, AgenticSessionPhase } from "@/types/agentic-session";
import { SessionStartingEvents } from "@/components/session/SessionStartingEvents";

// React Query hooks
import {
  useSession,
  useStopSession,
  useDeleteSession,
  useContinueSession,
  useReposStatus,
} from "@/services/queries";
import {
  useWorkspaceList,
} from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";
import {
  useOOTBWorkflows,
  useWorkflowMetadata,
} from "@/services/queries/use-workflows";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import { useMutation } from "@tanstack/react-query";

// Constants for artifact auto-refresh timing
// Moved outside component to avoid unnecessary effect re-runs
//
// Wait 1 second after last tool completion to batch rapid writes together
// Prevents excessive API calls during burst writes (e.g., when Claude creates multiple files in quick succession)
// Testing: 500ms was too aggressive (hit API rate limits), 2000ms felt sluggish to users


// Wait 2 seconds after session completes before final artifact refresh
// Backend can take 1-2 seconds to flush final artifacts to storage
// Ensures users see all artifacts even if final writes occur after status transition
const COMPLETION_DELAY_MS = 2000;

// NOTE: isCompletedToolUseMessage type guard removed — was only used by the
// old streamMessages useMemo. CopilotKit handles tool rendering now.

// ─── Workflow connect bridge ──────────────────────────────────────────
//
// Headless component that sits inside CopilotSessionProvider.
// When connectSignal changes (workflow activated), it calls
// agent.connectAgent() to replay persisted events — picking up
// server-initiated events like the runner's workflow greeting.

import { useAgent } from "@copilotkit/react-core/v2";

function WorkflowConnectBridge({
  sessionName,
  connectSignal,
}: {
  sessionName: string;
  connectSignal: number;
}) {
  const { agent } = useAgent({ agentId: sessionName });

  useEffect(() => {
    if (connectSignal === 0) return; // skip initial mount
    agent.connectAgent().catch((err: unknown) => {
      console.warn("[WorkflowConnectBridge] connectAgent failed:", err);
    });
  }, [connectSignal, agent]);

  return null;
}

// ─── Phase-aware overlay for non-running sessions ─────────────────────

type SessionPhaseOverlayProps = {
  phase: AgenticSessionPhase;
  session: AgenticSession | undefined;
  projectName: string;
  sessionName: string;
  onResume: () => void;
  isResuming: boolean;
};

function SessionPhaseOverlay({
  phase,
  session,
  projectName,
  sessionName,
  onResume,
  isResuming,
}: SessionPhaseOverlayProps) {
  // Creating / Pending — show live pod events timeline
  if (phase === "Creating" || phase === "Pending") {
    return (
      <SessionStartingEvents
        projectName={projectName}
        sessionName={sessionName}
      />
    );
  }

  // Stopping — simple spinner
  if (phase === "Stopping") {
    return (
      <div className="flex flex-col items-center justify-center h-full">
        <Loader2 className="h-10 w-10 animate-spin text-orange-500 mb-3" />
        <h3 className="font-semibold text-lg">Stopping Session</h3>
        <p className="text-sm text-muted-foreground mt-1">
          Saving workspace state...
        </p>
      </div>
    );
  }

  // Failed — show error from conditions
  if (phase === "Failed") {
    const conditions = session?.status?.conditions ?? [];
    const failedCondition = conditions.find(
      (c) => c.status === "False" && c.message,
    );
    const errorMessage =
      failedCondition?.message ?? "Session failed unexpectedly.";
    const errorReason = failedCondition?.reason;

    return (
      <div className="flex flex-col items-center justify-center h-full px-4">
        <div className="max-w-md w-full text-center">
          <div className="mb-4 rounded-full bg-red-500/10 p-4 inline-flex">
            <AlertCircle className="h-8 w-8 text-red-500" />
          </div>
          <h3 className="font-semibold text-lg mb-2">Session Failed</h3>
          {errorReason && (
            <Badge variant="destructive" className="mb-3 text-xs">
              {errorReason}
            </Badge>
          )}
          <p className="text-sm text-muted-foreground mb-6 break-words">
            {errorMessage}
          </p>
          <Button onClick={onResume} size="lg" className="w-full" disabled={isResuming}>
            {isResuming ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Resuming...
              </>
            ) : (
              "Retry Session"
            )}
          </Button>
        </div>
      </div>
    );
  }

  // Fallback (shouldn't normally be reached — Stopped/Completed use CopilotChatView)
  return null;
}

export default function ProjectSessionDetailPage({
  params,
}: {
  params: Promise<{ name: string; sessionName: string }>;
}) {
  const router = useRouter();
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>([]);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [repoChanging, setRepoChanging] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [userHasInteracted, setUserHasInteracted] = useState(false);
  // Incremented after workflow activation to signal CopilotKit to reconnect
  // via connectAgent() — picks up server-initiated events (e.g. greeting).
  const [connectSignal, setConnectSignal] = useState(0);

  // Left panel visibility and size state (persisted to localStorage)
  const [leftPanelVisible, setLeftPanelVisible] = useState(() => {
    if (typeof window === 'undefined') return true;
    const saved = localStorage.getItem('session-left-panel-visible');
    return saved === null ? true : saved === 'true';
  });
  

  // Directory browser state (unified for artifacts, repos, and workflow)
  const [selectedDirectory, setSelectedDirectory] = useState<DirectoryOption>({
    type: "artifacts",
    name: "Shared Artifacts",
    path: "artifacts",
  });
  const [directoryRemotes, setDirectoryRemotes] = useState<
    Record<string, DirectoryRemote>
  >({});
  const [remoteDialogOpen, setRemoteDialogOpen] = useState(false);
  const [customWorkflowDialogOpen, setCustomWorkflowDialogOpen] =
    useState(false);

  // Extract params
  useEffect(() => {
    params.then(({ name, sessionName: sName }) => {
      setProjectName(name);
      setSessionName(sName);
      try {
        const url = new URL(window.location.href);
        setBackHref(url.searchParams.get("backHref"));
      } catch {}
    });
  }, [params]);

  // Persist left panel visibility
  useEffect(() => {
    localStorage.setItem('session-left-panel-visible', String(leftPanelVisible));
  }, [leftPanelVisible]);

  // Session queue hook (localStorage-backed)
  const sessionQueue = useSessionQueue(projectName, sessionName);

  // React Query hooks
  const {
    data: session,
    isLoading,
    error,
    refetch: refetchSession,
  } = useSession(projectName, sessionName);
  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const continueMutation = useContinueSession();
  
  // Check integration status
  const { data: integrationsStatus } = useIntegrationsStatus();
  const githubConfigured = integrationsStatus?.github?.active != null;
  
  // Extract phase for sidebar state management
  const phase = session?.status?.phase || "Pending";

  // Fetch repos status directly from runner (real-time branch info)
  const { data: reposStatus } = useReposStatus(
    projectName,
    sessionName,
    phase === "Running" // Only poll when session is running
  );

  // Workflow management hook
  const workflowManagement = useWorkflowManagement({
    projectName,
    sessionName,
    sessionPhase: session?.status?.phase,
    onWorkflowActivated: () => {
      refetchSession();
      // Signal the CopilotKit bridge to call connectAgent() — this replays
      // persisted events including the runner's workflow greeting.
      setConnectSignal((n) => n + 1);
    },
  });

  // Poll session status when workflow is queued
  useEffect(() => {
    if (!workflowManagement.queuedWorkflow) return;
    
    const phase = session?.status?.phase;
    
    // If already running, we'll process workflow in the next effect
    if (phase === "Running") return;
    
    // Poll every 2 seconds to check if session is ready
    const pollInterval = setInterval(() => {
      refetchSession();
    }, 2000);
    
    return () => clearInterval(pollInterval);
  }, [workflowManagement.queuedWorkflow, session?.status?.phase, refetchSession]);

  // Process queued workflow when session becomes Running
  useEffect(() => {
    const phase = session?.status?.phase;
    const queuedWorkflow = workflowManagement.queuedWorkflow;
    if (phase === "Running" && queuedWorkflow && !queuedWorkflow.activatedAt) {
      // Session is now running, activate the queued workflow
      workflowManagement.activateWorkflow({
        id: queuedWorkflow.id,
        name: "Queued workflow",
        description: "",
        gitUrl: queuedWorkflow.gitUrl,
        branch: queuedWorkflow.branch,
        path: queuedWorkflow.path,
        enabled: true,
      }, phase);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.status?.phase, workflowManagement.queuedWorkflow]);

  // Poll session status when messages are queued
  useEffect(() => {
    const queuedMessages = sessionQueue.messages.filter(m => !m.sentAt);
    if (queuedMessages.length === 0) return;
    
    const phase = session?.status?.phase;
    
    // If already running, we'll process messages in the next effect
    if (phase === "Running") return;
    
    // Poll every 2 seconds to check if session is ready
    const pollInterval = setInterval(() => {
      refetchSession();
    }, 2000);
    
    return () => clearInterval(pollInterval);
  }, [sessionQueue.messages, session?.status?.phase, refetchSession]);

  // Note: Message sending is handled by CopilotKit's chat component.
  // Queued messages are cleared when session becomes Running and CopilotKit connects.

  // Repo management mutations
  const addRepoMutation = useMutation({
    mutationFn: async (repo: { url: string; branch: string; autoPush?: boolean }) => {
      setRepoChanging(true);
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/repos`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(repo),
        },
      );
      if (!response.ok) throw new Error("Failed to add repository");
      const result = await response.json();
      return { ...result, inputRepo: repo };
    },
    onSuccess: async (data) => {
      successToast("Repository cloning...");
      await new Promise((resolve) => setTimeout(resolve, 3000));
      await refetchSession();

      if (data.name && data.inputRepo) {
        try {
          // Repos are cloned to /workspace/repos/{name}
          const repoPath = `repos/${data.name}`;
          await fetch(
            `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/configure-remote`,
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                path: repoPath,
                remoteUrl: data.inputRepo.url,
                branch: data.inputRepo.branch || "main",
              }),
            },
          );

          const newRemotes = { ...directoryRemotes };
          newRemotes[repoPath] = {
            url: data.inputRepo.url,
            branch: data.inputRepo.branch || "main",
          };
          setDirectoryRemotes(newRemotes);
        } catch (err) {
          console.error("Failed to configure remote:", err);
        }
      }

      setRepoChanging(false);
      successToast("Repository added successfully");
    },
    onError: (error: Error) => {
      setRepoChanging(false);
      errorToast(error.message || "Failed to add repository");
    },
  });

  const removeRepoMutation = useMutation({
    mutationFn: async (repoName: string) => {
      setRepoChanging(true);
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/repos/${repoName}`,
        { method: "DELETE" },
      );
      if (!response.ok) throw new Error("Failed to remove repository");
      return response.json();
    },
    onSuccess: async () => {
      successToast("Repository removing...");
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await refetchSession();
      setRepoChanging(false);
      successToast("Repository removed successfully");
    },
    onError: (error: Error) => {
      setRepoChanging(false);
      errorToast(error.message || "Failed to remove repository");
    },
  });

  // File upload mutation
  const uploadFileMutation = useMutation({
    mutationFn: async (source: {
      type: "local" | "url";
      file?: File;
      url?: string;
      filename?: string;
    }) => {
      const formData = new FormData();
      formData.append("type", source.type);

      if (source.type === "local" && source.file) {
        formData.append("file", source.file);
        formData.append("filename", source.file.name);
      } else if (source.type === "url" && source.url && source.filename) {
        formData.append("url", source.url);
        formData.append("filename", source.filename);
      }

      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/workspace/upload`,
        {
          method: "POST",
          body: formData,
        },
      );

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Upload failed");
      }

      return response.json();
    },
    onSuccess: async (data) => {
      successToast(`File "${data.filename}" uploaded successfully`);
      // Refresh workspace to show uploaded file
      await refetchFileUploadsList();
      await refetchDirectoryFiles();
      await refetchArtifactsFiles();
      setUploadModalOpen(false);
    },
    onError: (error: Error) => {
      errorToast(error.message || "Failed to upload file");
    },
  });

  // File removal mutation
  const removeFileMutation = useMutation({
    mutationFn: async (fileName: string) => {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/workspace/file-uploads/${fileName}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to remove file");
      }

      return response.json();
    },
    onSuccess: async () => {
      successToast("File removed successfully");
      // Refresh file lists
      await refetchFileUploadsList();
      await refetchDirectoryFiles();
    },
    onError: (error: Error) => {
      errorToast(error.message || "Failed to remove file");
    },
  });

  // Fetch OOTB workflows
  const { data: ootbWorkflows = [] } = useOOTBWorkflows(projectName);

  // Fetch workflow metadata
  const { data: workflowMetadata } = useWorkflowMetadata(
    projectName,
    sessionName,
    !!workflowManagement.activeWorkflow &&
      !workflowManagement.workflowActivating,
  );

  // Git operations for selected directory
  const currentRemote = directoryRemotes[selectedDirectory.path];

  // Removed: mergeStatus and remoteBranches - agent handles all git operations now

  // Git operations hook
  const gitOps = useGitOperations({
    projectName,
    sessionName,
    directoryPath: selectedDirectory.path,
    remoteBranch: currentRemote?.branch || "main",
  });

  // Get repo info from reposStatus for repo-type directories
  const repoInfo = selectedDirectory.type === "repo"
    ? reposStatus?.repos?.find((r) => r.name === selectedDirectory.name)
    : undefined;

  // Get current branch for selected directory (use real-time reposStatus for repos)
  const currentBranch = selectedDirectory.type === "repo"
    ? repoInfo?.currentActiveBranch || gitOps.gitStatus?.branch || "main"
    : gitOps.gitStatus?.branch || "main";

  // Get hasRemote status for selected directory (use real-time reposStatus for repos)
  const hasRemote = selectedDirectory.type === "repo"
    ? !!repoInfo?.url
    : gitOps.gitStatus?.hasRemote ?? false;

  // Get remote URL for selected directory (use real-time reposStatus for repos)
  const remoteUrl = selectedDirectory.type === "repo"
    ? repoInfo?.url
    : gitOps.gitStatus?.remoteUrl;

  // File operations for directory explorer
  const fileOps = useFileOperations({
    projectName,
    sessionName,
    basePath: selectedDirectory.path,
  });

  const { data: directoryFiles = [], refetch: refetchDirectoryFiles } =
    useWorkspaceList(
      projectName,
      sessionName,
      fileOps.currentSubPath
        ? `${selectedDirectory.path}/${fileOps.currentSubPath}`
        : selectedDirectory.path,
      { enabled: openAccordionItems.includes("file-explorer") },
    );

  // Artifacts file operations
  const artifactsOps = useFileOperations({
    projectName,
    sessionName,
    basePath: "artifacts",
  });

  const { data: artifactsFiles = [], refetch: refetchArtifactsFilesRaw } =
    useWorkspaceList(
      projectName,
      sessionName,
      artifactsOps.currentSubPath
        ? `artifacts/${artifactsOps.currentSubPath}`
        : "artifacts",
    );

  // Stabilize refetchArtifactsFiles with useCallback to prevent infinite re-renders
  // React Query's refetch is already stable, but this ensures proper dependency tracking
  const refetchArtifactsFiles = useCallback(async () => {
    try {
      await refetchArtifactsFilesRaw();
    } catch (error) {
      console.error('Failed to refresh artifacts:', error);
      // Silent fail - don't interrupt user experience
    }
  }, [refetchArtifactsFilesRaw]);

  // File uploads list (for Context accordion)
  const { data: fileUploadsList = [], refetch: refetchFileUploadsList } =
    useWorkspaceList(
      projectName,
      sessionName,
      "file-uploads",
      { enabled: openAccordionItems.includes("context") },
    );

  // Track if we've already initialized from session
  const initializedFromSessionRef = useRef(false);
  const workflowLoadedFromSessionRef = useRef(false);

  // Note: userHasInteracted is only set when:
  // 1. User explicitly selects a workflow (handleWelcomeWorkflowSelect -> onUserInteraction)
  // 2. User sends a message via CopilotKit chat
  // It should NOT be set automatically when backend messages arrive

  // Load remotes from session annotations (one-time initialization)
  useEffect(() => {
    if (initializedFromSessionRef.current || !session) return;

    const annotations = session.metadata?.annotations || {};
    const remotes: Record<string, DirectoryRemote> = {};

    Object.keys(annotations).forEach((key) => {
      if (key.startsWith("ambient-code.io/remote-") && key.endsWith("-url")) {
        const path = key
          .replace("ambient-code.io/remote-", "")
          .replace("-url", "")
          .replace(/::/g, "/");
        const branchKey = key.replace("-url", "-branch");
        remotes[path] = {
          url: annotations[key],
          branch: annotations[branchKey] || "main",
        };
      }
    });

    setDirectoryRemotes(remotes);
    initializedFromSessionRef.current = true;
  }, [session]);

  // Compute directory options
  const directoryOptions = useMemo<DirectoryOption[]>(() => {
    const options: DirectoryOption[] = [
      { type: "artifacts", name: "Shared Artifacts", path: "artifacts" },
      { type: "file-uploads", name: "File Uploads", path: "file-uploads" },
    ];

    // Use real-time repos status from runner when available, otherwise fall back to CR status
    const reposToDisplay = reposStatus?.repos || session?.status?.reconciledRepos || session?.spec?.repos || [];

    // Deduplicate repos by name - only show one entry per repo directory
    const seenRepos = new Set<string>();
    reposToDisplay.forEach((repo: ReconciledRepo | SessionRepo) => {
      const repoName = ('name' in repo ? repo.name : undefined) || repo.url?.split('/').pop()?.replace('.git', '') || 'repo';

      // Skip if we've already added this repo
      if (seenRepos.has(repoName)) {
        return;
      }
      seenRepos.add(repoName);

      // Repos are cloned to /workspace/repos/{name}
      options.push({
        type: "repo",
        name: repoName,
        path: `repos/${repoName}`,
      });
    });

    if (workflowManagement.activeWorkflow && session?.spec?.activeWorkflow) {
      const workflowName =
        session.spec.activeWorkflow.gitUrl
          .split("/")
          .pop()
          ?.replace(".git", "") || "workflow";
      options.push({
        type: "workflow",
        name: `Workflow: ${workflowName}`,
        path: `workflows/${workflowName}`,
      });
    }

    return options;
  }, [session, workflowManagement.activeWorkflow, reposStatus]);

  // Workflow change handler
  const handleWorkflowChange = (value: string) => {
    const workflow = workflowManagement.handleWorkflowChange(value, ootbWorkflows, () =>
      setCustomWorkflowDialogOpen(true),
    );
    // Automatically trigger activation with the workflow directly (avoids state timing issues)
    if (workflow) {
      workflowManagement.activateWorkflow(workflow, session?.status?.phase);
    }
  };

  // Handle workflow selection from welcome experience
  const handleWelcomeWorkflowSelect = (workflowId: string) => {
    handleWorkflowChange(workflowId);
  };

  // Session has real messages when phase indicates activity has occurred.
  // CopilotKit handles all message display — this is for sidebar logic only.
  const hasRealMessages = useMemo(() => {
    const phase = session?.status?.phase;
    return phase === "Running" || phase === "Completed";
  }, [session?.status?.phase]);

  // Input enhancements — bridges page-level state (upload, queue) to CopilotKit's
  // custom Input component via context.  See session-contexts.ts.
  const inputEnhancements = useMemo<InputEnhancementsCtxType>(() => ({
    onPasteImage: async (file: File) => {
      await uploadFileMutation.mutateAsync({ type: "local", file });
    },
    sessionPhase: phase,
    queuedMessages: sessionQueue.messages,
    onCancelQueuedMessage: (id: string) => sessionQueue.cancelMessage(id),
    onUpdateQueuedMessage: (id: string, content: string) => sessionQueue.updateMessage(id, content),
    onClearQueue: () => sessionQueue.clearMessages(),
    queuedCount: sessionQueue.pendingCount,
    isRunActive: phase === "Running",
    onMarkSent: (id: string) => sessionQueue.markMessageSent(id),
    onQueueMessage: (content: string) => sessionQueue.addMessage(content),
  }), [phase, sessionQueue, uploadFileMutation]);

  // Load workflow from session when session data and workflows are available
  // Syncs the workflow panel with the workflow reported by the API
  useEffect(() => {
    if (workflowLoadedFromSessionRef.current || !session) return;
    if (session.spec?.activeWorkflow && ootbWorkflows.length === 0) return;

    // Sync workflow from session whenever it's set in the API
    if (session.spec?.activeWorkflow) {
      // Match by path (e.g., "workflows/spec-kit") - this uniquely identifies each OOTB workflow
      // Don't match by gitUrl since all OOTB workflows share the same repo URL
      const activePath = session.spec.activeWorkflow.path;
      const matchingWorkflow = ootbWorkflows.find((w) => w.path === activePath);
      if (matchingWorkflow) {
        workflowManagement.setActiveWorkflow(matchingWorkflow.id);
        workflowManagement.setSelectedWorkflow(matchingWorkflow.id);
        // Mark as interacted for existing sessions with messages
        if (hasRealMessages) {
          setUserHasInteracted(true);
        }
      } else {
        // No matching OOTB workflow found - treat as custom workflow
        workflowManagement.setActiveWorkflow("custom");
        workflowManagement.setSelectedWorkflow("custom");
        if (hasRealMessages) {
          setUserHasInteracted(true);
        }
      }
      workflowLoadedFromSessionRef.current = true;
    }
  }, [session, ootbWorkflows, workflowManagement, hasRealMessages]);

  // Auto-refresh artifacts periodically while session is running
  // CopilotKit handles chat — we just poll artifacts during active sessions
  const completionTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hasRefreshedOnCompletionRef = useRef(false);

  useEffect(() => {
    const phase = session?.status?.phase;
    if (phase !== "Running") return;

    // Poll artifacts every 10 seconds while session is running
    const interval = setInterval(() => {
        refetchArtifactsFiles();
    }, 10_000);

    return () => clearInterval(interval);
  }, [session?.status?.phase, refetchArtifactsFiles]);

  // Also refresh artifacts when session completes (catch any final artifacts)
  useEffect(() => {
    const phase = session?.status?.phase;
    if (phase === "Completed" && !hasRefreshedOnCompletionRef.current) {
      // Refresh after a short delay to ensure all final writes are complete
      completionTimeoutRef.current = setTimeout(() => {
        refetchArtifactsFiles();
      }, COMPLETION_DELAY_MS);
      hasRefreshedOnCompletionRef.current = true;
    } else if (phase !== "Completed") {
      // Clear any pending completion refresh to avoid race conditions
      if (completionTimeoutRef.current) {
        clearTimeout(completionTimeoutRef.current);
        completionTimeoutRef.current = null;
      }
      // Reset flag whenever leaving Completed state (handles Running, Error, Cancelled, etc.)
      hasRefreshedOnCompletionRef.current = false;
    }

    // Cleanup timeout on unmount or phase change
    return () => {
      if (completionTimeoutRef.current) {
        clearTimeout(completionTimeoutRef.current);
      }
    };
  }, [session?.status?.phase, refetchArtifactsFiles]);
  // Session action handlers
  const handleStop = () => {
    stopMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => successToast("Session stopped successfully"),
        onError: (err) =>
          errorToast(
            err instanceof Error ? err.message : "Failed to stop session",
          ),
      },
    );
  };

  const handleDelete = () => {
    const displayName = session?.spec.displayName || session?.metadata.name;
    if (
      !confirm(
        `Are you sure you want to delete agentic session "${displayName}"? This action cannot be undone.`,
      )
    ) {
      return;
    }

    deleteMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          router.push(
            backHref || `/projects/${encodeURIComponent(projectName)}/sessions`,
          );
        },
        onError: (err) =>
          errorToast(
            err instanceof Error ? err.message : "Failed to delete session",
          ),
      },
    );
  };

  const handleContinue = () => {
    continueMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => {
          successToast("Session restarted successfully");
        },
        onError: (err) =>
          errorToast(
            err instanceof Error ? err.message : "Failed to restart session",
          ),
      },
    );
  };

  // NOTE: sendChat, handleCommandClick, handleInterrupt removed.
  // CopilotKit's <CopilotChat> component handles all message sending and interruption.

  // Loading state
  if (isLoading || !projectName || !sessionName) {
    return (
      <div className="absolute inset-0 top-16 overflow-hidden bg-background flex items-center justify-center">
        <div className="flex items-center">
          <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
          <span className="ml-2">Loading session...</span>
        </div>
      </div>
    );
  }

  // Error state
  if (error || !session) {
    return (
      <div className="absolute inset-0 top-16 overflow-hidden bg-background flex flex-col">
        <div className="flex-shrink-0 bg-card border-b">
          <div className="container mx-auto px-6 py-4">
            <Breadcrumbs
              items={[
                { label: "Workspaces", href: "/projects" },
                { label: projectName, href: `/projects/${projectName}` },
                {
                  label: "Sessions",
                  href: `/projects/${projectName}/sessions`,
                },
                { label: "Error" },
              ]}
              className="mb-4"
            />
          </div>
        </div>
        <div className="flex-grow overflow-hidden">
          <div className="h-full container mx-auto px-6 py-6">
            <Card className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/50">
              <CardContent className="pt-6">
                <p className="text-red-700 dark:text-red-300">
                  Error:{" "}
                  {error instanceof Error ? error.message : "Session not found"}
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="absolute inset-0 top-16 overflow-hidden bg-background flex flex-col">
        {/* Fixed header */}
        <div className="flex-shrink-0 bg-card border-b">
          <div className="px-6 py-4">
            <div className="space-y-3 md:space-y-0">
              {/* Top row: Back button / Breadcrumb + Kebab menu */}
              <div className="flex items-center justify-between">
                {/* Mobile: Back button + Session name */}
                <div className="flex items-center gap-3 md:hidden">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => router.push(`/projects/${projectName}/sessions`)}
                    className="h-8 w-8 p-0"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </Button>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium truncate max-w-[150px]">
                      {session.spec.displayName || session.metadata.name}
                    </span>
                    <Badge
                      className={getPhaseColor(
                        session.status?.phase || "Pending",
                      )}
                    >
                      {session.status?.phase || "Pending"}
                    </Badge>
                  </div>
                </div>

                {/* Desktop: Full breadcrumb */}
                <div className="hidden md:block">
                  <Breadcrumbs
                    items={[
                      { label: "Workspaces", href: "/projects" },
                      { label: projectName, href: `/projects/${projectName}` },
                      {
                        label: "Sessions",
                        href: `/projects/${projectName}/sessions`,
                      },
                      {
                        label: session.spec.displayName || session.metadata.name,
                        rightIcon: (
                          <Badge
                            className={getPhaseColor(
                              session.status?.phase || "Pending",
                            )}
                          >
                            {session.status?.phase || "Pending"}
                          </Badge>
                        ),
                      },
                    ]}
                  />
                </div>

                {/* Kebab menu (both mobile and desktop) */}
                <SessionHeader
                  session={session}
                  projectName={projectName}
                  actionLoading={
                    stopMutation.isPending
                      ? "stopping"
                      : deleteMutation.isPending
                        ? "deleting"
                        : continueMutation.isPending
                          ? "resuming"
                          : null
                  }
                  onRefresh={refetchSession}
                  onStop={handleStop}
                  onContinue={handleContinue}
                  onDelete={handleDelete}
                  renderMode="kebab-only"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Mobile: Options menu button (below header border) - always show */}
        {session && (
          <div className="md:hidden px-6 py-1 bg-card border-b">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="h-8 w-8 p-0"
            >
              <SlidersHorizontal className="h-4 w-4" />
            </Button>
          </div>
        )}

        {/* Main content area — single CopilotKit provider for both layouts */}
        <CopilotSessionProvider projectName={projectName} sessionName={sessionName}>
        <WorkflowConnectBridge sessionName={sessionName} connectSignal={connectSignal} />
        <div className="flex-grow overflow-hidden bg-card">
          <div className="h-full relative">
              {/* Mobile sidebar overlay */}
              {mobileMenuOpen && (
                <div 
                  className="fixed inset-0 bg-background/80 backdrop-blur-sm z-40 md:hidden"
                  onClick={() => setMobileMenuOpen(false)}
                />
              )}

            {/* Mobile Left Column (overlay) */}
            {session && mobileMenuOpen && (
                <div className={cn(
                "fixed left-0 top-16 z-50 shadow-lg flex flex-col md:hidden",
                "w-[400px] h-[calc(100vh-4rem)] pt-6 pl-6 pr-6 bg-card relative",
                  phase !== "Running" && "pointer-events-none"
                )}>
                  {/* Backdrop blur layer for entire sidebar */}
                  {phase !== "Running" && (
                    <div className={cn(
                      "absolute inset-0 z-[5] backdrop-blur-[1px]",
                      ["Creating", "Pending", "Stopping"].includes(phase) && "bg-background/40",
                      ["Stopped", "Completed", "Failed"].includes(phase) && "bg-background/50 backdrop-blur-[2px]"
                    )} />
                  )}

                  {/* State overlay for non-running sessions */}
                  {phase !== "Running" && (
                    <div className="absolute inset-0 z-10 flex items-center justify-center pointer-events-auto">
                      <div className="text-center">
                        {/* Starting states */}
                        {["Creating", "Pending"].includes(phase) && (
                          <>
                            <Loader2 className="h-10 w-10 mx-auto mb-3 animate-spin text-blue-600" />
                            <h3 className="font-semibold text-lg mb-1">Starting Session</h3>
                            <p className="text-sm text-muted-foreground">
                              Setting up your workspace...
                            </p>
                          </>
                        )}
                        
                        {/* Stopping state */}
                        {phase === "Stopping" && (
                          <>
                            <Loader2 className="h-10 w-10 mx-auto mb-3 animate-spin text-orange-600" />
                            <h3 className="font-semibold text-lg mb-1">Stopping Session</h3>
                            <p className="text-sm text-muted-foreground">
                              Saving workspace state...
                            </p>
                          </>
                        )}
                        
                        {/* Hibernated states */}
                        {["Stopped", "Completed", "Failed"].includes(phase) && (
                          <div className="max-w-sm">
                            <h3 className="font-semibold text-lg mb-4">Session Hibernated</h3>
                            
                            {/* Session details */}
                            <div className="space-y-3 mb-6 text-left">
                              {workflowManagement.activeWorkflow && (
                                <div>
                                  <p className="text-xs font-medium text-muted-foreground mb-1.5">Workflow</p>
                                  <Badge variant="secondary" className="text-xs">
                                    {workflowManagement.activeWorkflow}
                                  </Badge>
                                </div>
                              )}
                              
                              {session?.spec?.repos && session.spec.repos.length > 0 && (
                                <div>
                                  <p className="text-xs font-medium text-muted-foreground mb-1.5">
                                    Repositories ({session.spec.repos.length})
                                  </p>
                                  <div className="text-sm text-foreground/80 space-y-1">
                                    {session.spec.repos.slice(0, 3).map((repo, idx) => (
                                      <div key={idx} className="truncate">
                                        • {repo.url?.split('/').pop()?.replace('.git', '')}
                                      </div>
                                    ))}
                                    {session.spec.repos.length > 3 && (
                                      <div className="text-xs text-muted-foreground mt-1">
                                        +{session.spec.repos.length - 3} more
                                      </div>
                                    )}
                                  </div>
                                </div>
                              )}
                              
                              {(!workflowManagement.activeWorkflow && (!session?.spec?.repos || session.spec.repos.length === 0)) && (
                                <div className="text-center py-2">
                                  <p className="text-xs text-muted-foreground">
                                    No workflow or repositories configured
                                  </p>
                                </div>
                              )}
                            </div>
                            
                            <Button onClick={handleContinue} size="lg" className="w-full" disabled={continueMutation.isPending}>
                              {continueMutation.isPending ? (
                                <>
                                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                  Resuming...
                                </>
                              ) : (
                                'Resume Session'
                              )}
                            </Button>
                          </div>
                        )}
                      </div>
                    </div>
                  )}

                {/* Mobile close button */}
                <div className="md:hidden flex justify-end mb-4">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setMobileMenuOpen(false)}
                    className="h-8 w-8 p-0"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <div className={cn(
                  "flex-grow pb-6 overflow-y-auto scrollbar-hide",
                  ["Stopped", "Completed", "Failed"].includes(phase) && "blur-[2px]"
                )}>
                  <Accordion
                    type="multiple"
                    value={openAccordionItems}
                    onValueChange={phase === "Running" ? setOpenAccordionItems : undefined}
                    className="w-full space-y-3"
                  >
                    <WorkflowsAccordion
                      sessionPhase={session?.status?.phase}
                      activeWorkflow={workflowManagement.activeWorkflow}
                      selectedWorkflow={workflowManagement.selectedWorkflow}
                      workflowActivating={workflowManagement.workflowActivating}
                      ootbWorkflows={ootbWorkflows}
                      isExpanded={openAccordionItems.includes("workflows")}
                      onWorkflowChange={handleWorkflowChange}
                      onResume={handleContinue}
                    />

                    <RepositoriesAccordion
                      repositories={reposStatus?.repos || session?.status?.reconciledRepos || session?.spec?.repos || []}
                      uploadedFiles={fileUploadsList.map((f) => ({
                        name: f.name,
                        path: f.path,
                        size: f.size,
                      }))}
                      onAddRepository={() => setContextModalOpen(true)}
                      onRemoveRepository={(repoName) =>
                        removeRepoMutation.mutate(repoName)
                      }
                      onRemoveFile={(fileName) =>
                        removeFileMutation.mutate(fileName)
                      }
                    />

                    <ArtifactsAccordion
                      files={artifactsFiles}
                      currentSubPath={artifactsOps.currentSubPath}
                      viewingFile={artifactsOps.viewingFile}
                      isLoadingFile={artifactsOps.loadingFile}
                      onFileOrFolderSelect={
                        artifactsOps.handleFileOrFolderSelect
                      }
                      onRefresh={refetchArtifactsFiles}
                      onDownloadFile={artifactsOps.handleDownloadFile}
                      onNavigateBack={artifactsOps.navigateBack}
                    />

                    <McpServersAccordion
                      projectName={projectName}
                      sessionName={sessionName}
                      sessionPhase={phase}
                    />

                    <IntegrationsAccordion />

                    {/* File Explorer */}
                    <AccordionItem
                      value="file-explorer"
                      className="border rounded-lg px-3 bg-card"
                    >
                      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                        <div className="flex items-center gap-2 w-full">
                          <Folder className="h-4 w-4" />
                          <span>File Explorer</span>
                          <Badge
                            variant="outline"
                            className="text-[10px] px-2 py-0.5"
                          >
                            EXPERIMENTAL
                          </Badge>
                          {gitOps.gitStatus?.hasChanges && (
                            <div className="flex gap-1 ml-auto mr-2">
                              {(gitOps.gitStatus?.totalAdded ?? 0) > 0 && (
                                <Badge
                                  variant="outline"
                                  className="bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800"
                                >
                                  +{gitOps.gitStatus.totalAdded}
                                </Badge>
                              )}
                              {(gitOps.gitStatus?.totalRemoved ?? 0) > 0 && (
                                <Badge
                                  variant="outline"
                                  className="bg-red-50 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-300 dark:border-red-800"
                                >
                                  -{gitOps.gitStatus.totalRemoved}
                                </Badge>
                              )}
                            </div>
                          )}
                        </div>
                      </AccordionTrigger>
                      <AccordionContent className="pt-2 pb-3">
                        <div className="space-y-3">
                          <p className="text-sm text-muted-foreground">
                            Browse, view, and manage files in your workspace
                            directories. Track changes and sync with Git for
                            version control.
                          </p>

                          {/* Directory Selector */}
                          <div className="flex items-center justify-between gap-2">
                            <Label className="text-xs text-muted-foreground">
                              Directory:
                            </Label>
                            <Select
                              value={`${selectedDirectory.type}:${selectedDirectory.path}`}
                              onValueChange={(value) => {
                                const [type, ...pathParts] = value.split(":");
                                const path = pathParts.join(":");
                                const option = directoryOptions.find(
                                  (opt) =>
                                    opt.type === type && opt.path === path,
                                );
                                if (option) setSelectedDirectory(option);
                              }}
                            >
                              <SelectTrigger className="w-[300px] h-auto min-h-[2.5rem] py-2.5 overflow-visible">
                                <div className="flex items-center gap-2 flex-wrap w-full pr-6 overflow-visible">
                                  <SelectValue />
                                </div>
                              </SelectTrigger>
                              <SelectContent>
                                {directoryOptions.map((opt) => {
                                  // Find branch info for repo directories from real-time status
                                  let branchName: string | undefined;
                                  if (opt.type === "repo") {
                                    // Extract repo name from path (repos/repoName -> repoName)
                                    const repoName = opt.path.replace(/^repos\//, "");

                                    // Try real-time repos status first
                                    const realtimeRepo = reposStatus?.repos?.find(
                                      (r) => r.name === repoName
                                    );

                                    // Fall back to CR status
                                    const reconciledRepo = session?.status?.reconciledRepos?.find(
                                      (r: ReconciledRepo) => {
                                        const rName = r.name || r.url?.split("/").pop()?.replace(".git", "");
                                        return rName === repoName;
                                      }
                                    );

                                    branchName = realtimeRepo?.currentActiveBranch
                                      || reconciledRepo?.currentActiveBranch
                                      || reconciledRepo?.branch;
                                  }

                                  return (
                                    <SelectItem
                                      key={`${opt.type}:${opt.path}`}
                                      value={`${opt.type}:${opt.path}`}
                                      className="py-2"
                                    >
                                      <div className="flex items-center gap-2 flex-wrap w-full">
                                        {opt.type === "artifacts" && (
                                          <Folder className="h-3 w-3" />
                                        )}
                                        {opt.type === "file-uploads" && (
                                          <CloudUpload className="h-3 w-3" />
                                        )}
                                        {opt.type === "repo" && (
                                          <GitBranch className="h-3 w-3" />
                                        )}
                                        {opt.type === "workflow" && (
                                          <Sparkles className="h-3 w-3" />
                                        )}
                                        <span className="text-xs">
                                          {opt.name}
                                        </span>
                                        {branchName && (
                                          <Badge variant="outline" className="text-xs px-1.5 py-0.5 max-w-full !whitespace-normal !overflow-visible break-words bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
                                            {branchName}
                                          </Badge>
                                        )}
                                      </div>
                                    </SelectItem>
                                  );
                                })}
                              </SelectContent>
                            </Select>
                          </div>

                          {/* File Browser */}
                          <div className="overflow-hidden">
                            <div className="px-2 py-1.5 border-y flex items-center justify-between bg-muted/30">
                              <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
                                {(fileOps.currentSubPath ||
                                  fileOps.viewingFile) && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={fileOps.navigateBack}
                                    className="h-6 px-1.5 mr-1"
                                  >
                                    ← Back
                                  </Button>
                                )}

                                <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
                                <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
                                  {selectedDirectory.path}
                                  {fileOps.currentSubPath &&
                                    `/${fileOps.currentSubPath}`}
                                  {fileOps.viewingFile &&
                                    `/${fileOps.viewingFile.path}`}
                                </code>
                              </div>

                              {fileOps.viewingFile ? (
                                <div className="flex items-center gap-1">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={fileOps.handleDownloadFile}
                                    className="h-6 px-2 flex-shrink-0"
                                    title="Download file"
                                  >
                                    <Download className="h-3 w-3" />
                                  </Button>
                                  <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        className="h-6 px-2 flex-shrink-0"
                                      >
                                        <MoreVertical className="h-3 w-3" />
                                      </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end">
                                      <DropdownMenuItem
                                        disabled
                                        className="text-xs text-muted-foreground"
                                      >
                                        Sync to Jira - Coming soon
                                      </DropdownMenuItem>
                                      <DropdownMenuItem
                                        disabled
                                        className="text-xs text-muted-foreground"
                                      >
                                        Sync to GDrive - Coming soon
                                      </DropdownMenuItem>
                                    </DropdownMenuContent>
                                  </DropdownMenu>
                                </div>
                              ) : (
                                <div className="flex items-center gap-1">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => setUploadModalOpen(true)}
                                    className="h-6 px-2 flex-shrink-0"
                                    title="Upload files"
                                  >
                                    <CloudUpload className="h-3 w-3" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => refetchDirectoryFiles()}
                                    className="h-6 px-2 flex-shrink-0"
                                    title="Refresh"
                                  >
                                    <FolderSync className="h-3 w-3" />
                                  </Button>
                                </div>
                              )}
                            </div>

                            <div className="p-2 max-h-64 overflow-y-auto">
                              {fileOps.loadingFile ? (
                                <div className="flex items-center justify-center py-8">
                                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                                </div>
                              ) : fileOps.viewingFile ? (
                                <div className="text-xs">
                                  <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
                                    <code>{fileOps.viewingFile.content}</code>
                                  </pre>
                                </div>
                              ) : directoryFiles.length === 0 ? (
                                <div className="text-center py-4 text-sm text-muted-foreground">
                                  <FolderTree className="h-8 w-8 mx-auto mb-2 opacity-30" />
                                  <p>No files yet</p>
                                  <p className="text-xs mt-1">
                                    Files will appear here
                                  </p>
                                </div>
                              ) : (
                                <FileTree
                                  nodes={directoryFiles.map(
                                    (item): FileTreeNode => {
                                      const node: FileTreeNode = {
                                        name: item.name,
                                        path: item.path,
                                        type: item.isDir ? "folder" : "file",
                                        sizeKb: item.size
                                          ? item.size / 1024
                                          : undefined,
                                      };

                                      // Don't add branch badges to individual files/folders
                                      // The branch is already shown in the directory selector dropdown

                                      return node;
                                    },
                                  )}
                                  onSelect={fileOps.handleFileOrFolderSelect}
                                />
                              )}
                            </div>
                          </div>

                          {/* Simplified Git Status Display */}
                          <div className="space-y-2">
                            {/* GitHub Not Configured Warning */}
                            {!githubConfigured && (
                              <Alert variant="default" className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/50">
                                <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-500" />
                                <AlertTitle className="text-amber-900 dark:text-amber-100">GitHub Not Configured</AlertTitle>
                                <AlertDescription className="text-amber-800 dark:text-amber-200">
                                  Configure GitHub integration in{" "}
                                  <a 
                                    href={`/projects/${projectName}?section=settings`}
                                    className="underline font-medium hover:text-amber-900 dark:hover:text-amber-100"
                                    onClick={(e) => e.stopPropagation()}
                                  >
                                    workspace settings
                                  </a>
                                  {" "}to enable git operations.
                                </AlertDescription>
                              </Alert>
                            )}

                            {/* State 1: No Git Initialized */}
                            {!gitOps.gitStatus?.initialized ? (
                              <div className="text-sm text-muted-foreground py-2">
                                <p>No git repository. Ask the agent to initialize git if needed.</p>
                              </div>
                            ) : !hasRemote ? (
                              /* State 2: Has Git, No Remote */
                              <div className="space-y-2">
                                <div className="border rounded-md px-2 py-1.5 text-xs">
                                  <div className="flex items-center gap-1.5 text-muted-foreground">
                                    <GitBranch className="h-3 w-3" />
                                    <span>{currentBranch}</span>
                                    <span className="text-muted-foreground/50">(local only)</span>
                                  </div>
                                </div>
                                <Button
                                  onClick={() => setRemoteDialogOpen(true)}
                                  size="sm"
                                  variant="outline"
                                  className="w-full"
                                  disabled={!githubConfigured}
                                >
                                  <Cloud className="mr-2 h-3 w-3" />
                                  Configure Remote
                                </Button>
                              </div>
                            ) : (
                              /* State 3: Has Git + Remote */
                              <div className="border rounded-md px-2 py-1.5 space-y-1">
                                {/* Remote Repository */}
                                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                  <Cloud className="h-3 w-3 flex-shrink-0" />
                                  <span className="truncate">
                                    {remoteUrl
                                      ?.split("/")
                                      .slice(-2)
                                      .join("/")
                                      .replace(".git", "") || ""}
                                  </span>
                                </div>

                                {/* Branch Tracking */}
                                <div className="flex items-center gap-1.5 text-xs">
                                  <GitBranch className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
                                  <span className="text-muted-foreground">
                                    {currentBranch}
                                  </span>
                                </div>
                              </div>
                            )}
                          </div>
                        </div>
                      </AccordionContent>
                    </AccordionItem>
                  </Accordion>
                </div>
              </div>
              )}

            {/* Floating show button when left panel is hidden (desktop only) */}
            {!leftPanelVisible && !mobileMenuOpen && (
                                      <Button
                variant="outline"
                                        size="sm"
                className="hidden md:flex fixed left-2 top-1/2 -translate-y-1/2 z-30 h-8 w-8 p-0 rounded-full shadow-md"
                onClick={() => setLeftPanelVisible(true)}
                title="Show left panel"
              >
                <ChevronRight className="h-4 w-4" />
                                      </Button>
            )}

            {/* Desktop resizable panels */}
            <ResizablePanelGroup direction="horizontal" autoSaveId="session-layout" className="hidden md:flex h-full">
              {leftPanelVisible && session && (
                <>
                  <ResizablePanel
                    id="left-panel"
                    order={1}
                    defaultSize={30}
                    minSize={20}
                    maxSize={50}
                  >
                    <div className={cn(
                      "flex flex-col h-[calc(100vh-8rem)] pt-6 px-6 bg-card relative",
                      phase !== "Running" && "pointer-events-none"
                    )}>
                      {/* Backdrop blur layer for entire sidebar */}
                      {phase !== "Running" && (
                        <div className={cn(
                          "absolute inset-0 z-[5] backdrop-blur-[1px]",
                          ["Creating", "Pending", "Stopping"].includes(phase) && "bg-background/40",
                          ["Stopped", "Completed", "Failed"].includes(phase) && "bg-background/50 backdrop-blur-[2px]"
                        )} />
                      )}

                      {/* State overlay for non-running sessions */}
                      {phase !== "Running" && (
                        <div className="absolute inset-0 z-10 flex items-center justify-center pointer-events-auto">
                          <div className="text-center">
                            {["Creating", "Pending"].includes(phase) && (
                              <>
                                <Loader2 className="h-10 w-10 mx-auto mb-3 animate-spin text-blue-600" />
                                <h3 className="font-semibold text-lg mb-1">Starting Session</h3>
                                <p className="text-sm text-muted-foreground">Setting up your workspace...</p>
                              </>
                            )}
                            {phase === "Stopping" && (
                              <>
                                <Loader2 className="h-10 w-10 mx-auto mb-3 animate-spin text-orange-600" />
                                <h3 className="font-semibold text-lg mb-1">Stopping Session</h3>
                                <p className="text-sm text-muted-foreground">Saving workspace state...</p>
                              </>
                            )}
                            {["Stopped", "Completed", "Failed"].includes(phase) && (
                              <div className="max-w-sm">
                                <h3 className="font-semibold text-lg mb-4">Session Hibernated</h3>
                                <Button onClick={handleContinue} size="lg" className="w-full" disabled={continueMutation.isPending}>
                                  {continueMutation.isPending ? (
                                    <>
                                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                      Resuming...
                                    </>
                                  ) : (
                                    'Resume Session'
                                  )}
                                    </Button>
                            </div>
                          )}
                        </div>
                        </div>
                      )}

                      <div className={cn(
                        "flex-grow pb-6 overflow-y-auto scrollbar-hide",
                        ["Stopped", "Completed", "Failed"].includes(phase) && "blur-[2px]"
                      )}>
                        <Accordion
                          type="multiple"
                          value={openAccordionItems}
                          onValueChange={phase === "Running" ? setOpenAccordionItems : undefined}
                          className="w-full space-y-3"
                        >
                          <WorkflowsAccordion
                            sessionPhase={session?.status?.phase}
                            activeWorkflow={workflowManagement.activeWorkflow}
                            selectedWorkflow={workflowManagement.selectedWorkflow}
                            workflowActivating={workflowManagement.workflowActivating}
                            ootbWorkflows={ootbWorkflows}
                            isExpanded={openAccordionItems.includes("workflows")}
                            onWorkflowChange={handleWorkflowChange}
                            onResume={handleContinue}
                          />
                          <RepositoriesAccordion
                            repositories={reposStatus?.repos || session?.status?.reconciledRepos || session?.spec?.repos || []}
                            uploadedFiles={fileUploadsList.map((f) => ({ name: f.name, path: f.path, size: f.size }))}
                            onAddRepository={() => setContextModalOpen(true)}
                            onRemoveRepository={(repoName) => removeRepoMutation.mutate(repoName)}
                            onRemoveFile={(fileName) => removeFileMutation.mutate(fileName)}
                          />
                          <ArtifactsAccordion
                            files={artifactsFiles}
                            currentSubPath={artifactsOps.currentSubPath}
                            viewingFile={artifactsOps.viewingFile}
                            isLoadingFile={artifactsOps.loadingFile}
                            onFileOrFolderSelect={artifactsOps.handleFileOrFolderSelect}
                            onRefresh={refetchArtifactsFiles}
                            onDownloadFile={artifactsOps.handleDownloadFile}
                            onNavigateBack={artifactsOps.navigateBack}
                          />
                          <McpServersAccordion
                            projectName={projectName}
                            sessionName={sessionName}
                            sessionPhase={phase}
                          />
                          <IntegrationsAccordion />

                          {/* File Explorer */}
                          <AccordionItem
                            value="file-explorer"
                            className="border rounded-lg px-3 bg-card"
                          >
                            <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                              <div className="flex items-center gap-2 w-full">
                                <Folder className="h-4 w-4" />
                                <span>File Explorer</span>
                                <Badge
                                  variant="outline"
                                  className="text-[10px] px-2 py-0.5"
                                >
                                  EXPERIMENTAL
                                </Badge>
                                {gitOps.gitStatus?.hasChanges && (
                                  <div className="flex gap-1 ml-auto mr-2">
                                    {(gitOps.gitStatus?.totalAdded ?? 0) > 0 && (
                                      <Badge
                                        variant="outline"
                                        className="bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800"
                                      >
                                        +{gitOps.gitStatus.totalAdded}
                                      </Badge>
                                    )}
                                    {(gitOps.gitStatus?.totalRemoved ?? 0) > 0 && (
                                      <Badge
                                        variant="outline"
                                        className="bg-red-50 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-300 dark:border-red-800"
                                      >
                                        -{gitOps.gitStatus.totalRemoved}
                                      </Badge>
                                    )}
                                  </div>
                                )}
                              </div>
                            </AccordionTrigger>
                            <AccordionContent className="pt-2 pb-3">
                              <div className="space-y-3">
                                <p className="text-sm text-muted-foreground">
                                  Browse, view, and manage files in your workspace
                                  directories. Track changes and sync with Git for
                                  version control.
                                </p>

                                {/* Directory Selector */}
                                <div className="flex items-center justify-between gap-2">
                                  <Label className="text-xs text-muted-foreground">
                                    Directory:
                                  </Label>
                                  <Select
                                    value={`${selectedDirectory.type}:${selectedDirectory.path}`}
                                    onValueChange={(value) => {
                                      const [type, ...pathParts] = value.split(":");
                                      const path = pathParts.join(":");
                                      const option = directoryOptions.find(
                                        (opt) =>
                                          opt.type === type && opt.path === path,
                                      );
                                      if (option) setSelectedDirectory(option);
                                    }}
                                  >
                                    <SelectTrigger className="w-[300px] h-auto min-h-[2.5rem] py-2.5 overflow-visible">
                                      <div className="flex items-center gap-2 flex-wrap w-full pr-6 overflow-visible">
                                        <SelectValue />
                                      </div>
                                    </SelectTrigger>
                                    <SelectContent>
                                      {directoryOptions.map((opt) => {
                                        // Find branch info for repo directories from real-time status
                                        let branchName: string | undefined;
                                        if (opt.type === "repo") {
                                          const repoName = opt.path.replace(/^repos\//, "");
                                          const realtimeRepo = reposStatus?.repos?.find(
                                            (r) => r.name === repoName
                                          );
                                          const reconciledRepo = session?.status?.reconciledRepos?.find(
                                            (r: ReconciledRepo) => {
                                              const rName = r.name || r.url?.split("/").pop()?.replace(".git", "");
                                              return rName === repoName;
                                            }
                                          );
                                          branchName = realtimeRepo?.currentActiveBranch
                                            || reconciledRepo?.currentActiveBranch
                                            || reconciledRepo?.branch;
                                        }

                                        return (
                                          <SelectItem
                                            key={`${opt.type}:${opt.path}`}
                                            value={`${opt.type}:${opt.path}`}
                                            className="py-2"
                                          >
                                            <div className="flex items-center gap-2 flex-wrap w-full">
                                              {opt.type === "artifacts" && (
                                                <Folder className="h-3 w-3" />
                                              )}
                                              {opt.type === "file-uploads" && (
                                                <CloudUpload className="h-3 w-3" />
                                              )}
                                              {opt.type === "repo" && (
                                                <GitBranch className="h-3 w-3" />
                                              )}
                                              {opt.type === "workflow" && (
                                                <Sparkles className="h-3 w-3" />
                                              )}
                                              <span className="text-xs">
                                                {opt.name}
                                              </span>
                                              {branchName && (
                                                <Badge variant="outline" className="text-xs px-1.5 py-0.5 max-w-full !whitespace-normal !overflow-visible break-words bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
                                                  {branchName}
                                                </Badge>
                                              )}
                                            </div>
                                          </SelectItem>
                                        );
                                      })}
                                    </SelectContent>
                                  </Select>
                                </div>

                                {/* File Browser */}
                                <div className="overflow-hidden">
                                  <div className="px-2 py-1.5 border-y flex items-center justify-between bg-muted/30">
                                    <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
                                      {(fileOps.currentSubPath ||
                                        fileOps.viewingFile) && (
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          onClick={fileOps.navigateBack}
                                          className="h-6 px-1.5 mr-1"
                                        >
                                          ← Back
                                        </Button>
                                      )}

                                      <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
                                      <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
                                        {selectedDirectory.path}
                                        {fileOps.currentSubPath &&
                                          `/${fileOps.currentSubPath}`}
                                        {fileOps.viewingFile &&
                                          `/${fileOps.viewingFile.path}`}
                                      </code>
                                    </div>

                                    {fileOps.viewingFile ? (
                                      <div className="flex items-center gap-1">
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          onClick={fileOps.handleDownloadFile}
                                          className="h-6 px-2 flex-shrink-0"
                                          title="Download file"
                                        >
                                          <Download className="h-3 w-3" />
                                        </Button>
                                        <DropdownMenu>
                                          <DropdownMenuTrigger asChild>
                                            <Button
                                              variant="ghost"
                                              size="sm"
                                              className="h-6 px-2 flex-shrink-0"
                                            >
                                              <MoreVertical className="h-3 w-3" />
                                            </Button>
                                          </DropdownMenuTrigger>
                                          <DropdownMenuContent align="end">
                                            <DropdownMenuItem
                                              disabled
                                              className="text-xs text-muted-foreground"
                                            >
                                              Sync to Jira - Coming soon
                                            </DropdownMenuItem>
                                            <DropdownMenuItem
                                              disabled
                                              className="text-xs text-muted-foreground"
                                            >
                                              Sync to GDrive - Coming soon
                                            </DropdownMenuItem>
                                          </DropdownMenuContent>
                                        </DropdownMenu>
                                      </div>
                                    ) : (
                                      <div className="flex items-center gap-1">
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          onClick={() => setUploadModalOpen(true)}
                                          className="h-6 px-2 flex-shrink-0"
                                          title="Upload files"
                                        >
                                          <CloudUpload className="h-3 w-3" />
                                        </Button>
                                        <Button
                                          variant="ghost"
                                          size="sm"
                                          onClick={() => refetchDirectoryFiles()}
                                          className="h-6 px-2 flex-shrink-0"
                                          title="Refresh"
                                        >
                                          <FolderSync className="h-3 w-3" />
                                        </Button>
                                      </div>
                                    )}
                                  </div>

                                  <div className="p-2 max-h-64 overflow-y-auto">
                                    {fileOps.loadingFile ? (
                                      <div className="flex items-center justify-center py-8">
                                        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                                      </div>
                                    ) : fileOps.viewingFile ? (
                                      <div className="text-xs">
                                        <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
                                          <code>{fileOps.viewingFile.content}</code>
                                        </pre>
                                      </div>
                                    ) : directoryFiles.length === 0 ? (
                                      <div className="text-center py-4 text-sm text-muted-foreground">
                                        <FolderTree className="h-8 w-8 mx-auto mb-2 opacity-30" />
                                        <p>No files yet</p>
                                        <p className="text-xs mt-1">
                                          Files will appear here
                                        </p>
                                      </div>
                                    ) : (
                                      <FileTree
                                        nodes={directoryFiles.map(
                                          (item): FileTreeNode => {
                                            const node: FileTreeNode = {
                                              name: item.name,
                                              path: item.path,
                                              type: item.isDir ? "folder" : "file",
                                              sizeKb: item.size
                                                ? item.size / 1024
                                                : undefined,
                                            };
                                            return node;
                                          },
                                        )}
                                        onSelect={fileOps.handleFileOrFolderSelect}
                                      />
                                    )}
                                  </div>
                                </div>

                                {/* Simplified Git Status Display */}
                                <div className="space-y-2">
                                  {/* GitHub Not Configured Warning */}
                                  {!githubConfigured && (
                                    <Alert variant="default" className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/50">
                                      <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-500" />
                                      <AlertTitle className="text-amber-900 dark:text-amber-100">GitHub Not Configured</AlertTitle>
                                      <AlertDescription className="text-amber-800 dark:text-amber-200">
                                        Configure GitHub integration in{" "}
                                        <a
                                          href={`/projects/${projectName}?section=settings`}
                                          className="underline font-medium hover:text-amber-900 dark:hover:text-amber-100"
                                          onClick={(e) => e.stopPropagation()}
                                        >
                                          workspace settings
                                        </a>
                                        {" "}to enable git operations.
                                      </AlertDescription>
                                    </Alert>
                                  )}

                                  {/* State 1: No Git Initialized */}
                                  {!gitOps.gitStatus?.initialized ? (
                                    <div className="text-sm text-muted-foreground py-2">
                                      <p>No git repository. Ask the agent to initialize git if needed.</p>
                                    </div>
                                  ) : !hasRemote ? (
                                    /* State 2: Has Git, No Remote */
                                    <div className="space-y-2">
                                      <div className="border rounded-md px-2 py-1.5 text-xs">
                                        <div className="flex items-center gap-1.5 text-muted-foreground">
                                          <GitBranch className="h-3 w-3" />
                                          <span>{currentBranch}</span>
                                          <span className="text-muted-foreground/50">(local only)</span>
                                        </div>
                                      </div>
                                      <Button
                                        onClick={() => setRemoteDialogOpen(true)}
                                        size="sm"
                                        variant="outline"
                                        className="w-full"
                                        disabled={!githubConfigured}
                                      >
                                        <Cloud className="mr-2 h-3 w-3" />
                                        Configure Remote
                                      </Button>
                                    </div>
                                  ) : (
                                    /* State 3: Has Git + Remote */
                                    <div className="border rounded-md px-2 py-1.5 space-y-1">
                                      {/* Remote Repository */}
                                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                        <Cloud className="h-3 w-3 flex-shrink-0" />
                                        <span className="truncate">
                                          {remoteUrl
                                            ?.split("/")
                                            .slice(-2)
                                            .join("/")
                                            .replace(".git", "") || ""}
                                        </span>
                                      </div>

                                      {/* Branch Tracking */}
                                      <div className="flex items-center gap-1.5 text-xs">
                                        <GitBranch className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
                                        <span className="text-muted-foreground">
                                          {currentBranch}
                                        </span>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              </div>
                            </AccordionContent>
                          </AccordionItem>
                  </Accordion>
                </div>

                      {/* Hide panel button */}
                      <div className="pt-2 pb-3 flex justify-center border-t">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setLeftPanelVisible(false)}
                          className="text-muted-foreground hover:text-foreground"
                  >
                          <ChevronLeft className="h-4 w-4 mr-1" />
                    <span className="text-xs">Hide panel</span>
                  </Button>
                </div>
              </div>
                  </ResizablePanel>
                  <ResizableHandle className="w-1 hover:bg-primary/50 transition-colors" />
                  </>
                )}
              <ResizablePanel id="right-panel" order={2} defaultSize={leftPanelVisible ? 70 : 100} minSize={50}>
                  <div className="flex-1 min-w-0 flex flex-col h-full">
                <Card className="relative flex-1 flex flex-col overflow-hidden py-0 border-0 rounded-none md:border-l">
                  <CardContent className="px-3 pt-0 pb-0 flex-1 flex flex-col overflow-hidden">
                    {/* Repository change overlay */}
                    {repoChanging && (
                      <div className="absolute inset-0 bg-background/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
                        <Alert className="max-w-md mx-4">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          <AlertTitle>Updating Repositories...</AlertTitle>
                          <AlertDescription>
                            <div className="space-y-2">
                              <p>
                                Please wait while repositories are being
                                updated. This may take 10-20 seconds...
                              </p>
                            </div>
                          </AlertDescription>
                        </Alert>
                      </div>
                    )}

                    <div className="flex flex-col flex-1 overflow-hidden">
                      {/* Phase-aware chat area */}
                      {phase === "Running" || ["Stopped", "Completed"].includes(phase) ? (
                        <CopilotChatView
                          projectName={projectName}
                          sessionName={sessionName}
                          className="flex-1"
                          isSessionActive={phase === "Running"}
                          workflowMetadata={workflowMetadata}
                          onResume={phase !== "Running" ? handleContinue : undefined}
                          isResuming={continueMutation.isPending}
                          inputEnhancements={inputEnhancements}
                          renderWelcome={(hasMessages) => (
                            <WelcomeExperience
                              ootbWorkflows={ootbWorkflows}
                              onWorkflowSelect={handleWelcomeWorkflowSelect}
                              onUserInteraction={() => setUserHasInteracted(true)}
                              userHasInteracted={userHasInteracted}
                              sessionPhase={phase}
                              hasRealMessages={hasMessages}
                              onLoadWorkflow={() => setCustomWorkflowDialogOpen(true)}
                              selectedWorkflow={workflowManagement.selectedWorkflow}
                            />
                          )}
                        />
                      ) : (
                        <SessionPhaseOverlay
                          phase={phase}
                          session={session}
                          projectName={projectName}
                          sessionName={sessionName}
                          onResume={handleContinue}
                          isResuming={continueMutation.isPending}
                        />
                      )}
                    </div>
                  </CardContent>
                </Card>
                  </div>
              </ResizablePanel>
            </ResizablePanelGroup>

            {/* Mobile right column */}
            <div className="md:hidden h-full flex flex-col">
                <Card className="relative flex-1 flex flex-col overflow-hidden py-0 border-0 rounded-none">
                  <CardContent className="px-3 pt-0 pb-0 flex-1 flex flex-col overflow-hidden">
                    {repoChanging && (
                      <div className="absolute inset-0 bg-background/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
                        <Alert className="max-w-md mx-4">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          <AlertTitle>Updating Repositories...</AlertTitle>
                          <AlertDescription>
                            <div className="space-y-2">
                            <p>Please wait while repositories are being updated. This may take 10-20 seconds...</p>
                            </div>
                          </AlertDescription>
                        </Alert>
                      </div>
                    )}
                    <div className="flex flex-col flex-1 overflow-hidden">
                      {/* Phase-aware chat area — mobile */}
                      {phase === "Running" || ["Stopped", "Completed"].includes(phase) ? (
                        <CopilotChatView
                          projectName={projectName}
                          sessionName={sessionName}
                          className="flex-1"
                          isSessionActive={phase === "Running"}
                          workflowMetadata={workflowMetadata}
                          onResume={phase !== "Running" ? handleContinue : undefined}
                          isResuming={continueMutation.isPending}
                          inputEnhancements={inputEnhancements}
                          renderWelcome={(hasMessages) => (
                            <WelcomeExperience
                              ootbWorkflows={ootbWorkflows}
                              onWorkflowSelect={handleWelcomeWorkflowSelect}
                              onUserInteraction={() => setUserHasInteracted(true)}
                              userHasInteracted={userHasInteracted}
                              sessionPhase={phase}
                              hasRealMessages={hasMessages}
                              onLoadWorkflow={() => setCustomWorkflowDialogOpen(true)}
                              selectedWorkflow={workflowManagement.selectedWorkflow}
                            />
                          )}
                        />
                      ) : (
                        <SessionPhaseOverlay
                          phase={phase}
                          session={session}
                          projectName={projectName}
                          sessionName={sessionName}
                          onResume={handleContinue}
                          isResuming={continueMutation.isPending}
                        />
                      )}
                    </div>
                  </CardContent>
                </Card>
            </div>
          </div>
        </div>
        </CopilotSessionProvider>
      </div>

      {/* Modals */}
      <AddContextModal
        open={contextModalOpen}
        onOpenChange={setContextModalOpen}
        onAddRepository={async (url, branch, autoPush) => {
          await addRepoMutation.mutateAsync({ url, branch, autoPush });
          setContextModalOpen(false);
        }}
        onUploadFile={() => setUploadModalOpen(true)}
        isLoading={addRepoMutation.isPending}
        autoBranch={session?.autoBranch}
      />

      <UploadFileModal
        open={uploadModalOpen}
        onOpenChange={setUploadModalOpen}
        onUploadFile={async (source) => {
          await uploadFileMutation.mutateAsync(source);
        }}
        isLoading={uploadFileMutation.isPending}
      />

      <CustomWorkflowDialog
        open={customWorkflowDialogOpen}
        onOpenChange={setCustomWorkflowDialogOpen}
        onSubmit={(url, branch, path) => {
          workflowManagement.setCustomWorkflow(url, branch, path);
          setCustomWorkflowDialogOpen(false);
          // Automatically activate the custom workflow (same as OOTB workflows)
          const customWorkflow = {
            id: "custom",
            name: "Custom workflow",
            description: `Custom workflow from ${url}`,
            gitUrl: url,
            branch: branch || "main",
            path: path || "",
            enabled: true,
          };
          workflowManagement.activateWorkflow(customWorkflow, session?.status?.phase);
        }}
        isActivating={workflowManagement.workflowActivating}
      />

      <ManageRemoteDialog
        open={remoteDialogOpen}
        onOpenChange={setRemoteDialogOpen}
        onSave={async (url, branch) => {
          const success = await gitOps.configureRemote(url, branch || "main");
          if (success) {
            const newRemotes = { ...directoryRemotes };
            newRemotes[selectedDirectory.path] = { url, branch: branch || "main" };
            setDirectoryRemotes(newRemotes);
            setRemoteDialogOpen(false);
          }
        }}
        directoryName={selectedDirectory.name}
        currentUrl={currentRemote?.url}
        currentBranch={currentRemote?.branch}
        isLoading={gitOps.isConfiguringRemote}
      />
    </>
  );
}
