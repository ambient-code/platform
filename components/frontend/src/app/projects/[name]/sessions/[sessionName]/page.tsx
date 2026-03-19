"use client";

import { useState, useEffect, useMemo, useRef, useCallback } from "react";
import {
  Loader2,
  PanelRight,
  PanelRightClose,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

// Custom components
import MessagesTab from "@/components/session/MessagesTab";
import { SessionStartingEvents } from "@/components/session/SessionStartingEvents";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { SessionHeader } from "./session-header";

// Extracted components
import { AddContextModal } from "./components/modals/add-context-modal";
import { UploadFileModal } from "./components/modals/upload-file-modal";
import { CustomWorkflowDialog } from "./components/modals/custom-workflow-dialog";
import { ManageRemoteDialog } from "./components/modals/manage-remote-dialog";

// New layout components
import { ContentTabs } from "./components/content-tabs";
import { FileViewer } from "./components/file-viewer";
import { ExplorerPanel } from "./components/explorer/explorer-panel";
import { SessionSettingsModal } from "./components/session-settings-modal";
import { WorkflowSelector } from "./components/workflow-selector";
import { useExplorerState } from "./hooks/use-explorer-state";
import { useFileTabs } from "./hooks/use-file-tabs";

// Extracted hooks and utilities
import { useGitOperations } from "./hooks/use-git-operations";
import { useWorkflowManagement } from "./hooks/use-workflow-management";
import { useFileOperations } from "./hooks/use-file-operations";
import { useSessionQueue } from "@/hooks/use-session-queue";
import { useDraftInput } from "@/hooks/use-draft-input";
import { useResizePanel } from "@/hooks/use-resize-panel";
import type { DirectoryOption, DirectoryRemote } from "./lib/types";

import type { MessageObject, ToolUseMessages, HierarchicalToolMessage, ReconciledRepo, SessionRepo } from "@/types/agentic-session";
import type { PlatformToolCall } from "@/types/agui";

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
  sessionKeys,
  useRunnerTypes,
} from "@/services/queries";
import { useCapabilities } from "@/services/queries/use-capabilities";
import {
  useWorkspaceList,
} from "@/services/queries/use-workspace";
import { toast } from "sonner";
import {
  useOOTBWorkflows,
  useWorkflowMetadata,
} from "@/services/queries/use-workflows";

import { useMutation, useQueryClient } from "@tanstack/react-query";
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
  const queryClient = useQueryClient();
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [repoChanging, setRepoChanging] = useState(false);
  const [pendingRepo, setPendingRepo] = useState<{ url: string; branch: string; status: "Cloning" } | null>(null);
  const [settingsModalOpen, setSettingsModalOpen] = useState(false);

  // Explorer panel state
  const explorer = useExplorerState();
  const explorerResize = useResizePanel("session-explorer-width", 340, 250, 500, "right");

  // File tabs state
  const fileTabs = useFileTabs();


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

  // Draft input hook (localStorage-backed)
  const { draft: chatInput, setDraft: setChatInput, clearDraft } = useDraftInput(projectName, sessionName);

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

  // Fetch runner capabilities and derive agent display name
  const { data: capabilities } = useCapabilities(projectName, sessionName, phase === "Running");
  const { data: runnerTypes } = useRunnerTypes(projectName);
  const agentName = useMemo(() => {
    if (capabilities?.framework && runnerTypes) {
      const matched = runnerTypes.find((rt) => rt.id === capabilities.framework);
      if (matched) return matched.displayName;
    }
    return undefined;
  }, [capabilities?.framework, runnerTypes]);

  // Track the current Langfuse trace ID for feedback association
  const [langfuseTraceId, setLangfuseTraceId] = useState<string | null>(null);

  // AG-UI streaming hook - replaces useSessionMessages and useSendChatMessage
  // Note: autoConnect is intentionally false to avoid SSR hydration mismatch
  // Connection is triggered manually in useEffect after client hydration
  const aguiStream = useAGUIStream({
    projectName: projectName || "",
    sessionName: sessionName || "",
    autoConnect: false, // Manual connection after hydration
    onError: (err) => {
      console.error("AG-UI stream error:", err)
    },
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
            toast.error(err instanceof Error ? err.message : "Failed to send queued message");
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
      setPendingRepo({ url: repo.url, branch: repo.branch, status: "Cloning" });
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
      // Refresh both data sources so the real repo entry replaces the optimistic one
      await refetchSession();
      await queryClient.invalidateQueries({
        queryKey: sessionKeys.reposStatus(projectName, sessionName),
      });
      setPendingRepo(null);

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
      toast.success("Repository added successfully");
    },
    onError: (error: Error) => {
      setPendingRepo(null);
      setRepoChanging(false);
      toast.error(error.message || "Failed to add repository");
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
      await refetchSession();
      // Invalidate runner repos cache so the removed repo disappears immediately
      queryClient.invalidateQueries({
        queryKey: sessionKeys.reposStatus(projectName, sessionName),
      });
      setRepoChanging(false);
      toast.success("Repository removed successfully");
    },
    onError: (error: Error) => {
      setRepoChanging(false);
      toast.error(error.message || "Failed to remove repository");
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
      toast.success(`File "${data.filename}" uploaded successfully`);
      // Refresh workspace to show uploaded file
      await refetchFileUploadsList();
      await refetchDirectoryFiles();
      await refetchArtifactsFiles();
      setUploadModalOpen(false);
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to upload file");
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
      toast.success("File removed successfully");
      // Refresh file lists
      await refetchFileUploadsList();
      await refetchDirectoryFiles();
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to remove file");
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
      { enabled: explorer.visible },
    );

  // Artifacts file operations
  const artifactsOps = useFileOperations({
    projectName,
    sessionName,
    basePath: "artifacts",
  });

  const { refetch: refetchArtifactsFilesRaw } =
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

  // File uploads list (for Context tab in explorer)
  const { data: fileUploadsList = [], refetch: refetchFileUploadsList } =
    useWorkspaceList(
      projectName,
      sessionName,
      "file-uploads",
      { enabled: explorer.visible && explorer.activeTab === "context" },
    );

  // Track if we've already initialized from session
  const initializedFromSessionRef = useRef(false);
  const workflowLoadedFromSessionRef = useRef(false);

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

  // Phase 1: convert committed messages + streaming tool cards into display format.
  // Does NOT depend on currentMessage / currentReasoning so it skips the full
  // O(n) traversal during text-streaming deltas (the most frequent event type).
  const committedStreamMessages: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage> = useMemo(() => {

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
      tc: PlatformToolCall,
      timestamp: string
    ): ToolUseMessages => {
      const toolInput = parseToolArgs(tc.function.arguments);
      return {
        type: "tool_use_messages",
        timestamp,
        toolUseBlock: {
          type: "tool_use_block",
          id: tc.id,
          name: tc.function.name,
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
    const allToolCalls = new Map<string, { tc: PlatformToolCall; timestamp: string }>();

    for (const msg of aguiState.messages) {
      const timestamp = msg.timestamp || "";

      if (msg.toolCalls && Array.isArray(msg.toolCalls)) {
        for (const tc of msg.toolCalls) {
          if (tc && tc.id && tc.function?.name) {
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
      const streamingTC: PlatformToolCall = {
        id: streamingToolId,
        type: "function",
        function: {
          name: toolName,
          arguments: aguiState.currentToolCall.args || "",
        },
        parentToolUseId: streamingParentId,
        status: "running",
      };

      if (!allToolCalls.has(streamingToolId)) {
        allToolCalls.set(streamingToolId, {
          tc: streamingTC,
          timestamp: aguiState.pendingToolCalls?.get(streamingToolId)?.timestamp || ""
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
                  timestamp: childMsg.timestamp || "",
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
          console.warn(`  ⚠️ Orphaned child: ${tc.function.name} (${toolId.substring(0, 8)}) - parent ${tc.parentToolUseId.substring(0, 8)} not found`);
          topLevelTools.add(toolId);
        }
      }
    }

    // Track which tool calls we've already rendered
    const renderedToolCalls = new Set<string>();

    // Phase C: Process messages and build hierarchical structure
    for (const msg of aguiState.messages) {
      const timestamp = msg.timestamp || "";

      // Handle text content by role
      if (msg.role === "user") {
        // Hide AskUserQuestion response messages from the chat
        const msgMeta = msg.metadata as Record<string, unknown> | undefined;
        if (msgMeta?.type === "ask_user_question_response") {
          continue;
        }

        result.push({
          type: "user_message",
          id: msg.id,  // Preserve message ID for feedback association
          content: { type: "text_block", text: (typeof msg.content === 'string' ? msg.content : '') || "" },
          timestamp,
        });
      } else if (msg.role === "assistant") {
        // Check if content is a structured reasoning block
        const contentObj = typeof msg.content === 'object' && msg.content !== null ? msg.content as Record<string, unknown> : null;
        const metadata = msg.metadata as Record<string, unknown> | undefined;
        if (
          contentObj?.type === "reasoning_block" ||
          metadata?.type === "reasoning_block" ||
          metadata?.type === "thinking_block"  // TODO: remove after all sessions use REASONING_* events
        ) {
          const thinkingText =
            (contentObj?.thinking as string) ||
            (metadata?.thinking as string) ||
            (typeof msg.content === 'string' ? msg.content : '') ||
            "";
          result.push({
            type: "agent_message",
            id: msg.id,
            content: {
              type: "reasoning_block",
              thinking: thinkingText,
              signature: (contentObj?.signature as string) || (metadata?.signature as string) || "",
            },
            model: "claude",
            timestamp,
          });
        } else if (msg.content && typeof msg.content === 'string') {
          // Only push text message if there's actual string content
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
          if (!tc || !tc.id || !tc.function?.name) continue;

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
          const toolInput = parseToolArgs(tc.function.arguments);

          const toolMessage: HierarchicalToolMessage = {
            type: "tool_use_messages",
            timestamp,
            toolUseBlock: {
              type: "tool_use_block",
              id: tc.id,
              name: tc.function.name,
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
            renderedToolCalls.add(childTC.id);  // Track to prevent duplicates below
            return createToolMessage(childTC, childMsg.timestamp || "");
          })
          .filter((c): c is ToolUseMessages => c !== null);

        // Also include any streaming children from pendingToolCalls
        for (const [childId, childTool] of pendingToolCalls) {
          if (childTool.parentToolUseId === toolId && !renderedToolCalls.has(childId)) {
            const childInput = parseToolArgs(childTool.args || "");
            children.push({
              type: "tool_use_messages",
              timestamp: childTool.timestamp || "",
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
          timestamp: pendingTool.timestamp || "",
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
    aguiState.currentToolCall,   // Needed in Phase A to avoid orphaned-child promotion
    aguiState.pendingToolCalls,  // CRITICAL: Include so UI updates when new tools start
    aguiState.pendingChildren,   // CRITICAL: Include so UI updates when children finish
  ]);

  // Phase 2: append streaming text / reasoning bubbles to the committed list.
  // This is O(1) and is the only memo that re-runs on every TEXT_MESSAGE_CONTENT
  // or REASONING_MESSAGE_CONTENT delta (the most frequent events during active runs).
  const streamMessages: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage> = useMemo(() => {
    const result: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage> = [];

    // Prepend synthetic user message for initialPrompt if not already in the stream
    const initialPrompt = session?.spec?.initialPrompt;
    if (initialPrompt) {
      const hasInitialUserMessage = committedStreamMessages.some(
        (msg) => {
          if (msg.type !== "user_message" || !("content" in msg)) return false;
          const content = msg.content;
          if (typeof content === "object" && content !== null && "type" in content && content.type === "text_block" && "text" in content) {
            return content.text === initialPrompt;
          }
          return false;
        }
      );
      if (!hasInitialUserMessage) {
        result.push({
          type: "user_message",
          content: { type: "text_block", text: initialPrompt },
          timestamp: session?.metadata?.creationTimestamp || "",
        });
      }
    }

    result.push(...committedStreamMessages);

    const activeReasoning = aguiState.currentReasoning || aguiState.currentThinking;
    if (activeReasoning?.content) {
      result.push({
        type: "agent_message",
        content: {
          type: "reasoning_block",
          thinking: activeReasoning.content,
          signature: "",
        },
        model: "claude",
        timestamp: activeReasoning.timestamp || "",
        streaming: true,
      } as MessageObject & { streaming?: boolean });
    }

    if (aguiState.currentMessage?.content) {
      result.push({
        type: "agent_message",
        content: { type: "text_block", text: aguiState.currentMessage.content },
        model: "claude",
        timestamp: aguiState.currentMessage.timestamp || "",
        streaming: true,
      } as MessageObject & { streaming?: boolean });
    }

    return result;
  }, [
    committedStreamMessages,
    aguiState.currentMessage,
    aguiState.currentReasoning,
    aguiState.currentThinking,
    session?.spec?.initialPrompt,
    session?.metadata?.creationTimestamp,
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
      } else {
        // No matching OOTB workflow found - treat as custom workflow.
        // Restore the full custom workflow details from the session CR
        // so they survive page refresh.
        const aw = session.spec.activeWorkflow;
        workflowManagement.setCustomWorkflow(
          aw.gitUrl,
          aw.branch || "main",
          aw.path || ""
        );
        workflowManagement.setActiveWorkflow("custom");
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
        onSuccess: () => toast.success("Session stopped successfully"),
        onError: (err) =>
          toast.error(
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
          toast.error(
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
          toast.success("Session restarted successfully");
        },
        onError: (err) =>
          toast.error(
            err instanceof Error ? err.message : "Failed to restart session",
          ),
      },
    );
  };

  const sendChat = async () => {
    if (!chatInput.trim()) return;

    const finalMessage = chatInput.trim();
    clearDraft();

    // Mark user interaction when they send first message
    const phase = session?.status?.phase;

    // If session is not yet running, queue the message for later
    // This includes: undefined (loading), "Pending", "Creating", or any other non-Running state
    if (!phase || phase !== "Running") {
      sessionQueue.addMessage(finalMessage);
      return;
    }

    try {
      await aguiSendMessage(finalMessage);
      // Invalidate session caches so sidebar/list reflect the new activity
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(projectName, sessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.list(projectName) });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send message");
    }
  };

  // Send an AskUserQuestion response (hidden from chat, properly formatted)
  const sendToolAnswer = async (formattedAnswer: string) => {
    try {
      await aguiSendMessage(formattedAnswer, {
        type: "ask_user_question_response",
      });
      // Invalidate session caches so sidebar/list reflect the resumed activity
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(projectName, sessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.list(projectName) });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send answer");
      throw err;
    }
  };

  const handleCommandClick = async (slashCommand: string) => {
    try {
      await aguiSendMessage(slashCommand);
      toast.success(`Command ${slashCommand} sent`);
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(projectName, sessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.list(projectName) });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send command");
    }
  };

  // LEGACY: Old handleInterrupt removed - now using aguiInterrupt from useAGUIStream
  // which calls the proper AG-UI interrupt endpoint that signals Claude SDK

  // Computed values for explorer panel
  const removingRepoName = removeRepoMutation.isPending
    ? removeRepoMutation.variables
    : null;

  const explorerRepositories = useMemo(() => [
    ...(pendingRepo ? [pendingRepo] : []),
    ...(reposStatus?.repos || session?.status?.reconciledRepos || session?.spec?.repos || []).map(
      (r) => {
        const name = ('name' in r ? r.name : undefined) || r.url?.split('/').pop()?.replace('.git', '');
        return name === removingRepoName ? { ...r, status: "Removing" as const } : r;
      },
    ),
  ], [pendingRepo, reposStatus?.repos, session?.status?.reconciledRepos, session?.spec?.repos, removingRepoName]);

  const repoBranches = useMemo(() => {
    const branches: Record<string, string | undefined> = {};

    // Build lookup maps once — O(n) instead of O(n²) finds
    const realtimeRepoMap = new Map(
      (reposStatus?.repos || []).map((r) => [r.name, r]),
    );
    const reconciledRepoMap = new Map(
      (session?.status?.reconciledRepos || []).map((r: ReconciledRepo) => {
        const name = r.name || r.url?.split("/").pop()?.replace(".git", "");
        return [name, r];
      }),
    );

    directoryOptions.forEach((opt) => {
      if (opt.type === "repo") {
        const repoName = opt.path.replace(/^repos\//, "");
        const realtimeRepo = realtimeRepoMap.get(repoName);
        const reconciledRepo = reconciledRepoMap.get(repoName);
        branches[repoName] = realtimeRepo?.currentActiveBranch
          || reconciledRepo?.currentActiveBranch
          || reconciledRepo?.branch;
      }
    });
    return branches;
  }, [directoryOptions, reposStatus?.repos, session?.status?.reconciledRepos]);

  // Handle file open from explorer panel -> opens as tab
  const handleFileOpenInTab = useCallback((filePath: string) => {
    const fileName = filePath.split("/").pop() ?? filePath;
    fileTabs.openFile({ path: filePath, name: fileName });
  }, [fileTabs]);

  // Memoize derived props to avoid unstable references on every render
  const explorerGitStatus = useMemo(() =>
    gitOps.gitStatus ? {
      hasChanges: gitOps.gitStatus.hasChanges ?? false,
      totalAdded: gitOps.gitStatus.totalAdded ?? 0,
      totalRemoved: gitOps.gitStatus.totalRemoved ?? 0,
    } : undefined,
    [gitOps.gitStatus],
  );

  const explorerUploadedFiles = useMemo(() =>
    fileUploadsList.map((f) => ({
      name: f.name,
      path: f.path,
      size: f.size,
    })),
    [fileUploadsList],
  );

  const handleRefresh = useCallback(() => {
    refetchDirectoryFiles();
  }, [refetchDirectoryFiles]);

  const handleOpenUploadModal = useCallback(() => {
    setUploadModalOpen(true);
  }, []);

  const handleOpenContextModal = useCallback(() => {
    setContextModalOpen(true);
  }, []);

  const handleRemoveRepository = useCallback((repoName: string) => {
    removeRepoMutation.mutate(repoName);
  }, [removeRepoMutation]);

  const handleRemoveFile = useCallback((fileName: string) => {
    removeFileMutation.mutate(fileName);
  }, [removeFileMutation]);

  // Loading state
  if (isLoading || !projectName || !sessionName) {
    return (
      <div className="h-full overflow-hidden bg-background flex items-center justify-center">
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
      <div className="h-full overflow-hidden bg-background flex flex-col">
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

  // Chat/FileViewer content rendering helper
  const renderMainContent = () => {
    if (fileTabs.activeTab.type === "file") {
      return (
        <FileViewer
          projectName={projectName}
          sessionName={sessionName}
          filePath={fileTabs.activeTab.path}
          sessionPhase={phase}
        />
      );
    }

    // Chat view
    return (
      <Card className="relative flex-1 flex flex-col overflow-hidden py-0 border-0 rounded-none">
        <CardContent className="px-6 pt-0 pb-0 flex-1 flex flex-col overflow-hidden">
          {repoChanging && (
            <div className="absolute inset-0 bg-background/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
              <Alert className="max-w-md mx-4">
                <Loader2 className="h-4 w-4 animate-spin" />
                <AlertTitle>Updating Repositories...</AlertTitle>
                <AlertDescription>
                  <p>Please wait while repositories are being updated. This may take 10-20 seconds...</p>
                </AlertDescription>
              </Alert>
            </div>
          )}

          <div className="relative flex flex-col flex-1 overflow-hidden">
            {(phase === "Creating" || phase === "Pending") && (
              <div className="absolute inset-0 bg-background/80 backdrop-blur-sm z-10">
                <SessionStartingEvents
                  projectName={projectName}
                  sessionName={sessionName}
                />
              </div>
            )}
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
                onSendToolAnswer={sendToolAnswer}
                onInterrupt={aguiInterrupt}
                onGoToResults={() => {}}
                onContinue={handleContinue}
                workflowMetadata={workflowMetadata}
                onCommandClick={handleCommandClick}
                isRunActive={isRunActive}
                queuedMessages={sessionQueue.messages}
                hasRealMessages={hasRealMessages}
                onCancelQueuedMessage={sessionQueue.cancelMessage}
                onUpdateQueuedMessage={sessionQueue.updateMessage}
                onClearQueue={sessionQueue.clearMessages}
                agentName={agentName}
                onAddRepository={handleOpenContextModal}
                onUploadFile={handleOpenUploadModal}
                workflowSlot={
                  <WorkflowSelector
                    sessionPhase={session?.status?.phase}
                    activeWorkflow={workflowManagement.activeWorkflow}
                    activeWorkflowDetails={
                      workflowManagement.activeWorkflow === "custom" && session?.spec?.activeWorkflow
                        ? {
                            gitUrl: session.spec.activeWorkflow.gitUrl,
                            branch: session.spec.activeWorkflow.branch || "main",
                            path: session.spec.activeWorkflow.path || "",
                          }
                        : undefined
                    }
                    selectedWorkflow={workflowManagement.selectedWorkflow}
                    workflowActivating={workflowManagement.workflowActivating}
                    ootbWorkflows={ootbWorkflows}
                    onWorkflowChange={handleWorkflowChange}
                    onLoadCustom={() => setCustomWorkflowDialogOpen(true)}
                  />
                }
              />
            </FeedbackProvider>
          </div>
        </CardContent>
      </Card>
    );
  };

  const actionLoading = stopMutation.isPending
    ? "stopping"
    : deleteMutation.isPending
      ? "deleting"
      : continueMutation.isPending
        ? "resuming"
        : null;

  return (
    <>
      <title>{`${session.spec.displayName || session.metadata.name} · Ambient Code Platform`}</title>
      <div className="h-full overflow-hidden bg-card flex">
        {/* Center: Content tabs + Chat/FileViewer */}
        <div className="flex-1 min-w-0 flex flex-col h-full">
          {/* Tab bar */}
          <ContentTabs
            openTabs={fileTabs.openTabs}
            activeTab={fileTabs.activeTab}
            onSwitchToChat={fileTabs.switchToChat}
            onSwitchToFile={fileTabs.switchToFile}
            onCloseFile={fileTabs.closeFile}
            rightActions={
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={explorer.toggle}
                  className={cn(
                    "h-7 px-2 gap-1.5",
                    explorer.visible && "bg-accent"
                  )}
                  title={explorer.visible ? "Hide explorer" : "Show explorer"}
                >
                  {explorer.visible ? (
                    <PanelRightClose className="h-4 w-4" />
                  ) : (
                    <PanelRight className="h-4 w-4" />
                  )}
                  Explorer
                </Button>
                <SessionHeader
                  session={session}
                  projectName={projectName}
                  actionLoading={actionLoading}
                  onRefresh={refetchSession}
                  onStop={handleStop}
                  onContinue={handleContinue}
                  onDelete={handleDelete}
                  onOpenSettings={() => setSettingsModalOpen(true)}
                  renderMode="kebab-only"
                />
              </>
            }
          />

          {/* Main content */}
          <div className="relative flex-1 flex flex-col overflow-hidden">
            {renderMainContent()}
          </div>
        </div>

        {/* Right: Explorer panel */}
        <div
          className={cn(
            "h-full overflow-hidden flex-shrink-0 relative",
            !explorerResize.isDragging && "transition-[width] duration-200 ease-in-out",
            explorer.visible ? "" : "!w-0"
          )}
          style={{ width: explorer.visible ? explorerResize.width : 0 }}
        >
          {/* Resize handle */}
          <div
            onMouseDown={explorerResize.onMouseDown}
            className="absolute top-0 left-0 w-1 h-full cursor-col-resize hover:bg-primary/50 transition-colors z-10"
          />
          <div className="h-full" style={{ width: explorerResize.width }}>
            <ExplorerPanel
              visible={explorer.visible}
              activeTab={explorer.activeTab}
              onTabChange={explorer.setActiveTab}
              onClose={explorer.close}
              directoryOptions={directoryOptions}
              selectedDirectory={selectedDirectory}
              onDirectoryChange={setSelectedDirectory}
              files={directoryFiles}
              currentSubPath={fileOps.currentSubPath}
              viewingFile={fileOps.viewingFile}
              isLoadingFile={fileOps.loadingFile}
              onFileOrFolderSelect={fileOps.handleFileOrFolderSelect}
              onNavigateBack={fileOps.navigateBack}
              onRefresh={handleRefresh}
              onDownloadFile={fileOps.handleDownloadFile}
              onFileOpen={handleFileOpenInTab}
              gitStatus={explorerGitStatus}
              repoBranches={repoBranches}
              repositories={explorerRepositories}
              uploadedFiles={explorerUploadedFiles}
              onAddRepository={handleOpenContextModal}
              onUploadFile={handleOpenUploadModal}
              onRemoveRepository={handleRemoveRepository}
              onRemoveFile={handleRemoveFile}
            />
          </div>
        </div>
      </div>

      {/* Modals */}
      <SessionSettingsModal
        open={settingsModalOpen}
        onOpenChange={setSettingsModalOpen}
        session={session}
        projectName={projectName}
      />

      <AddContextModal
        open={contextModalOpen}
        onOpenChange={setContextModalOpen}
        onAddRepository={async (url, branch, autoPush) => {
          await addRepoMutation.mutateAsync({ url, branch, autoPush });
          setContextModalOpen(false);
        }}
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
