"use client";

import React, { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { MessageSquare, Settings } from "lucide-react";
import { StreamMessage } from "@/components/ui/stream-message";
import { LoadingDots } from "@/components/ui/message";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ChatInputBox } from "@/components/chat/ChatInputBox";
import { QueuedMessageBubble } from "@/components/chat/QueuedMessageBubble";
import { WelcomeExperience } from "@/components/chat/WelcomeExperience";
import type { AgenticSession, MessageObject, ToolUseMessages } from "@/types/agentic-session";
import type { WorkflowMetadata } from "@/app/projects/[name]/sessions/[sessionName]/lib/types";
import type { QueuedMessageItem } from "@/hooks/use-session-queue";

export type MessagesTabProps = {
  session: AgenticSession;
  streamMessages: Array<MessageObject | ToolUseMessages>;
  chatInput: string;
  setChatInput: (v: string) => void;
  onSendChat: () => Promise<void>;
  onInterrupt: () => Promise<void>;
  onGoToResults?: () => void;
  onContinue: () => void;
  workflowMetadata?: WorkflowMetadata;
  onCommandClick?: (slashCommand: string) => void;
  isRunActive?: boolean;  // Track if agent is actively processing
  showWelcomeExperience?: boolean;
  welcomeExperienceComponent?: React.ReactNode;
  activeWorkflow?: string | null;  // Track if workflow has been selected
  userHasInteracted?: boolean;  // Track if user has sent any messages
  queuedMessages?: QueuedMessageItem[];  // Messages queued while session wasn't running
  hasRealMessages?: boolean;  // Track if there are real user/agent messages
  onCancelQueuedMessage?: (messageId: string) => void;  // Cancel a queued message
  onUpdateQueuedMessage?: (messageId: string, newContent: string) => void;  // Update a queued message
  onPasteImage?: (file: File) => Promise<void>;  // Handle pasted images
  onClearQueue?: () => void;  // Clear all queued messages
};


const MessagesTab: React.FC<MessagesTabProps> = ({ session, streamMessages, chatInput, setChatInput, onSendChat, onInterrupt, onGoToResults, onContinue, workflowMetadata, onCommandClick, isRunActive = false, showWelcomeExperience, welcomeExperienceComponent, activeWorkflow, userHasInteracted = false, queuedMessages = [], hasRealMessages = false, onCancelQueuedMessage, onUpdateQueuedMessage, onPasteImage, onClearQueue }) => {
  const [sendingChat, setSendingChat] = useState(false);
  const [showSystemMessages, setShowSystemMessages] = useState(false);
  const [waitingDotCount, setWaitingDotCount] = useState(0);

  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);

  const phase = session?.status?.phase || "";
  const isInteractive = session?.spec?.interactive;

  // Determine if session is in a terminal state
  const isTerminalState = ["Completed", "Failed", "Stopped"].includes(phase);

  // Filter out system messages unless showSystemMessages is true
  const filteredMessages = streamMessages.filter((msg) => {
    if (showSystemMessages) return true;

    // Hide system_message type by default
    // Check if msg has a type property and if it's a system_message
    if ('type' in msg && msg.type === "system_message") {
      return false;
    }

    return true;
  });

  // Check if user is scrolled to the bottom
  const checkIfAtBottom = () => {
    const container = messagesContainerRef.current;
    if (!container) return true;

    // For normal scroll (not reversed), we check if scrollTop + clientHeight >= scrollHeight
    const threshold = 50; // pixels from bottom to still consider "at bottom"
    const isBottom = container.scrollHeight - container.scrollTop - container.clientHeight < threshold;
    return isBottom;
  };

  // Handle scroll event to track if user is at bottom
  const handleScroll = () => {
    setIsAtBottom(checkIfAtBottom());
  };

  // Scroll to bottom function - only scrolls the messages container, not the whole page
  const scrollToBottom = () => {
    const container = messagesContainerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  };

  // Auto-scroll to bottom when new messages arrive, but only if user was already at bottom
  useEffect(() => {
    if (isAtBottom) {
      scrollToBottom();
    }
  }, [filteredMessages, isAtBottom]);

  // Initial scroll to bottom on mount
  useEffect(() => {
    scrollToBottom();
  }, []);

  // Animate dots for "Please wait one moment" message
  useEffect(() => {
    const unsentCount = queuedMessages.filter(m => !m.sentAt).length;
    if (unsentCount === 0) return;

    const interval = setInterval(() => {
      setWaitingDotCount((prev) => (prev + 1) % 4); // Cycles 0, 1, 2, 3
    }, 500); // Change dot every 500ms

    return () => clearInterval(interval);
  }, [queuedMessages]);

  const handleSendChat = async () => {
    setSendingChat(true);
    try {
      await onSendChat();
    } finally {
      setSendingChat(false);
    }
  };

  // Handle welcome prompt selection
  const handleWelcomePromptSelect = useCallback((prompt: string) => {
    setChatInput(prompt);
  }, [setChatInput]);

  // Calculate pending queued messages count
  const pendingQueuedCount = queuedMessages.filter(m => !m.sentAt).length;

  // Prompt history: extract user messages from stream (newest first)
  const messageHistory = useMemo(() => {
    return streamMessages
      .filter((m) => "type" in m && m.type === "user_message")
      .map((m) => {
        const content = (m as { content?: unknown }).content;
        if (typeof content === "string") return content;
        if (content && typeof content === "object" && "text" in content) return (content as { text: string }).text;
        return "";
      })
      .filter(Boolean)
      .reverse();
  }, [streamMessages]);

  // Queued message history: unsent queued messages (newest first)
  const queuedMessageHistoryItems = useMemo(() => {
    return (queuedMessages || [])
      .filter((m) => !m.sentAt)
      .map((m) => ({ id: m.id, content: m.content }))
      .reverse();
  }, [queuedMessages]);

  // Determine if we should show messages
  // Messages should be hidden until workflow is selected OR user sends a message when welcome experience is active
  // BUT always show messages if there are real messages (e.g., when loading an existing session with messages)
  const shouldShowMessages = !showWelcomeExperience || activeWorkflow || userHasInteracted || hasRealMessages;

  return (
    <div className="flex flex-col h-full">
      <div
        ref={messagesContainerRef}
        onScroll={handleScroll}
        className="flex-1 flex flex-col gap-2 overflow-y-auto p-3 scrollbar-thin"
      >
        {/* Show welcome experience if active */}
        {showWelcomeExperience && !userHasInteracted && !hasRealMessages && (
          welcomeExperienceComponent || (
            <WelcomeExperience
              onSelectPrompt={handleWelcomePromptSelect}
              visible={!activeWorkflow}
            />
          )
        )}

        {/* Show filtered messages only if workflow is selected or welcome experience is not shown */}
        {shouldShowMessages && filteredMessages.map((m, idx) => (
          <StreamMessage key={`sm-${idx}`} message={m} isNewest={idx === filteredMessages.length - 1} onGoToResults={onGoToResults} />
        ))}

        {/* Show queued messages with distinct styling and cancel button */}
        {queuedMessages.filter(m => !m.sentAt).map((item) => (
          <QueuedMessageBubble
            key={`queued-${item.id}`}
            message={item}
            onCancel={(messageId) => onCancelQueuedMessage?.(messageId)}
          />
        ))}

        {/* Show "Please wait" message while queued messages are waiting */}
        {queuedMessages.filter(m => !m.sentAt).length > 0 && (
          <div className="mb-4 mt-2">
            <div className="flex space-x-3 items-start">
              {/* Avatar */}
              <div className="flex-shrink-0">
                <div className="w-8 h-8 rounded-full flex items-center justify-center bg-blue-600">
                  <span className="text-white text-xs font-semibold">AI</span>
                </div>
              </div>

              {/* Message Content */}
              <div className="flex-1 min-w-0">
                {/* Timestamp */}
                <div className="text-[10px] text-muted-foreground/60 mb-1">just now</div>
                <div className="rounded-lg bg-card">
                  {/* Content */}
                  <p className="text-sm text-muted-foreground leading-relaxed mb-[0.2rem]">
                    Please wait one moment{".".repeat(waitingDotCount)}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Show loading indicator when agent is actively processing */}
        {shouldShowMessages && isRunActive && filteredMessages.length > 0 && (
          <div className="pl-12 pr-4 py-2">
            <LoadingDots />
          </div>
        )}

        {/* Show empty state only if no welcome experience and no messages */}
        {!showWelcomeExperience && filteredMessages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No messages yet</p>
            <p className="text-xs mt-1">
              {isInteractive
                ? isTerminalState
                  ? `Session has ${phase.toLowerCase()}.`
                  : "Start by sending a message below."
                : "This session is not interactive."}
            </p>
          </div>
        )}
      </div>

      {/* Settings for non-interactive sessions with messages */}
      {!isInteractive && filteredMessages.length > 0 && (
        <div className="sticky bottom-0 border-t bg-muted/50">
          <div className="p-3">
            <div className="flex items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                    <Settings className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuCheckboxItem
                    checked={showSystemMessages}
                    onCheckedChange={setShowSystemMessages}
                  >
                    Show system messages
                  </DropdownMenuCheckboxItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <p className="text-sm text-muted-foreground">Non-interactive session</p>
            </div>
          </div>
        </div>
      )}

      {isInteractive && (
        <ChatInputBox
          value={chatInput}
          onChange={setChatInput}
          onSend={handleSendChat}
          onInterrupt={onInterrupt}
          onPasteImage={onPasteImage}
          isRunActive={isRunActive}
          isSending={sendingChat}
          agents={workflowMetadata?.agents}
          commands={workflowMetadata?.commands}
          onCommandClick={onCommandClick}
          showSystemMessages={showSystemMessages}
          onShowSystemMessagesChange={setShowSystemMessages}
          queuedCount={pendingQueuedCount}
          sessionPhase={phase}
          onContinue={onContinue}
          messageHistory={messageHistory}
          queuedMessageHistory={queuedMessageHistoryItems}
          onUpdateQueuedMessage={onUpdateQueuedMessage}
          onCancelQueuedMessage={onCancelQueuedMessage}
          onClearQueue={onClearQueue}
        />
      )}
    </div>
  );
};

export default MessagesTab;
