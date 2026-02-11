"use client";

import React, { useCallback, useContext, useMemo, useRef, useState, useEffect } from "react";
import { useChatContext } from "@copilotkit/react-ui";
import type { InputProps } from "@copilotkit/react-ui";
import {
  Loader2,
  Play,
  Paperclip,
  Clock,
  X,
  Pencil,
  Settings,
  Terminal,
  Users,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useToast } from "@/hooks/use-toast";
import { useResizeTextarea } from "@/hooks/use-resize-textarea";
import { AttachmentPreview, type PendingAttachment } from "@/components/chat/AttachmentPreview";
import { QueuedMessageBubble } from "@/components/chat/QueuedMessageBubble";
import { SessionActiveCtx, ResumeCtx, WorkflowMetadataCtx, InputEnhancementsCtx } from "./session-contexts";
import { AutocompletePopup, type AutocompleteItem } from "./AutocompletePopup";

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10 MB

function generatePreview(file: File): Promise<string | undefined> {
  return new Promise((resolve) => {
    if (!file.type.startsWith("image/")) {
      resolve(undefined);
      return;
    }
    const reader = new FileReader();
    reader.onload = (e) => resolve(e.target?.result as string);
    reader.onerror = () => resolve(undefined);
    reader.readAsDataURL(file);
  });
}

function makeAttachmentId() {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// ─── Toolbar item-list popover (shared between Agents and Commands) ───

type ToolbarItemListProps = {
  items: Array<{ id: string; name: string; description?: string; slashCommand?: string }>;
  type: "agent" | "command";
  onInsertAgent?: (name: string) => void;
  onRunCommand?: (slashCommand: string) => void;
};

const ToolbarItemList: React.FC<ToolbarItemListProps> = ({ items, type, onInsertAgent, onRunCommand }) => {
  const heading = type === "agent" ? "Available Agents" : "Available Commands";
  const subtitle = type === "agent"
    ? "Mention agents in your message to collaborate with them"
    : "Run workflow commands to perform specific actions";
  const emptyLabel = type === "agent" ? "No agents available" : "No commands available";

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <h4 className="font-medium text-sm">{heading}</h4>
        <p className="text-xs text-muted-foreground">{subtitle}</p>
      </div>
      <div className="max-h-[400px] overflow-y-scroll space-y-2 pr-2 scrollbar-thin">
        {items.length === 0 ? (
          <p className="text-xs text-muted-foreground py-2">{emptyLabel}</p>
        ) : (
          items.map((item) => {
            const isAgent = type === "agent";
            const shortName = isAgent ? item.name.split(" - ")[0] : "";
            return (
              <div key={item.id} className="p-3 rounded-md border bg-muted/30">
                <div className="flex items-center justify-between mb-1">
                  <h3 className="text-sm font-bold">{item.name}</h3>
                  {isAgent ? (
                    <Button variant="outline" size="sm" className="flex-shrink-0 h-7 text-xs" onClick={() => onInsertAgent?.(shortName)}>
                      @{shortName}
                    </Button>
                  ) : (
                    <Button variant="outline" size="sm" className="flex-shrink-0 h-7 text-xs" onClick={() => onRunCommand?.(item.slashCommand ?? "")}>
                      Run {item.slashCommand}
                    </Button>
                  )}
                </div>
                {item.description && (
                  <p className="text-xs text-muted-foreground">{item.description}</p>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

/**
 * Custom Input — session-aware, CopilotKit-compatible.
 *
 * Handles:
 * - Session active/inactive states with resume banner
 * - `/` slash-command and `@` agent autocomplete (inline)
 * - Agent/Command toolbar popovers (click-to-browse)
 * - Settings dropdown (show/hide system messages)
 * - Queued message bubbles with cancel
 * - File attach (paperclip) and clipboard paste (images)
 * - Drag-to-resize textarea
 * - Prompt history (Arrow Up/Down through sent + queued messages)
 * - Queued message editing (update in-place or send as new)
 * - Queue count indicator with clear button
 * - Phase-aware status banners (Creating, Terminal)
 */
export function SessionAwareInput({
  inProgress,
  onSend,
  onStop,
  hideStopButton = false,
  chatReady = false,
}: InputProps) {
  const isSessionActive = useContext(SessionActiveCtx);
  const { onResume, isResuming } = useContext(ResumeCtx);
  const context = useChatContext();
  const { commands, agents } = useContext(WorkflowMetadataCtx);
  const {
    onPasteImage,
    sessionPhase,
    queuedMessages,
    onCancelQueuedMessage,
    onUpdateQueuedMessage,
    onClearQueue,
    queuedCount,
    isRunActive,
    showSystemMessages,
    onShowSystemMessagesChange,
    onCommandClick,
    onQueueMessage,
    onMarkSent,
  } = useContext(InputEnhancementsCtx);

  const { toast } = useToast();
  const [text, setText] = useState("");

  // ── Resume banner flag for hibernated sessions ─────────────────────
  const showResumeBanner = !isSessionActive && !!onResume;
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const autocompleteRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // ── Resize ─────────────────────────────────────────────────────────
  const { textareaHeight, handleResizeStart } = useResizeTextarea({ defaultHeight: 80, minHeight: 44, maxHeight: 260 });

  // ── Attachments ────────────────────────────────────────────────────
  const [pendingAttachments, setPendingAttachments] = useState<PendingAttachment[]>([]);

  // ── Toolbar popover states ─────────────────────────────────────────
  const [agentsPopoverOpen, setAgentsPopoverOpen] = useState(false);
  const [commandsPopoverOpen, setCommandsPopoverOpen] = useState(false);

  // ── Prompt history ─────────────────────────────────────────────────
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [draftInput, setDraftInput] = useState("");
  const [editingQueuedId, setEditingQueuedId] = useState<string | null>(null);
  const [sentHistory, setSentHistory] = useState<string[]>([]);

  type HistoryEntry = { text: string; queuedId?: string };

  const combinedHistory = useMemo<HistoryEntry[]>(() => {
    const queued = queuedMessages
      .filter((m) => !m.sentAt)
      .map((m) => ({ text: m.content, queuedId: m.id }));
    const sent = sentHistory.map((t) => ({ text: t }));
    return [...queued, ...sent];
  }, [queuedMessages, sentHistory]);

  const resetHistory = () => {
    setHistoryIndex(-1);
    setDraftInput("");
    setEditingQueuedId(null);
  };

  // ── Pending (unsent) queued messages for bubble display ────────────
  const pendingQueuedMessages = useMemo(
    () => queuedMessages.filter((m) => !m.sentAt),
    [queuedMessages],
  );

  // ── Autocomplete state ─────────────────────────────────────────────
  const [acOpen, setAcOpen] = useState(false);
  const [acType, setAcType] = useState<"command" | "agent" | null>(null);
  const [acFilter, setAcFilter] = useState("");
  const [acTriggerPos, setAcTriggerPos] = useState(0);
  const [acSelected, setAcSelected] = useState(0);

  // ── Phase-derived state ────────────────────────────────────────────
  const isTerminalState = ["Completed", "Failed", "Stopped"].includes(sessionPhase);
  const isCreating = ["Creating", "Pending"].includes(sessionPhase);

  // Textarea is only fully disabled when there's no session at all.
  // During Creating/Pending, users can type (messages get queued).
  // During Running + agent busy, users can type (messages get queued).
  // chatReady=false just means CopilotKit hasn't connected yet — still allow typing for the queue.
  const fullyDisabled = !isSessionActive && !onResume && !isCreating;
  const canSendDirect = isSessionActive && chatReady && !inProgress;

  // ── Queue drain: send next queued message when CopilotKit is ready ─
  // Uses onSend (same path as manual send) so there's exactly ONE send
  // mechanism.  Triggers when canSendDirect transitions false → true
  // (agent finished or CopilotKit just connected).
  const prevCanSendRef = useRef(canSendDirect);
  useEffect(() => {
    const justBecameReady = canSendDirect && !prevCanSendRef.current;
    prevCanSendRef.current = canSendDirect;

    if (!justBecameReady) return;

    const next = queuedMessages.find((m) => !m.sentAt);
    if (!next) return;

    // Mark as sent first, then send through CopilotKit's normal onSend
    onMarkSent?.(next.id);
    onSend(next.content);
  }, [canSendDirect, queuedMessages, onSend, onMarkSent]);

  // ── Filtered autocomplete items ────────────────────────────────────
  const acItems = useMemo<AutocompleteItem[]>(() => {
    if (!acOpen || !acType) return [];
    const filter = acFilter.toLowerCase();

    if (acType === "command") {
      return commands
        .filter(
          (cmd) =>
            cmd.name.toLowerCase().includes(filter) ||
            cmd.slashCommand.toLowerCase().includes(filter) ||
            cmd.description?.toLowerCase().includes(filter),
        )
        .map((cmd) => ({
          id: cmd.id,
          label: cmd.slashCommand,
          detail: cmd.name,
          insertText: `${cmd.slashCommand} `,
        }));
    }

    return agents
      .filter(
        (a) =>
          a.name.toLowerCase().includes(filter) ||
          a.description?.toLowerCase().includes(filter),
      )
      .map((a) => {
        const short = a.name.split(" - ")[0];
        return {
          id: a.id,
          label: `@${short}`,
          detail: a.name,
          insertText: `@${short} `,
        };
      });
  }, [acOpen, acType, acFilter, commands, agents]);

  // ── Close autocomplete ─────────────────────────────────────────────
  const closeAutocomplete = useCallback(() => {
    setAcOpen(false);
    setAcType(null);
    setAcFilter("");
    setAcSelected(0);
  }, []);

  // ── Click-outside dismissal ────────────────────────────────────────
  useEffect(() => {
    if (!acOpen) return;
    const handler = (e: MouseEvent) => {
      if (
        autocompleteRef.current &&
        !autocompleteRef.current.contains(e.target as Node) &&
        textareaRef.current &&
        !textareaRef.current.contains(e.target as Node)
      ) {
        closeAutocomplete();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [acOpen, closeAutocomplete]);

  // ── Handle autocomplete selection ──────────────────────────────────
  const handleAcSelect = useCallback(
    (item: AutocompleteItem) => {
      if (!textareaRef.current) return;
      const cursorPos = textareaRef.current.selectionStart;
      const before = text.substring(0, acTriggerPos);
      const after = text.substring(cursorPos);
      const newText = before + item.insertText + after;
      setText(newText);
      closeAutocomplete();

      requestAnimationFrame(() => {
        if (textareaRef.current) {
          const pos = before.length + item.insertText.length;
          textareaRef.current.selectionStart = pos;
          textareaRef.current.selectionEnd = pos;
          textareaRef.current.focus();
        }
      });
    },
    [text, acTriggerPos, closeAutocomplete],
  );

  // ── Input change handler with trigger detection ────────────────────
  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newValue = e.target.value;
      const cursorPos = e.target.selectionStart;
      setText(newValue);

      if (historyIndex >= 0) {
        setHistoryIndex(-1);
        setDraftInput("");
      }

      if (cursorPos > 0) {
        const charBefore = newValue[cursorPos - 1];
        const textBefore = newValue.substring(0, cursorPos);

        if (charBefore === "/" || charBefore === "@") {
          const isAtStart = cursorPos === 1;
          const afterWhitespace = cursorPos >= 2 && /\s/.test(newValue[cursorPos - 2]);
          if (isAtStart || afterWhitespace) {
            setAcTriggerPos(cursorPos - 1);
            setAcType(charBefore === "@" ? "agent" : "command");
            setAcFilter("");
            setAcSelected(0);
            setAcOpen(true);
            return;
          }
        }

        if (acOpen) {
          const filterText = textBefore.substring(acTriggerPos + 1);
          if (cursorPos <= acTriggerPos || /\s/.test(filterText)) {
            closeAutocomplete();
          } else {
            setAcFilter(filterText);
            setAcSelected(0);
          }
        }
      } else if (acOpen) {
        closeAutocomplete();
      }
    },
    [acOpen, acTriggerPos, closeAutocomplete, historyIndex],
  );

  // ── Upload pending attachments ─────────────────────────────────────
  const uploadPendingAttachments = useCallback(async (): Promise<boolean> => {
    const toUpload = pendingAttachments.filter((a) => !a.uploading && !a.error);
    if (toUpload.length === 0 || !onPasteImage) return true;

    for (const attachment of toUpload) {
      setPendingAttachments((prev) =>
        prev.map((a) => (a.id === attachment.id ? { ...a, uploading: true } : a)),
      );
      try {
        await onPasteImage(attachment.file);
        setPendingAttachments((prev) =>
          prev.map((a) => (a.id === attachment.id ? { ...a, uploading: false } : a)),
        );
      } catch {
        setPendingAttachments((prev) =>
          prev.map((a) =>
            a.id === attachment.id ? { ...a, uploading: false, error: "Upload failed" } : a,
          ),
        );
        return false;
      }
    }
    return true;
  }, [pendingAttachments, onPasteImage]);

  // ── Send message (or queue if agent is busy / session not ready) ──
  const send = useCallback(async () => {
    if (fullyDisabled) return;
    const trimmed = text.trim();
    if (trimmed.length === 0 && pendingAttachments.length === 0) return;

    // If editing a queued message, update it in-place
    if (editingQueuedId && onUpdateQueuedMessage) {
      onUpdateQueuedMessage(editingQueuedId, trimmed);
      setText("");
      resetHistory();
      setPendingAttachments([]);
      toast({ title: "Queued message updated" });
      return;
    }

    // If we can send directly (session running, CopilotKit ready, agent idle), do it
    if (canSendDirect) {
      const uploaded = await uploadPendingAttachments();
      if (!uploaded) return;

      if (trimmed) {
        setSentHistory((prev) => [trimmed, ...prev.slice(0, 49)]);
      }

      onSend(text);
      setText("");
      closeAutocomplete();
      resetHistory();
      setPendingAttachments([]);
      textareaRef.current?.focus();
      return;
    }

    // Otherwise queue the message — it'll be drained by QueueDrainer when ready
    if (trimmed && onQueueMessage) {
      onQueueMessage(trimmed);
      setText("");
      closeAutocomplete();
      resetHistory();
      textareaRef.current?.focus();
      toast({
        title: "Message queued",
        description: isRunActive
          ? "Your message will be sent when the agent is ready."
          : "Your message will be sent when the session starts.",
      });
    }
  }, [
    fullyDisabled, canSendDirect, text, pendingAttachments, editingQueuedId,
    onUpdateQueuedMessage, uploadPendingAttachments, onSend, closeAutocomplete,
    toast, onQueueMessage, isRunActive,
  ]);

  // ── "Send as new" when editing a queued message ────────────────────
  const handleSendAsNew = useCallback(async () => {
    if (!text.trim() || !editingQueuedId) return;
    onCancelQueuedMessage?.(editingQueuedId);
    resetHistory();

    const uploaded = await uploadPendingAttachments();
    if (!uploaded) return;

    setSentHistory((prev) => [text.trim(), ...prev.slice(0, 49)]);
    onSend(text);
    setText("");
    setPendingAttachments([]);
  }, [text, editingQueuedId, onCancelQueuedMessage, uploadPendingAttachments, onSend]);

  // ── Paste handler (images) ─────────────────────────────────────────
  const handlePaste = useCallback(
    async (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const items = Array.from(e.clipboardData?.items || []);
      const imageItems = items.filter((item) => item.type.startsWith("image/"));

      if (imageItems.length > 0 && onPasteImage) {
        e.preventDefault();
        for (const item of imageItems) {
          const file = item.getAsFile();
          if (!file) continue;
          if (file.size > MAX_FILE_SIZE) {
            toast({ variant: "destructive", title: "File too large", description: `Maximum file size is 10 MB.` });
            continue;
          }
          const renamedFile =
            file.name === "image.png" || file.name === "image.jpg"
              ? new File([file], `paste-${new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19)}.png`, { type: file.type })
              : file;
          const preview = await generatePreview(renamedFile);
          setPendingAttachments((prev) => [...prev, { id: makeAttachmentId(), file: renamedFile, preview }]);
        }
      }
    },
    [onPasteImage, toast],
  );

  // ── File picker handler ────────────────────────────────────────────
  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(e.target.files || []);
      for (const file of files) {
        if (file.size > MAX_FILE_SIZE) {
          toast({ variant: "destructive", title: "File too large", description: `"${file.name}" exceeds 10 MB.` });
          continue;
        }
        const preview = await generatePreview(file);
        setPendingAttachments((prev) => [...prev, { id: makeAttachmentId(), file, preview }]);
      }
      e.target.value = "";
    },
    [toast],
  );

  const hasContent = text.trim().length > 0 || pendingAttachments.length > 0;
  // Can send directly or queue — button is enabled whenever there's content and we're not fully dead
  const canSendOrQueue = !fullyDisabled && hasContent;
  const canStop = isSessionActive && inProgress && !hideStopButton;
  const sendDisabled = !canSendOrQueue && !canStop;

  const willQueue = !canSendDirect && !fullyDisabled;
  const { buttonIcon, buttonAlt } = useMemo(() => {
    if (fullyDisabled) return { buttonIcon: context.icons.sendIcon, buttonAlt: "Session unavailable" };
    if (canStop) return { buttonIcon: context.icons.stopIcon, buttonAlt: "Stop" };
    return { buttonIcon: context.icons.sendIcon, buttonAlt: willQueue ? "Queue" : "Send" };
  }, [fullyDisabled, canStop, willQueue, context.icons]);

  const placeholder = fullyDisabled
    ? "Session is not available"
    : isCreating
      ? "Type a message (will be queued until session starts)..."
      : inProgress
        ? "Type a message (will be queued until agent is ready)..."
        : !chatReady
          ? "Type a message (will be queued)..."
          : "Send a message... (type / or @ for suggestions)";

  // ── Keyboard handler ───────────────────────────────────────────────
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (acOpen && acItems.length > 0) {
        if (e.key === "ArrowDown") { e.preventDefault(); setAcSelected((p) => Math.min(p + 1, acItems.length - 1)); return; }
        if (e.key === "ArrowUp") { e.preventDefault(); setAcSelected((p) => Math.max(p - 1, 0)); return; }
        if (e.key === "Enter" || e.key === "Tab") { e.preventDefault(); handleAcSelect(acItems[acSelected]); return; }
        if (e.key === "Escape") { e.preventDefault(); closeAutocomplete(); return; }
      }

      // Ctrl+Space to manually open agent autocomplete
      if (e.key === " " && e.ctrlKey) {
        e.preventDefault();
        const cursorPos = textareaRef.current?.selectionStart || 0;
        setAcTriggerPos(cursorPos);
        setAcType("agent");
        setAcFilter("");
        setAcSelected(0);
        setAcOpen(true);
        return;
      }

      if (e.key === "ArrowUp" && combinedHistory.length > 0) {
        const cursorPos = textareaRef.current?.selectionStart ?? 0;
        if (cursorPos === 0 || text === "") {
          e.preventDefault();
          const newIndex = historyIndex + 1;
          if (newIndex < combinedHistory.length) {
            if (historyIndex === -1) setDraftInput(text);
            setHistoryIndex(newIndex);
            const entry = combinedHistory[newIndex];
            setText(entry.text);
            setEditingQueuedId(entry.queuedId ?? null);
          }
          return;
        }
      }

      if (e.key === "ArrowDown" && historyIndex >= 0) {
        const cursorAtEnd = (textareaRef.current?.selectionStart ?? 0) === text.length;
        if (cursorAtEnd || text === "") {
          e.preventDefault();
          const newIndex = historyIndex - 1;
          setHistoryIndex(newIndex);
          if (newIndex < 0) { setText(draftInput); setEditingQueuedId(null); setDraftInput(""); }
          else { const entry = combinedHistory[newIndex]; setText(entry.text); setEditingQueuedId(entry.queuedId ?? null); }
          return;
        }
      }

      if (e.key === "Escape" && editingQueuedId) { e.preventDefault(); setText(draftInput); resetHistory(); return; }
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); if (canSendOrQueue) send(); }
    },
    [acOpen, acItems, acSelected, handleAcSelect, closeAutocomplete, combinedHistory, historyIndex, text, draftInput, editingQueuedId, canSendOrQueue, send],
  );

  const getTextareaStyle = () => {
    if (editingQueuedId) return "border-blue-400/50 bg-blue-50/30 dark:bg-blue-950/10";
    if (isRunActive) return "border-amber-400/50 bg-amber-50/30 dark:bg-amber-950/10";
    return "";
  };

  // ── Resume banner ──────────────────────────────────────────────────
  if (showResumeBanner) {
    return (
      <div className="copilotKitInputContainer">
        <button
          type="button"
          onClick={onResume}
          disabled={isResuming}
          className="w-full flex items-center justify-center gap-2 py-3 px-4 rounded-lg border border-border bg-muted/50 hover:bg-muted text-sm font-medium transition-colors disabled:opacity-60"
        >
          {isResuming ? (<><Loader2 className="h-4 w-4 animate-spin" />Resuming session...</>) : (<><Play className="h-4 w-4" />Resume session to continue chatting</>)}
        </button>
      </div>
    );
  }

  return (
    <div className="copilotKitInputContainer">
      {/* Phase status banners */}
      {isCreating && (
        <div className="flex items-center gap-2 px-3 py-1.5 mx-1 mb-1 rounded-md bg-muted text-xs text-muted-foreground">
          <Loader2 className="h-3 w-3 animate-spin" />
          Session is starting up. Messages will be queued.
        </div>
      )}
      {isTerminalState && (
        <div className="flex items-center gap-2 px-3 py-1.5 mx-1 mb-1 rounded-md bg-muted text-xs text-muted-foreground">
          Session has {sessionPhase.toLowerCase()}.
        </div>
      )}

      {/* Queued message bubbles */}
      {pendingQueuedMessages.length > 0 && (
        <div className="mx-1 mb-1 max-h-40 overflow-y-auto">
          {pendingQueuedMessages.map((msg) => (
            <QueuedMessageBubble
              key={msg.id}
              message={msg}
              onCancel={(id) => onCancelQueuedMessage?.(id)}
            />
          ))}
        </div>
      )}

      {/* Editing queued message indicator */}
      {editingQueuedId && (
        <div className="flex items-center gap-2 px-3 py-1.5 mx-1 mb-1 rounded-md bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 text-xs text-blue-700 dark:text-blue-300">
          <Pencil className="h-3 w-3" />
          Editing queued message
          <button onClick={() => { setText(draftInput); resetHistory(); }} className="ml-auto hover:text-blue-900 dark:hover:text-blue-100">
            <X className="h-3 w-3" />
          </button>
        </div>
      )}

      {/* Attachment previews */}
      {pendingAttachments.length > 0 && (
        <div className="mx-1 mb-1">
          <AttachmentPreview attachments={pendingAttachments} onRemove={(id) => setPendingAttachments((prev) => prev.filter((a) => a.id !== id))} />
        </div>
      )}

      <div className="copilotKitInput" style={{ position: "relative" }}>
        {/* Resize handle */}
        <div className="absolute -top-1.5 left-1/2 -translate-x-1/2 z-10 cursor-ns-resize px-3 py-1 group" onMouseDown={handleResizeStart} onTouchStart={handleResizeStart}>
          <div className="w-8 h-1 rounded-full bg-border group-hover:bg-muted-foreground/50 transition-colors" />
        </div>

        {/* Queue indicator */}
        {isRunActive && queuedCount > 0 && (
          <div className="absolute -top-5 left-1 flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
            <Clock className="h-3 w-3" />
            {queuedCount} message{queuedCount > 1 ? "s" : ""} queued
            {onClearQueue && (
              <button onClick={onClearQueue} className="ml-1 flex items-center gap-0.5 hover:text-amber-800 dark:hover:text-amber-200" title="Clear all queued messages">
                <X className="h-3 w-3" />Clear
              </button>
            )}
          </div>
        )}

        {/* Autocomplete popup */}
        {acOpen && (
          <AutocompletePopup items={acItems} selectedIndex={acSelected} onSelect={handleAcSelect} onHover={setAcSelected} popupRef={autocompleteRef} triggerType={acType} />
        )}

        <textarea
          ref={textareaRef}
          placeholder={placeholder}
          disabled={fullyDisabled}
          autoFocus={false}
          value={text}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          rows={1}
          style={{ resize: "none", height: textareaHeight }}
          className={getTextareaStyle()}
        />

        <div className="copilotKitInputControls">
          {/* Settings dropdown */}
          {onShowSystemMessagesChange && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button type="button" className="copilotKitInputControlButton" aria-label="Settings">
                  <Settings className="h-4 w-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Display Settings</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuCheckboxItem checked={showSystemMessages ?? false} onCheckedChange={(checked) => onShowSystemMessagesChange(checked)}>
                  Show system messages
                </DropdownMenuCheckboxItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}

          {/* Attach file button */}
          {onPasteImage && (
            <>
              <button type="button" onClick={() => fileInputRef.current?.click()} className="copilotKitInputControlButton" aria-label="Attach file" title="Attach file" style={{ opacity: fullyDisabled ? 0.4 : 1 }} disabled={fullyDisabled}>
                <Paperclip className="h-4 w-4" />
              </button>
              <input ref={fileInputRef} type="file" multiple accept="image/*,.pdf,.txt,.md,.json,.csv,.xml,.yaml,.yml,.log,.py,.js,.ts,.go,.java,.rs,.rb,.sh,.html,.css" className="hidden" onChange={handleFileSelect} />
            </>
          )}

          {/* Agents popover */}
          <Popover open={agentsPopoverOpen} onOpenChange={setAgentsPopoverOpen}>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm" className="h-7 gap-1.5 text-xs" disabled={agents.length === 0}>
                <Users className="h-3.5 w-3.5" />
                Agents
                {agents.length > 0 && (
                  <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">{agents.length}</Badge>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" side="top" className="w-[500px]">
              <ToolbarItemList
                items={agents}
                type="agent"
                onInsertAgent={(name) => {
                  setText((prev) => prev + `@${name} `);
                  setAgentsPopoverOpen(false);
                  textareaRef.current?.focus();
                }}
              />
            </PopoverContent>
          </Popover>

          {/* Commands popover */}
          <Popover open={commandsPopoverOpen} onOpenChange={setCommandsPopoverOpen}>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm" className="h-7 gap-1.5 text-xs" disabled={commands.length === 0}>
                <Terminal className="h-3.5 w-3.5" />
                Commands
                {commands.length > 0 && (
                  <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">{commands.length}</Badge>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" side="top" className="w-[500px]">
              <ToolbarItemList
                items={commands}
                type="command"
                onRunCommand={(slashCommand) => {
                  onCommandClick?.(slashCommand);
                  setCommandsPopoverOpen(false);
                }}
              />
            </PopoverContent>
          </Popover>

          <div style={{ flexGrow: 1 }} />

          {/* Send as new (when editing queued) */}
          {editingQueuedId && (
            <button onClick={handleSendAsNew} disabled={!hasContent} className="text-xs text-muted-foreground hover:text-foreground disabled:opacity-50 mr-1">
              Send as new
            </button>
          )}

          <button
            disabled={sendDisabled}
            onClick={canStop ? onStop : () => send()}
            data-copilotkit-in-progress={inProgress}
            className="copilotKitInputControlButton"
            aria-label={editingQueuedId ? "Update" : buttonAlt}
          >
            {buttonIcon}
          </button>
        </div>
      </div>
    </div>
  );
}
