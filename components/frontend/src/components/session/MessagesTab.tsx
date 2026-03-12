"use client";

import React, { useState, useRef, useEffect, useMemo, useLayoutEffect, useCallback } from "react";
import { MessageSquare } from "lucide-react";
import { StreamMessage } from "@/components/ui/stream-message";
import { LoadingDots } from "@/components/ui/message";
import { Button } from "@/components/ui/button";
import { ChatInputBox } from "@/components/chat/ChatInputBox";
import { QueuedMessageBubble } from "@/components/chat/QueuedMessageBubble";
import type { AgenticSession, MessageObject, ToolUseMessages, HierarchicalToolMessage } from "@/types/agentic-session";
import type { WorkflowMetadata } from "@/app/projects/[name]/sessions/[sessionName]/lib/types";
import type { QueuedMessageItem } from "@/hooks/use-session-queue";

/** Maximum number of messages rendered at once. Older messages are loaded on demand. */
const MAX_VISIBLE_MESSAGES = 100;

/** Derive a stable React key for any message variant. */
function getMessageKey(m: MessageObject | ToolUseMessages | HierarchicalToolMessage, idx: number): string {
  if ('id' in m && m.id) return m.id;
  if ('toolUseBlock' in m && m.toolUseBlock?.id) return `tool-${m.toolUseBlock.id}`;
  // Both MessageObject and HierarchicalToolMessage carry type+timestamp; the `in` guard is sufficient.
  if ('type' in m && 'timestamp' in m) return `${m.type}-${m.timestamp}-${idx}`;
  // Last resort: index within the visible window. This key shifts when the window expands
  // (Load earlier), but only affects messages with no other stable identifier.
  return `sm-${idx}`;
}

export type MessagesTabProps = {
  session: AgenticSession;
  streamMessages: Array<MessageObject | ToolUseMessages | HierarchicalToolMessage>;
  chatInput: string;
  setChatInput: (v: string) => void;
  onSendChat: () => Promise<void>;
  onSendToolAnswer?: (formattedAnswer: string) => Promise<void>;
  onInterrupt: () => Promise<void>;
  onGoToResults?: () => void;
  onContinue: () => void;
  workflowMetadata?: WorkflowMetadata;
  onCommandClick?: (slashCommand: string) => void;
  isRunActive?: boolean;
  showWelcomeExperience?: boolean;
  welcomeExperienceComponent?: React.ReactNode;
  activeWorkflow?: string | null;
  userHasInteracted?: boolean;
  queuedMessages?: QueuedMessageItem[];
  hasRealMessages?: boolean;
  onCancelQueuedMessage?: (messageId: string) => void;
  onUpdateQueuedMessage?: (messageId: string, newContent: string) => void;
  onPasteImage?: (file: File) => Promise<void>;
  onClearQueue?: () => void;
  agentName?: string;
};


const MessagesTab: React.FC<MessagesTabProps> = ({ session, streamMessages, chatInput, setChatInput, onSendChat, onSendToolAnswer, onInterrupt, onGoToResults, onContinue, workflowMetadata, onCommandClick, isRunActive = false, showWelcomeExperience, welcomeExperienceComponent, activeWorkflow, userHasInteracted = false, queuedMessages = [], hasRealMessages = false, onCancelQueuedMessage, onUpdateQueuedMessage, onPasteImage, onClearQueue, agentName }) => {
  const [sendingChat, setSendingChat] = useState(false);
  const [showSystemMessages, setShowSystemMessages] = useState(false);
  const [waitingDotCount, setWaitingDotCount] = useState(0);

  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);
  // How many messages (counting from the end) are currently rendered.
  const [loadedMessageCount, setLoadedMessageCount] = useState(MAX_VISIBLE_MESSAGES);
  // Refs for scroll-position preservation when loading earlier messages.
  const prevScrollHeightRef = useRef(0);
  const prevScrollTopRef = useRef(0);
  const preservingScrollRef = useRef(false);

  const phase = session?.status?.phase || "";

  const isTerminalState = ["Completed", "Failed", "Stopped"].includes(phase);
  const isCreating = ["Creating", "Pending"].includes(phase);

  const filteredMessages = streamMessages.filter((msg) => {
    if (showSystemMessages) return true;
    if ('type' in msg && msg.type === "system_message") return false;
    return true;
  });

  // Only render the latest N messages; older ones are revealed via "Load earlier" button.
  const visibleMessages = useMemo(() => {
    if (filteredMessages.length <= loadedMessageCount) return filteredMessages;
    return filteredMessages.slice(filteredMessages.length - loadedMessageCount);
  }, [filteredMessages, loadedMessageCount]);

  const hasMoreMessages = filteredMessages.length > loadedMessageCount;

  const checkIfAtBottom = () => {
    const container = messagesContainerRef.current;
    if (!container) return true;
    const threshold = 50;
    return container.scrollHeight - container.scrollTop - container.clientHeight < threshold;
  };

  const handleScroll = () => {
    const bottom = checkIfAtBottom();
    // Avoid a re-render when the value hasn't changed.
    setIsAtBottom((prev) => (prev === bottom ? prev : bottom));
  };

  const scrollToBottom = () => {
    const container = messagesContainerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  };

  // Load earlier messages and preserve the user's visual scroll position.
  const loadEarlierMessages = useCallback(() => {
    const container = messagesContainerRef.current;
    if (container) {
      // Capture BEFORE state update so we have the pre-render values.
      prevScrollHeightRef.current = container.scrollHeight;
      prevScrollTopRef.current = container.scrollTop;
      preservingScrollRef.current = true;
    }
    setLoadedMessageCount((prev) => prev + MAX_VISIBLE_MESSAGES);
  }, []);

  // After React commits the expanded list, forcibly set scrollTop so the
  // viewport stays anchored to the same content regardless of whether the
  // browser's native scroll-anchoring has already adjusted it.
  useLayoutEffect(() => {
    if (!preservingScrollRef.current) return;
    preservingScrollRef.current = false;
    const container = messagesContainerRef.current;
    if (container) {
      const scrollDelta = container.scrollHeight - prevScrollHeightRef.current;
      container.scrollTop = prevScrollTopRef.current + scrollDelta;
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- refs are stable; loadedMessageCount is the intended trigger
  }, [loadedMessageCount]);

  useEffect(() => {
    if (isAtBottom) {
      scrollToBottom();
    }
  }, [filteredMessages, isAtBottom]);

  useEffect(() => {
    scrollToBottom();
  }, []);

  useEffect(() => {
    const unsentCount = queuedMessages.filter(m => !m.sentAt).length;
    if (unsentCount === 0) return;

    const interval = setInterval(() => {
      setWaitingDotCount((prev) => (prev + 1) % 4);
    }, 500);

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

  const queuedMessageHistoryItems = useMemo(() => {
    return (queuedMessages || [])
      .filter((m) => !m.sentAt)
      .map((m) => ({ id: m.id, content: m.content }))
      .reverse();
  }, [queuedMessages]);

  const pendingQueuedCount = queuedMessages.filter(m => !m.sentAt).length;

  const shouldShowMessages = !showWelcomeExperience || activeWorkflow || userHasInteracted || hasRealMessages;

  return (
    <div className="flex flex-col h-full">
      <div
        ref={messagesContainerRef}
        onScroll={handleScroll}
        className="flex-1 flex flex-col gap-2 overflow-y-auto p-3 scrollbar-thin"
      >
        {showWelcomeExperience && welcomeExperienceComponent}

        {shouldShowMessages && hasMoreMessages && (
          <div className="flex justify-center py-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={loadEarlierMessages}
              className="text-xs text-muted-foreground underline underline-offset-2"
            >
              Load earlier messages ({filteredMessages.length - loadedMessageCount} hidden)
            </Button>
          </div>
        )}

        {shouldShowMessages && visibleMessages.map((m, idx) => (
          <StreamMessage
            key={getMessageKey(m, idx)}
            message={m}
            isNewest={idx === visibleMessages.length - 1}
            onGoToResults={onGoToResults}
            agentName={agentName}
            onSubmitAnswer={onSendToolAnswer}
          />
        ))}

        {/* Queued messages with cancel buttons */}
        {queuedMessages.filter(m => !m.sentAt).map((item) => (
          <QueuedMessageBubble
            key={`queued-${item.id}`}
            message={item}
            onCancel={onCancelQueuedMessage || (() => {})}
          />
        ))}

        {pendingQueuedCount > 0 && (
          <div className="mb-4 mt-2">
            <div className="flex space-x-3 items-start">
              <div className="flex-shrink-0">
                <div className="w-8 h-8 rounded-full flex items-center justify-center bg-primary ring-2 ring-background">
                  <span className="text-white text-xs font-semibold">AI</span>
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-[10px] text-muted-foreground/60 mb-1">just now</div>
                <div className="rounded-lg bg-card">
                  <p className="text-sm text-muted-foreground leading-relaxed mb-[0.2rem]">
                    Please wait one moment{".".repeat(waitingDotCount)}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {shouldShowMessages && isRunActive && (
          <div className="pl-12 pr-4 py-2">
            <LoadingDots />
          </div>
        )}

        {!showWelcomeExperience && !isCreating && visibleMessages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No messages yet</p>
            <p className="text-xs mt-1">
              {isTerminalState
                ? `Session has ${phase.toLowerCase()}.`
                : "Start by sending a message below."}
            </p>
          </div>
        )}
      </div>

      <ChatInputBox
        value={chatInput}
        onChange={setChatInput}
        onSend={handleSendChat}
        onInterrupt={onInterrupt}
        onPasteImage={onPasteImage}
        isRunActive={isRunActive}
        isSending={sendingChat}
        agents={workflowMetadata?.agents || []}
        commands={workflowMetadata?.commands || []}
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
    </div>
  );
};

export default MessagesTab;
