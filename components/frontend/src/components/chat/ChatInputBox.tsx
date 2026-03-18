"use client";

import React, { useState, useRef, useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Loader2,
  Terminal,
  Users,
  Clock,
  X,
  Pencil,
  Plus,
  GitBranch,
  Upload,
  ArrowUp,
} from "lucide-react";
import { Separator } from "@/components/ui/separator";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { toast } from "sonner";
import { useResizeTextarea } from "@/hooks/use-resize-textarea";
import { useAutocomplete } from "@/hooks/use-autocomplete";
import type { AutocompleteAgent, AutocompleteCommand } from "@/hooks/use-autocomplete";
import { AutocompletePopover } from "./AutocompletePopover";
import { AttachmentPreview, type PendingAttachment } from "./AttachmentPreview";

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

export type ChatInputBoxProps = {
  value: string;
  onChange: (value: string) => void;
  onSend: () => Promise<void>;
  onInterrupt: () => Promise<void>;
  onPasteImage?: (file: File) => Promise<void>;
  isRunActive?: boolean;
  isSending?: boolean;
  disabled?: boolean;
  placeholder?: string;
  agents?: AutocompleteAgent[];
  commands?: AutocompleteCommand[];
  onCommandClick?: (slashCommand: string) => void;
  queuedCount?: number;
  sessionPhase?: string;
  onContinue?: () => void;
  messageHistory?: string[];
  queuedMessageHistory?: Array<{ id: string; content: string }>;
  onUpdateQueuedMessage?: (messageId: string, newContent: string) => void;
  onCancelQueuedMessage?: (messageId: string) => void;
  onClearQueue?: () => void;
  onAddRepository?: () => void;
  onUploadFile?: () => void;
  workflowSlot?: React.ReactNode;
};

type HistoryEntry = {
  text: string;
  queuedId?: string;
};

/** Generate a preview data-URL for image files. */
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

// ---------------------------------------------------------------------------
// Toolbar item-list popover — shared between Agents and Commands
// ---------------------------------------------------------------------------

type ToolbarItemListProps = {
  items: AutocompleteAgent[] | AutocompleteCommand[];
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
            const agent = isAgent ? (item as AutocompleteAgent) : null;
            const cmd = !isAgent ? (item as AutocompleteCommand) : null;
            const shortName = isAgent ? agent!.name.split(" - ")[0] : "";

            return (
              <div key={item.id} className="p-3 rounded-md border bg-muted/30">
                <div className="flex items-center justify-between mb-1">
                  <h3 className="text-sm font-bold">{item.name}</h3>
                  {isAgent ? (
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-shrink-0 h-7 text-xs"
                      onClick={() => onInsertAgent?.(shortName)}
                    >
                      @{shortName}
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-shrink-0 h-7 text-xs"
                      onClick={() => onRunCommand?.(cmd!.slashCommand)}
                    >
                      Run {cmd!.slashCommand}
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

// ---------------------------------------------------------------------------
// ChatInputBox
// ---------------------------------------------------------------------------

export const ChatInputBox: React.FC<ChatInputBoxProps> = ({
  value,
  onChange,
  onSend,
  onInterrupt,
  onPasteImage,
  isRunActive = false,
  isSending = false,
  disabled = false,
  placeholder,
  agents = [],
  commands = [],
  onCommandClick,
  queuedCount = 0,
  sessionPhase = "",
  onContinue,
  messageHistory = [],
  queuedMessageHistory = [],
  onUpdateQueuedMessage,
  onCancelQueuedMessage,
  onClearQueue,
  onAddRepository,
  onUploadFile,
  workflowSlot,
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { textareaHeight, handleResizeStart } = useResizeTextarea();

  // Phase-derived state
  const isTerminalState = ["Completed", "Failed", "Stopped"].includes(sessionPhase);
  const isCreating = ["Creating", "Pending"].includes(sessionPhase);

  // Autocomplete (consolidated via hook)
  const autocomplete = useAutocomplete({ agents, commands });

  // Attachment state
  const [pendingAttachments, setPendingAttachments] = useState<PendingAttachment[]>([]);

  // Popover states
  const [agentsPopoverOpen, setAgentsPopoverOpen] = useState(false);
  const [commandsPopoverOpen, setCommandsPopoverOpen] = useState(false);

  // Interrupting state
  const [interrupting, setInterrupting] = useState(false);

  // Prompt history state
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [draftInput, setDraftInput] = useState("");
  const [editingQueuedId, setEditingQueuedId] = useState<string | null>(null);

  // Combined history: queued (unsent) first, then sent messages — all newest-first
  const combinedHistory = useMemo<HistoryEntry[]>(() => {
    const queued = queuedMessageHistory.map((m) => ({ text: m.content, queuedId: m.id }));
    const sent = messageHistory.map((text) => ({ text }));
    return [...queued, ...sent];
  }, [queuedMessageHistory, messageHistory]);

  const resetHistory = () => {
    setHistoryIndex(-1);
    setDraftInput("");
    setEditingQueuedId(null);
  };

  // Dynamic placeholder
  const getPlaceholder = () => {
    if (placeholder) return placeholder;
    if (isTerminalState) return "Type a message to resume this session...";
    if (isCreating) return "Type a message (will be queued until session starts)...";
    if (isRunActive) return "Type a message (will be queued)...";
    return "Type a message... (\u{1F4CE} attach \u00B7 \u2318V paste \u00B7 \u2191 history \u00B7 Enter send \u00B7 Shift+Enter newline)";
  };

  // Handle paste events for images
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
            toast.error("File too large", {
              description: `Maximum file size is 10MB. Your file is ${(file.size / (1024 * 1024)).toFixed(1)}MB.`,
            });
            continue;
          }

          const renamedFile =
            file.name === "image.png" || file.name === "image.jpg"
              ? new File(
                  [file],
                  `paste-${new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19)}.png`,
                  { type: file.type }
                )
              : file;

          const preview = await generatePreview(renamedFile);
          setPendingAttachments((prev) => [
            ...prev,
            { id: makeAttachmentId(), file: renamedFile, preview },
          ]);
        }
      }
    },
    [onPasteImage]
  );

  const handleRemoveAttachment = (attachmentId: string) => {
    setPendingAttachments((prev) => prev.filter((a) => a.id !== attachmentId));
  };

  // Handle native file picker selection
  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(e.target.files || []);
      for (const file of files) {
        if (file.size > MAX_FILE_SIZE) {
          toast.error("File too large", {
            description: `Maximum file size is 10MB. "${file.name}" is ${(file.size / (1024 * 1024)).toFixed(1)}MB.`,
          });
          continue;
        }

        const preview = await generatePreview(file);
        setPendingAttachments((prev) => [
          ...prev,
          { id: makeAttachmentId(), file, preview },
        ]);
      }
      e.target.value = "";
    },
    []
  );

  // Handle input change — delegate autocomplete detection to hook
  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    onChange(newValue);

    if (historyIndex >= 0) {
      setHistoryIndex(-1);
      setDraftInput("");
    }

    autocomplete.handleInputChange(newValue, e.target.selectionStart);
  };

  // Handle key events
  const handleKeyDown = async (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Let autocomplete hook handle its keys first
    if (autocomplete.handleKeyDown(e)) {
      // If Enter/Tab was pressed, perform the selection
      if ((e.key === "Enter" || e.key === "Tab") && autocomplete.filteredItems.length > 0) {
        const item = autocomplete.filteredItems[autocomplete.selectedIndex];
        const cursorPos = textareaRef.current?.selectionStart ?? value.length;
        const newCursorPos = autocomplete.select(item, value, cursorPos, onChange);
        setTimeout(() => {
          if (textareaRef.current) {
            textareaRef.current.selectionStart = newCursorPos;
            textareaRef.current.selectionEnd = newCursorPos;
            textareaRef.current.focus();
          }
        }, 0);
      }
      return;
    }

    // Prompt history: Up arrow
    if (e.key === "ArrowUp" && combinedHistory.length > 0) {
      const cursorPos = textareaRef.current?.selectionStart ?? 0;
      if (cursorPos === 0 || value === "") {
        e.preventDefault();
        const newIndex = historyIndex + 1;
        if (newIndex < combinedHistory.length) {
          if (historyIndex === -1) setDraftInput(value);
          setHistoryIndex(newIndex);
          const entry = combinedHistory[newIndex];
          onChange(entry.text);
          setEditingQueuedId(entry.queuedId ?? null);
        }
        return;
      }
    }

    // Prompt history: Down arrow
    if (e.key === "ArrowDown" && historyIndex >= 0) {
      const cursorAtEnd = (textareaRef.current?.selectionStart ?? 0) === value.length;
      if (cursorAtEnd || value === "") {
        e.preventDefault();
        const newIndex = historyIndex - 1;
        setHistoryIndex(newIndex);
        if (newIndex < 0) {
          onChange(draftInput);
          setEditingQueuedId(null);
          setDraftInput("");
        } else {
          const entry = combinedHistory[newIndex];
          onChange(entry.text);
          setEditingQueuedId(entry.queuedId ?? null);
        }
        return;
      }
    }

    // Escape to cancel editing queued message
    if (e.key === "Escape" && editingQueuedId) {
      e.preventDefault();
      onChange(draftInput);
      resetHistory();
      return;
    }

    // Ctrl+Space to manually trigger autocomplete
    if (e.key === " " && e.ctrlKey) {
      e.preventDefault();
      const cursorPos = textareaRef.current?.selectionStart || 0;
      autocomplete.open("agent", cursorPos);
      return;
    }

    // Enter to send (or update queued message)
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      await handleSendOrUpdate();
    }
  };

  // Upload pending attachments
  const uploadPendingAttachments = async (): Promise<boolean> => {
    const toUpload = pendingAttachments.filter((a) => !a.uploading && !a.error);
    if (toUpload.length === 0 || !onPasteImage) return true;

    for (const attachment of toUpload) {
      setPendingAttachments((prev) =>
        prev.map((a) => (a.id === attachment.id ? { ...a, uploading: true } : a))
      );
      try {
        await onPasteImage(attachment.file);
        setPendingAttachments((prev) =>
          prev.map((a) => (a.id === attachment.id ? { ...a, uploading: false } : a))
        );
      } catch {
        setPendingAttachments((prev) =>
          prev.map((a) =>
            a.id === attachment.id ? { ...a, uploading: false, error: "Upload failed" } : a
          )
        );
        return false;
      }
    }
    return true;
  };

  const hasContent = value.trim() || pendingAttachments.length > 0;

  const handleSendOrUpdate = async () => {
    if (!hasContent || isSending) return;

    // If editing a queued message, update it in place
    if (editingQueuedId && onUpdateQueuedMessage) {
      onUpdateQueuedMessage(editingQueuedId, value.trim());
      onChange("");
      resetHistory();
      setPendingAttachments([]);
      toast("Queued message updated", { description: "The queued message has been updated." });
      return;
    }

    const uploaded = await uploadPendingAttachments();
    if (!uploaded) return;

    if (isRunActive) {
      toast("Message queued", { description: "Your message will be sent when the agent is ready." });
    }
    await onSend();
    resetHistory();
    setPendingAttachments([]);
  };

  const handleSendAsNew = async () => {
    if (!value.trim() || isSending || !editingQueuedId) return;
    onCancelQueuedMessage?.(editingQueuedId);
    resetHistory();

    const uploaded = await uploadPendingAttachments();
    if (!uploaded) return;

    if (isRunActive) {
      toast("Message queued", {
        description: "Original cancelled. New message will be sent when the agent is ready.",
      });
    }
    await onSend();
    setPendingAttachments([]);
  };

  const handleInterrupt = async () => {
    setInterrupting(true);
    try {
      await onInterrupt();
    } finally {
      setInterrupting(false);
    }
  };

  return (
    <div className="bg-card">
      <div className="px-4 pt-2 pb-2 space-y-1.5">
        {/* Phase status banner */}
        {isCreating && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-muted text-xs text-muted-foreground">
            <Loader2 className="h-3 w-3 animate-spin" />
            Session is starting up. Messages will be queued.
          </div>
        )}
        {isTerminalState && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-muted text-xs text-muted-foreground">
            Session has {sessionPhase.toLowerCase()}.
            {onContinue && (
              <button type="button" onClick={onContinue} className="text-link hover:underline font-medium">
                Resume session
              </button>
            )}
          </div>
        )}

        {/* Editing queued message indicator */}
        {editingQueuedId && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 text-xs text-blue-700 dark:text-blue-300">
            <Pencil className="h-3 w-3" />
            Editing queued message
            <button
              type="button"
              onClick={() => { onChange(draftInput); resetHistory(); }}
              className="ml-auto hover:text-blue-900 dark:hover:text-blue-100"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {/* Attachment preview */}
        <AttachmentPreview attachments={pendingAttachments} onRemove={handleRemoveAttachment} />

        {/* Single bordered container for textarea + toolbar */}
        <div className={`relative border rounded-lg bg-background shadow-sm focus-within:ring-1 focus-within:ring-ring ${editingQueuedId ? "border-blue-400/50" : isRunActive ? "border-amber-400/50" : ""}`}>
          {/* Resize handle */}
          <div
            className="absolute -top-1.5 left-1/2 -translate-x-1/2 z-10 cursor-ns-resize px-3 py-1 group"
            onMouseDown={handleResizeStart}
            onTouchStart={handleResizeStart}
          >
            <div className="w-8 h-1 rounded-full bg-border group-hover:bg-muted-foreground/50 transition-colors" />
          </div>

          {/* Queue indicator with clear button */}
          {isRunActive && queuedCount > 0 && (
            <div className="absolute -top-6 left-0 flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
              <Clock className="h-3 w-3" />
              {queuedCount} message{queuedCount > 1 ? "s" : ""} queued
              {onClearQueue && (
                <button
                  type="button"
                  onClick={onClearQueue}
                  className="ml-1 flex items-center gap-0.5 hover:text-amber-800 dark:hover:text-amber-200"
                  title="Clear all queued messages"
                >
                  <X className="h-3 w-3" />
                  Clear
                </button>
              )}
            </div>
          )}

          <textarea
            ref={textareaRef}
            className={`w-full border-0 rounded-t-lg p-2 text-sm transition-colors resize-none focus:outline-none focus-visible:ring-0 focus-visible:ring-offset-0 ${editingQueuedId ? "bg-blue-50/30 dark:bg-blue-950/10" : isRunActive ? "bg-amber-50/30 dark:bg-amber-950/10" : "bg-transparent"}`}
            placeholder={getPlaceholder()}
            value={value}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            style={{ height: textareaHeight }}
            disabled={disabled}
          />

          {/* Autocomplete popup */}
          <AutocompletePopover
            open={autocomplete.isOpen}
            type={autocomplete.type}
            filter={autocomplete.filter}
            selectedIndex={autocomplete.selectedIndex}
            items={autocomplete.filteredItems}
            onSelect={(item) => {
              const cursorPos = textareaRef.current?.selectionStart ?? value.length;
              const newCursorPos = autocomplete.select(item, value, cursorPos, onChange);
              setTimeout(() => {
                if (textareaRef.current) {
                  textareaRef.current.selectionStart = newCursorPos;
                  textareaRef.current.selectionEnd = newCursorPos;
                  textareaRef.current.focus();
                }
              }, 0);
            }}
            onSelectedIndexChange={autocomplete.setSelectedIndex}
            onClose={autocomplete.close}
          />

          {/* Toolbar — inside the container */}
          <div className="flex items-center justify-between px-2 py-2">
            <div className="flex items-center gap-2">
              {/* Add context dropdown */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 w-7 p-0"
                  >
                    <Plus className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" side="top">
                  <DropdownMenuItem onClick={() => onAddRepository?.()}>
                    <GitBranch className="w-4 h-4 mr-2" />
                    Add Repository
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => onUploadFile?.()}>
                    <Upload className="w-4 h-4 mr-2" />
                    Upload File
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                accept="image/*,.pdf,.txt,.md,.json,.csv,.xml,.yaml,.yml,.log,.py,.js,.ts,.go,.java,.rs,.rb,.sh,.html,.css"
                className="hidden"
                onChange={handleFileSelect}
              />

              <Separator orientation="vertical" className="h-5" />

              {/* Agents popover */}
              <Popover open={agentsPopoverOpen} onOpenChange={setAgentsPopoverOpen}>
                <PopoverTrigger asChild>
                  <Button variant="outline" size="sm" className="h-7 gap-1.5" disabled={agents.length === 0}>
                    <Users className="h-3.5 w-3.5" />
                    Agents
                    {agents.length > 0 && (
                      <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">
                        {agents.length}
                      </Badge>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" side="top" className="w-[500px]">
                  <ToolbarItemList
                    items={agents}
                    type="agent"
                    onInsertAgent={(name) => {
                      onChange(value + `@${name} `);
                      setAgentsPopoverOpen(false);
                      textareaRef.current?.focus();
                    }}
                  />
                </PopoverContent>
              </Popover>

              {/* Commands popover */}
              <Popover open={commandsPopoverOpen} onOpenChange={setCommandsPopoverOpen}>
                <PopoverTrigger asChild>
                  <Button variant="outline" size="sm" className="h-7 gap-1.5" disabled={commands.length === 0}>
                    <Terminal className="h-3.5 w-3.5" />
                    Commands
                    {commands.length > 0 && (
                      <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">
                        {commands.length}
                      </Badge>
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

            </div>

            {/* Right side: Workflow selector + Send/Stop buttons */}
            <div className="flex gap-2 items-center">
              {workflowSlot}

              {isRunActive ? (
                <Button variant="destructive" size="sm" onClick={handleInterrupt} disabled={interrupting}>
                  {interrupting && <Loader2 className="w-3 h-3 mr-1 animate-spin" />}
                  Stop
                </Button>
              ) : editingQueuedId ? (
                <>
                  <button
                    type="button"
                    onClick={handleSendAsNew}
                    disabled={!hasContent || isSending}
                    className="text-xs text-muted-foreground hover:text-foreground disabled:opacity-50"
                  >
                    Send as new
                  </button>
                  <Button size="sm" onClick={handleSendOrUpdate} disabled={!hasContent || isSending || disabled}>
                    {isSending && <Loader2 className="w-3 h-3 mr-1 animate-spin" />}
                    Update
                  </Button>
                </>
              ) : (
                <Button
                  size="icon"
                  className="h-8 w-8 rounded-full bg-primary hover:bg-primary/90 text-primary-foreground"
                  onClick={handleSendOrUpdate}
                  disabled={!hasContent || isSending || disabled}
                >
                  {isSending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <ArrowUp className="h-4 w-4" />
                  )}
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ChatInputBox;
