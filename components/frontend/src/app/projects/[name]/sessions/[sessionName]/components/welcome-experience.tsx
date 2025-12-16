"use client";

import { useState, useEffect } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { WorkflowConfig } from "../lib/types";

type WelcomeExperienceProps = {
  ootbWorkflows: WorkflowConfig[];
  onWorkflowSelect: (workflowId: string) => void;
  onUserInteraction: () => void;
  userHasInteracted: boolean;
  sessionPhase?: string;
  hasRealMessages: boolean;
};

const WELCOME_MESSAGE = `Welcome to Ambient AI! Your workspace and all of its context have been loaded. Please select a workflow below to get started, or type a message to begin chatting.`;

export function WelcomeExperience({
  ootbWorkflows,
  onWorkflowSelect,
  onUserInteraction,
  userHasInteracted,
  sessionPhase,
  hasRealMessages,
}: WelcomeExperienceProps) {
  const [displayedText, setDisplayedText] = useState("");
  const [isTypingComplete, setIsTypingComplete] = useState(false);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(null);
  const [cardsEverShown, setCardsEverShown] = useState(false);

  // Determine if we should show workflow cards and animation
  const isInitialPhase = sessionPhase === "Pending" || sessionPhase === "Creating";
  const shouldShowAnimation = isInitialPhase && !userHasInteracted;
  // Show workflow cards if they haven't been shown yet and session is not in terminal state
  const isTerminalPhase = sessionPhase === "Completed" || sessionPhase === "Failed" || sessionPhase === "Stopped";
  const shouldShowWorkflowCards = (cardsEverShown || !hasRealMessages) && !isTerminalPhase;

  // Streaming text effect
  useEffect(() => {
    if (!shouldShowAnimation) {
      // Skip animation if session is already running or user has interacted
      setDisplayedText(WELCOME_MESSAGE);
      setIsTypingComplete(true);
      return;
    }

    let currentIndex = 0;
    const interval = setInterval(() => {
      if (currentIndex < WELCOME_MESSAGE.length) {
        setDisplayedText(WELCOME_MESSAGE.slice(0, currentIndex + 1));
        currentIndex++;
      } else {
        setIsTypingComplete(true);
        clearInterval(interval);
      }
    }, 25); // 25ms per character

    return () => clearInterval(interval);
  }, [shouldShowAnimation]);

  // Track when cards are shown for the first time
  useEffect(() => {
    if (isTypingComplete && !hasRealMessages && !isTerminalPhase) {
      setCardsEverShown(true);
    }
  }, [isTypingComplete, hasRealMessages, isTerminalPhase]);

  const handleWorkflowSelect = (workflowId: string) => {
    setSelectedWorkflowId(workflowId);
    onWorkflowSelect(workflowId);
    onUserInteraction();
  };

  const enabledWorkflows = ootbWorkflows.filter((w) => w.enabled);

  return (
    <>
      <style dangerouslySetInnerHTML={{
        __html: `
          @keyframes fadeInUp {
            from {
              opacity: 0;
              transform: translateY(20px);
            }
            to {
              opacity: 1;
              transform: translateY(0);
            }
          }
        `
      }} />
    <div className="space-y-4">
      {/* Static welcome message styled like a chat message */}
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
            <div className="rounded-lg p-3 bg-card">
              {/* Content */}
              <div className="text-sm text-muted-foreground leading-relaxed whitespace-pre-wrap">
                {displayedText}
                {shouldShowAnimation && !isTypingComplete && (
                  <span className="inline-block w-1 h-4 ml-0.5 bg-primary animate-pulse" />
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Workflow cards - show after typing completes (only for initial phases) */}
      {shouldShowWorkflowCards && isTypingComplete && enabledWorkflows.length > 0 && (
        <div className="px-4 space-y-2 animate-fade-in-up">
          <p className="text-xs font-medium text-muted-foreground">
            Suggested workflows:
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
            {enabledWorkflows.map((workflow, index) => (
              <Card
                key={workflow.id}
                className={cn(
                  "cursor-pointer transition-all hover:shadow-md hover:border-primary/50",
                  selectedWorkflowId === workflow.id
                    ? "border-primary bg-primary/5"
                    : selectedWorkflowId !== null
                      ? "opacity-50 cursor-not-allowed"
                      : ""
                )}
                style={{
                  animation: `fadeInUp 0.5s ease-out ${index * 0.1}s both`
                }}
                onClick={() => {
                  if (selectedWorkflowId === null) {
                    handleWorkflowSelect(workflow.id);
                  }
                }}
              >
                <CardContent className="p-4 space-y-2">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold">{workflow.name}</h3>
                    {selectedWorkflowId === workflow.id && (
                      <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                        Selected
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground line-clamp-2">
                    {workflow.description}
                  </p>
                  {workflow.agentCount !== undefined && workflow.agentCount > 0 && (
                    <Badge variant="outline" className="text-[10px]">
                      {workflow.agentCount} agent{workflow.agentCount !== 1 ? "s" : ""}
                    </Badge>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
    </>
  );
}

