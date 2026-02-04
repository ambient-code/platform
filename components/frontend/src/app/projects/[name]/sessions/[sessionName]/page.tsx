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
} from "lucide-react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

// Custom components
import MessagesTab from "@/components/session/MessagesTab";
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
import { McpIntegrationsAccordion } from "./components/accordions/mcp-integrations-accordion";
import { WelcomeExperience } from "./components/welcome-experience";
// Extracted hooks and utilities
import { useGitOperations } from "./hooks/use-git-operations";
import { useWorkflowManagement } from "./hooks/use-workflow-management";
import { useFileOperations } from "./hooks/use-file-operations";
import { useSessionQueue } from "@/hooks/use-session-queue";
import type { DirectoryOption, DirectoryRemote } from "./lib/types";

import type { MessageObject, ToolUseMessages, HierarchicalToolMessage, ReconciledRepo, SessionRepo } from "@/types/agentic-session";
import type { AGUIToolCall } from "@/types/agui";

// AG-UI streaming
import { useAGUIStream } from "@/hooks/use-agui-stream";

// React Query hooks
import {
  useSession,
  useStopSession,
  useDeleteSession,
  useContinueSession,
  useReposStatus,
  useCurrentUser,
} from "@/services/queries";
import {
  useWorkspaceList,
} from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";
import {
  useOOTBWorkflows,
  useWorkflowMetadata,
} from "@/services/queries/use-workflows";
import { useProjectIntegrationStatus } from "@/services/queries/use-projects";
import { useMutation } from "@tanstack/react-query";
import { FeedbackProvider } from "@/contexts/FeedbackContext";

// Constants for artifact auto-refresh timing
// Moved outside component to avoid unnecessary effect re-runs
//
// Wait 1 second after last tool completion to batch rapid writes together
// Prevents excessive API calls during burst writes (e.g., when Claude creates multiple files in quick succession)
// Testing: 500ms was too aggressive (hit API rate limits), 2000ms felt sluggish to users
const ARTIFACTS_DEBOUNCE_MS = 1000;

// Wait 2 seconds after session completes before final artifact refresh
// Backend can take 1-2 seconds to flush final artifacts to storage
// Ensures users see all artifacts even if final writes occur after status transition
const COMPLETION_DELAY_MS = 2000;

/**
 * Type guard to check if a message is a completed ToolUseMessages with result.
 * Extracted for testability and proper validation.
 * Uses proper type assertion and validation.
 */
function isCompletedToolUseMessage(msg: MessageObject | ToolUseMessages): msg is ToolUseMessages {
  if (msg.type !== "tool_use_messages") {
    return false;
  }
  
  // Cast to ToolUseMessages for proper type checking
  const toolMsg = msg as ToolUseMessages;
  
  return (
    toolMsg.resultBlock !== undefined &&
    toolMsg.resultBlock !== null &&
    typeof toolMsg.resultBlock === "object" &&
    toolMsg.resultBlock.content !== null
  );
}

export default function ProjectSessionDetailPage({
  params,
}: {
  params: Promise<{ name: string; sessionName: string }>;
}) {
  const router = useRouter();
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");
  const [chatInput, setChatInput] = useState("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>([]);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [repoChanging, setRepoChanging] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [userHasInteracted, setUserHasInteracted] = useState(false);

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
  const { data: integrationStatus } = useProjectIntegrationStatus(projectName);
  const githubConfigured = integrationStatus?.github ?? false;
  
  // Get current user for feedback context
  const { data: currentUser } = useCurrentUser();

  // Extract phase for sidebar state management
  const phase = session?.status?.phase || "Pending";

  // Fetch repos status directly from runner (real-time branch info)
  const { data: reposStatus } = useReposStatus(
    projectName,
    sessionName,
    phase === "Running" // Only poll when session is running
  );

  // Track the current Langfuse trace ID for feedback association
  const [langfuseTraceId, setLangfuseTraceId] = useState<string | null>(null);
  
  // AG-UI streaming hook - replaces useSessionMessages and useSendChatMessage
  // Note: autoConnect is intentionally false to avoid SSR hydration mismatch
  // Connection is triggered manually in useEffect after client hydration
  const aguiStream = useAGUIStream({
    projectName: projectName || "",
    sessionName: sessionName || "",
    autoConnect: false, // Manual connection after hydration
    onError: (err) => console.error("AG-UI stream error:", err),
    onTraceId: (traceId) => setLangfuseTraceId(traceId),  // Capture Langfuse trace ID for feedback
  });
  const aguiState = aguiStream.state;
  const aguiSendMessage = aguiStream.sendMessage;
  const aguiInterrupt = aguiStream.interrupt;
  const isRunActive = aguiStream.isRunActive;
  const aguiConnectRef = useRef(aguiStream.connect);
  
  // Keep connect ref up to date
  useEffect(() => {
    aguiConnectRef.current = aguiStream.connect;
  }, [aguiStream.connect]);

  // Connect to AG-UI event stream for history and live updates
  // AG-UI pattern: GET /agui/events streams ALL thread events (past + future)
  // POST /agui/run creates runs, events broadcast to GET subscribers
  const hasConnectedRef = useRef(false);
  const disconnectRef = useRef(aguiStream.disconnect);
  
  // Keep disconnect ref up to date without triggering re-renders
  useEffect(() => {
    disconnectRef.current = aguiStream.disconnect;
  }, [aguiStream.disconnect]);
  
  useEffect(() => {
    if (!projectName || !sessionName) return;
    
    // Connect once on mount and keep connection open
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true;
      aguiConnectRef.current();
    }
    
    // CRITICAL: Disconnect when navigating away to prevent hung connections
    return () => {
      console.log('[Session Detail] Unmounting, disconnecting AG-UI stream');
      disconnectRef.current();
      hasConnectedRef.current = false;
    };
    // NOTE: Only depend on projectName and sessionName - NOT aguiStream
    // aguiStream is an object that changes every render, which would cause infinite reconnects
  }, [projectName, sessionName]);

  // Auto-send initial prompt (handles session start, workflow activation, restarts)
  // AG-UI pattern: Client (or backend) initiates runs via POST /agui/run
  const lastProcessedPromptRef = useRef<string>("");
  
  useEffect(() => {
    if (!session || !aguiSendMessage) return;
    
    const initialPrompt = session?.spec?.initialPrompt;
    
    // NOTE: Initial prompt execution handled by backend auto-trigger (StartSession handler)
    // Backend waits for subscriber before executing, ensuring events are received
    // This works for both UI and headless/API usage
    
    // Track that we've seen this prompt (for workflow changes)
    if (initialPrompt && lastProcessedPromptRef.current !== initialPrompt) {
      lastProcessedPromptRef.current = initialPrompt;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.spec?.initialPrompt, session?.status?.phase, aguiState.messages.length, aguiState.status]);

  // Workflow management hook
  const workflowManagement = useWorkflowManagement({
    projectName,
    sessionName,
    sessionPhase: session?.status?.phase,
    onWorkflowActivated: refetchSession,
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

  // Process queued messages when session becomes Running
  useEffect(() => {
    const phase = session?.status?.phase;
    const unsentMessages = sessionQueue.messages.filter(m => !m.sentAt);
    
    if (phase === "Running" && unsentMessages.length > 0) {
      // Session is now running, send all queued messages
      const processMessages = async () => {
        for (const messageItem of unsentMessages) {
          try {
            await aguiSendMessage(messageItem.content);
            sessionQueue.markMessageSent(messageItem.id);
            // Small delay between messages to avoid overwhelming the system
            await new Promise(resolve => setTimeout(resolve, 100));
          } catch (err) {
            errorToast(err instanceof Error ? err.message : "Failed to send queued message");
          }
        }
      };
      
      processMessages();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.status?.phase, sessionQueue.messages.length]);

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
  // 2. User sends a message (sendChat sets it to true)
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

  // Convert AG-UI messages to display format with hierarchical tool call rendering
  const streamMessages: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage> = useMemo(() => {
    
    // Helper function to parse tool arguments
    const parseToolArgs = (args: string | undefined): Record<string, unknown> => {
      if (!args) return {};
      try {
        const parsed = JSON.parse(args);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          return parsed as Record<string, unknown>;
        }
        return { value: parsed };
      } catch {
        return { _raw: String(args || '') };
      }
    };

    // Helper function to create a tool message from a tool call
    const createToolMessage = (
      tc: AGUIToolCall,
      timestamp: string
    ): ToolUseMessages => {
      const toolInput = parseToolArgs(tc.args);
      return {
        type: "tool_use_messages",
        timestamp,
        toolUseBlock: {
          type: "tool_use_block",
          id: tc.id,
          name: tc.name,
          input: toolInput,
        },
        resultBlock: {
          type: "tool_result_block",
          tool_use_id: tc.id,
          content: tc.result || null,
          is_error: tc.status === "error",
        },
      };
    };

    const result: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage> = [];
    
    // Phase A: Collect all tool calls from all messages for hierarchy building
    const allToolCalls = new Map<string, { tc: AGUIToolCall; timestamp: string }>();
    
    for (const msg of aguiState.messages) {
      // Use msg.timestamp from backend, fallback to current time for legacy messages
      // Note: After backend fix, new messages will have proper timestamps
      const timestamp = msg.timestamp || new Date().toISOString();
      
      if (msg.toolCalls && Array.isArray(msg.toolCalls)) {
        for (const tc of msg.toolCalls) {
          if (tc && tc.id && tc.name) {
            allToolCalls.set(tc.id, { tc, timestamp });
          }
        }
      }
    }
    
    // Add currently streaming tool call to the map if present
    // This ensures streaming tools (both parents and children) are included in hierarchy
    // CRITICAL: Don't require name - add even if name is null to prevent orphaned children
    if (aguiState.currentToolCall?.id) {
      const streamingToolId = aguiState.currentToolCall.id;
      const streamingParentId = aguiState.currentToolCall.parentToolUseId;
      const toolName = aguiState.currentToolCall.name || "unknown_tool";  // Default if null
      
      // Create a pseudo-tool-call for the streaming tool
      const streamingTC: AGUIToolCall = {
        id: streamingToolId,
        name: toolName,
        args: aguiState.currentToolCall.args || "",
        type: "function",
        parentToolUseId: streamingParentId,
        status: "running",
      };
      
      if (!allToolCalls.has(streamingToolId)) {
        allToolCalls.set(streamingToolId, { 
          tc: streamingTC, 
          // Use timestamp from currentToolCall if available, fallback to current time for legacy
          timestamp: aguiState.pendingToolCalls?.get(streamingToolId)?.timestamp || new Date().toISOString() 
        });
      }
    }
    
    // Add pending children to render map so they show during streaming!
    // These are children that finished before their parent tool finished
    if (aguiState.pendingChildren && aguiState.pendingChildren.size > 0) {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      for (const [parentId, children] of aguiState.pendingChildren.entries()) {
        for (const childMsg of children) {
          if (childMsg.toolCalls) {
            for (const tc of childMsg.toolCalls) {
              if (!allToolCalls.has(tc.id)) {
                allToolCalls.set(tc.id, {
                  tc: tc,
                  // Use timestamp from child message if available, fallback to current time
                  timestamp: childMsg.timestamp || new Date().toISOString(),
                });
              }
            }
          }
        }
      }
    }
    
    // Phase B: Build parent-child relationships
    const topLevelTools = new Set<string>();
    const childrenByParent = new Map<string, string[]>();
    
    for (const [toolId, { tc }] of allToolCalls) {
      if (tc.parentToolUseId) {
        // This is a child tool call
        if (!childrenByParent.has(tc.parentToolUseId)) {
          childrenByParent.set(tc.parentToolUseId, []);
        }
        childrenByParent.get(tc.parentToolUseId)!.push(toolId);
      } else {
        // This is a top-level tool call
        topLevelTools.add(toolId);
      }
    }
    
    // Handle orphaned children - but DON'T promote to top-level if parent is streaming
    for (const [toolId, { tc }] of allToolCalls) {
      if (tc.parentToolUseId && !allToolCalls.has(tc.parentToolUseId)) {
        // Check if parent is the currently streaming tool
        if (aguiState.currentToolCall?.id === tc.parentToolUseId) {
          // Don't promote to top-level - parent is streaming and will appear
        } else {
          // Parent truly not found, render as top-level (fallback)
          console.warn(`  ⚠️ Orphaned child: ${tc.name} (${toolId.substring(0, 8)}) - parent ${tc.parentToolUseId.substring(0, 8)} not found`);
          topLevelTools.add(toolId);
        }
      }
    }
    
    // Track which tool calls we've already rendered
    const renderedToolCalls = new Set<string>();
    
    // Phase C: Process messages and build hierarchical structure
    for (const msg of aguiState.messages) {
      // Use msg.timestamp from backend, fallback to current time for legacy messages
      const timestamp = msg.timestamp || new Date().toISOString();
      
      // Handle text content by role
      if (msg.role === "user") {
        result.push({
          type: "user_message",
          id: msg.id,  // Preserve message ID for feedback association
          content: { type: "text_block", text: msg.content || "" },
          timestamp,
        });
      } else if (msg.role === "assistant") {
        // Check if this is a thinking block (from RAW event)
        const metadata = msg.metadata as Record<string, unknown> | undefined;
        if (metadata?.type === "thinking_block") {
          result.push({
            type: "agent_message",
            id: msg.id,  // Preserve message ID for feedback association
            content: {
              type: "thinking_block",
              thinking: metadata.thinking as string || "",
              signature: metadata.signature as string || "",
            },
            model: "claude",
            timestamp,
          });
        } else if (msg.content) {
          // Only push text message if there's actual content
          result.push({
            type: "agent_message",
            id: msg.id,  // Preserve message ID for feedback association
            content: { type: "text_block", text: msg.content },
            model: "claude",
            timestamp,
          });
        }
      } else if (msg.role === "tool") {
        // Standalone tool results (not from toolCalls array)
        if (msg.toolCallId && !allToolCalls.has(msg.toolCallId)) {
          result.push({
            type: "tool_use_messages",
            timestamp,
            toolUseBlock: {
              type: "tool_use_block",
              id: msg.toolCallId,
              name: msg.name || "tool",
              input: {},
            },
            resultBlock: {
              type: "tool_result_block",
              tool_use_id: msg.toolCallId,
              content: msg.content || null,
              is_error: false,
            },
          });
        }
      } else if (msg.role === "system") {
        result.push({
          type: "system_message",
          subtype: "system.message",
          data: { message: msg.content || "" },
          timestamp,
        });
      }
      
      // Handle tool calls embedded in this message
      if (msg.toolCalls && Array.isArray(msg.toolCalls)) {
        for (const tc of msg.toolCalls) {
          if (!tc || !tc.id || !tc.name) continue;
          
          // Skip if already rendered or if it's a child (will be rendered inside parent)
          if (renderedToolCalls.has(tc.id)) {
            continue;
          }
          if (!topLevelTools.has(tc.id)) {
            continue;
          }
          
          // Build children array for this tool call
          const childIds = childrenByParent.get(tc.id) || [];
          
          const children: ToolUseMessages[] = childIds
            .map(childId => {
              const childData = allToolCalls.get(childId);
              if (!childData) return null;
              renderedToolCalls.add(childId);
              return createToolMessage(childData.tc, childData.timestamp);
            })
            .filter((c): c is ToolUseMessages => c !== null);
          
          // Create the hierarchical tool message
          const toolInput = parseToolArgs(tc.args);
          
          const toolMessage: HierarchicalToolMessage = {
            type: "tool_use_messages",
            timestamp,
            toolUseBlock: {
              type: "tool_use_block",
              id: tc.id,
              name: tc.name,
              input: toolInput,
            },
            resultBlock: {
              type: "tool_result_block",
              tool_use_id: tc.id,
              content: tc.result || null,
              is_error: tc.status === "error",
            },
            children: children.length > 0 ? children : undefined,
          };
          
          result.push(toolMessage);
          renderedToolCalls.add(tc.id);
        }
      }
    }
    
    // Add streaming message if currently streaming
    if (aguiState.currentMessage?.content) {
      result.push({
        type: "agent_message",
        content: { type: "text_block", text: aguiState.currentMessage.content },
        model: "claude",
        // Use timestamp from currentMessage (captured from TEXT_MESSAGE_START), fallback to current time
        timestamp: aguiState.currentMessage.timestamp || new Date().toISOString(),
        streaming: true,
      } as MessageObject & { streaming?: boolean });
    }
    
    // Render ALL currently streaming tool calls (supports parallel tool execution)
    // CRITICAL: This renders tools immediately when TOOL_CALL_START arrives,
    // not waiting until TOOL_CALL_END like the allToolCalls map approach does
    const pendingToolCalls = aguiState.pendingToolCalls || new Map();
    
    for (const [toolId, pendingTool] of pendingToolCalls) {
      if (renderedToolCalls.has(toolId)) continue;
      
      const toolName = pendingTool.name || "unknown_tool";
      const toolArgs = pendingTool.args || "";
      const streamingParentId = pendingTool.parentToolUseId;
      
      // Only render if this is a top-level tool (not a child waiting for parent)
      // Children will be rendered nested inside their parent
      const isTopLevel = !streamingParentId || !pendingToolCalls.has(streamingParentId);
      
      if (isTopLevel) {
        const toolInput = parseToolArgs(toolArgs);
        
        // Get any pending children for this tool (children that finished before parent)
        const pendingForThis = aguiState.pendingChildren?.get(toolId) || [];
        const children: ToolUseMessages[] = pendingForThis
          .map(childMsg => {
            const childTC = childMsg.toolCalls?.[0];
            if (!childTC) return null;
            // Use timestamp from child message if available
            return createToolMessage(childTC, childMsg.timestamp || new Date().toISOString());
          })
          .filter((c): c is ToolUseMessages => c !== null);
        
        // Also include any streaming children from pendingToolCalls
        for (const [childId, childTool] of pendingToolCalls) {
          if (childTool.parentToolUseId === toolId && !renderedToolCalls.has(childId)) {
            const childInput = parseToolArgs(childTool.args || "");
            children.push({
              type: "tool_use_messages",
              // Use timestamp from pending tool call (captured from TOOL_CALL_START)
              timestamp: childTool.timestamp || new Date().toISOString(),
              toolUseBlock: {
                type: "tool_use_block",
                id: childId,
                name: childTool.name,
                input: childInput,
              },
              resultBlock: {
                type: "tool_result_block",
                tool_use_id: childId,
                content: null,  // Still streaming
                is_error: false,
              },
            });
            renderedToolCalls.add(childId);
          }
        }
        
        // Also include any children from the childrenByParent map
        const childIds = childrenByParent.get(toolId) || [];
        for (const childId of childIds) {
          if (renderedToolCalls.has(childId)) continue;
          const childData = allToolCalls.get(childId);
          if (childData) {
            children.push(createToolMessage(childData.tc, childData.timestamp));
            renderedToolCalls.add(childId);
          }
        }
        
        const streamingToolMessage: HierarchicalToolMessage = {
          type: "tool_use_messages",
          // Use timestamp from pending tool call (captured from TOOL_CALL_START)
          timestamp: pendingTool.timestamp || new Date().toISOString(),
          toolUseBlock: {
            type: "tool_use_block",
            id: toolId,
            name: toolName,
            input: toolInput,
          },
          resultBlock: {
            type: "tool_result_block",
            tool_use_id: toolId,
            content: null,  // No result yet - still running!
            is_error: false,
          },
          children: children.length > 0 ? children : undefined,
        };
        
        result.push(streamingToolMessage);
        renderedToolCalls.add(toolId);
      }
    }
    
    return result;
  }, [
    aguiState.messages,
    aguiState.currentMessage,
    aguiState.currentToolCall,
    aguiState.pendingToolCalls,  // CRITICAL: Include so UI updates when new tools start
    aguiState.pendingChildren,   // CRITICAL: Include so UI updates when children finish
  ]);

  // Check if there are any real messages (user or assistant messages, not just system)
  const hasRealMessages = useMemo(() => {
    return streamMessages.some(
      (msg) => msg.type === "user_message" || msg.type === "agent_message"
    );
  }, [streamMessages]);

  // Clear queued messages when first agent response arrives
  useEffect(() => {
    const sentMessages = sessionQueue.messages.filter(m => m.sentAt);
    if (sentMessages.length > 0 && streamMessages.length > 0) {
      // Check if there's at least one agent message (response to our queued messages)
      const hasAgentResponse = streamMessages.some(
        msg => msg.type === "agent_message" || msg.type === "tool_use_messages"
      );
      
      if (hasAgentResponse) {
        sessionQueue.clearMessages();
      }
    }
  }, [sessionQueue, streamMessages]);

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

  // Auto-refresh artifacts when messages complete
  // UX improvement: Automatically refresh the artifacts panel when Claude writes new files,
  // so users can see their changes immediately without manually clicking the refresh button
  const previousToolResultCount = useRef(0);
  const artifactsRefreshTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const completionTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hasRefreshedOnCompletionRef = useRef(false);

  // Memoize the completed tool count to avoid redundant filtering
  // Uses extracted type guard for testability and proper validation
  const completedToolCount = useMemo(() => {
    return streamMessages.filter(isCompletedToolUseMessage).length;
  }, [streamMessages]);

  useEffect(() => {
    // Initialize on first mount to avoid triggering refresh for existing tools
    if (previousToolResultCount.current === 0 && completedToolCount > 0) {
      previousToolResultCount.current = completedToolCount;
      return;
    }

    // If we have new completed tools, refresh artifacts after a short delay
    if (completedToolCount > previousToolResultCount.current && completedToolCount > 0) {
      // Clear any pending refresh timeout
      if (artifactsRefreshTimeoutRef.current) {
        clearTimeout(artifactsRefreshTimeoutRef.current);
      }

      // Debounce refresh to avoid excessive calls during rapid tool completions
      artifactsRefreshTimeoutRef.current = setTimeout(() => {
        refetchArtifactsFiles();
      }, ARTIFACTS_DEBOUNCE_MS);

      previousToolResultCount.current = completedToolCount;
    }

    // Cleanup timeout on unmount or effect re-run
    return () => {
      if (artifactsRefreshTimeoutRef.current) {
        clearTimeout(artifactsRefreshTimeoutRef.current);
      }
    };
  }, [completedToolCount, refetchArtifactsFiles]);

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

  const sendChat = async () => {
    if (!chatInput.trim()) return;

    const finalMessage = chatInput.trim();
    setChatInput("");

    // Mark user interaction when they send first message
    setUserHasInteracted(true);

    const phase = session?.status?.phase;
    
    // If session is not yet running, queue the message for later
    // This includes: undefined (loading), "Pending", "Creating", or any other non-Running state
    if (!phase || phase !== "Running") {
      sessionQueue.addMessage(finalMessage);
      return;
    }

    try {
      await aguiSendMessage(finalMessage);
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to send message");
    }
  };

  const handleCommandClick = async (slashCommand: string) => {
    try {
      await aguiSendMessage(slashCommand);
      successToast(`Command ${slashCommand} sent`);
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to send command");
    }
  };

  // LEGACY: Old handleInterrupt removed - now using aguiInterrupt from useAGUIStream
  // which calls the proper AG-UI interrupt endpoint that signals Claude SDK

  const handleEndSession = () => {
    // Use stop API to end the session
    stopMutation.mutate(
      { projectName, sessionName, data: { reason: "end_session" } },
      {
        onSuccess: () => successToast("Session ended successfully"),
        onError: (err) =>
          errorToast(
            err instanceof Error ? err.message : "Failed to end session",
          ),
      },
    );
  };

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

        {/* Main content area */}
        <div className="flex-grow overflow-hidden bg-card">
          <div className="h-full">
            <div className="h-full flex gap-6">
              {/* Mobile sidebar overlay */}
              {mobileMenuOpen && (
                <div 
                  className="fixed inset-0 bg-background/80 backdrop-blur-sm z-40 md:hidden"
                  onClick={() => setMobileMenuOpen(false)}
                />
              )}

              {/* Left Column - Accordions - always show with state-based styling */}
              {session && (
                <div className={cn(
                  "flex-[0_0_400px] min-w-[350px] max-w-[500px] flex flex-col sticky top-0 self-start h-[calc(100vh-8rem)] pt-6 pl-6 pr-6 bg-card relative",
                  "md:flex md:pr-0",
                  mobileMenuOpen ? "fixed left-0 top-16 z-50 shadow-lg" : "hidden",
                  // Disable interactions when not running
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

                    <McpIntegrationsAccordion
                      projectName={projectName}
                      sessionName={sessionName}
                    />

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
                          <div className="border rounded-lg overflow-hidden">
                            <div className="px-2 py-1.5 border-b flex items-center justify-between bg-muted/30">
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

              {/* Right Column - Messages */}
              <div className="flex-1 min-w-0 flex flex-col">
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
                      <FeedbackProvider
                        projectName={projectName}
                        sessionName={sessionName}
                        username={currentUser?.username || currentUser?.displayName || "anonymous"}
                        initialPrompt={session?.spec?.initialPrompt}
                        activeWorkflow={workflowManagement.activeWorkflow || undefined}
                        messages={streamMessages}
                        traceId={langfuseTraceId || undefined}
                        messageFeedback={aguiState.messageFeedback}
                      >
                        <MessagesTab
                          session={session}
                          streamMessages={streamMessages}
                          chatInput={chatInput}
                          setChatInput={setChatInput}
                          onSendChat={() => Promise.resolve(sendChat())}
                          onInterrupt={aguiInterrupt}
                          onEndSession={() => Promise.resolve(handleEndSession())}
                          onGoToResults={() => {}}
                          onContinue={handleContinue}
                          workflowMetadata={workflowMetadata}
                          onCommandClick={handleCommandClick}
                          isRunActive={isRunActive}
                          showWelcomeExperience={!["Completed", "Failed", "Stopped", "Stopping"].includes(session?.status?.phase || "")}
                          activeWorkflow={workflowManagement.activeWorkflow}
                          userHasInteracted={userHasInteracted}
                          queuedMessages={sessionQueue.messages}
                          hasRealMessages={hasRealMessages}
                          welcomeExperienceComponent={
                            <WelcomeExperience
                              ootbWorkflows={ootbWorkflows}
                              onWorkflowSelect={handleWelcomeWorkflowSelect}
                              onUserInteraction={() => setUserHasInteracted(true)}
                              userHasInteracted={userHasInteracted}
                              sessionPhase={session?.status?.phase}
                              hasRealMessages={hasRealMessages}
                              onLoadWorkflow={() => setCustomWorkflowDialogOpen(true)}
                              selectedWorkflow={workflowManagement.selectedWorkflow}
                            />
                          }
                        />
                      </FeedbackProvider>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          </div>
        </div>
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
