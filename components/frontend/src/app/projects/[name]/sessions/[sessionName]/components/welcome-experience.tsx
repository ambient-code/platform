"use client";

import { useState, useEffect, useRef } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Search } from "lucide-react";
import { cn } from "@/lib/utils";
import type { WorkflowConfig } from "../lib/types";

type WelcomeExperienceProps = {
  ootbWorkflows: WorkflowConfig[];
  onWorkflowSelect: (workflowId: string) => void;
  onUserInteraction: () => void;
  userHasInteracted: boolean;
  sessionPhase?: string;
  hasRealMessages: boolean;
  onLoadWorkflow?: () => void;
  selectedWorkflow?: string;
};

const WELCOME_MESSAGE = `Welcome to Ambient AI! Please select a workflow or type a message to get started.`;
const SETUP_MESSAGE = `Great! Give me a moment to get set up`;

export function WelcomeExperience({
  ootbWorkflows,
  onWorkflowSelect,
  onUserInteraction,
  userHasInteracted,
  sessionPhase,
  hasRealMessages,
  onLoadWorkflow,
  selectedWorkflow = "none",
}: WelcomeExperienceProps) {
  const [displayedText, setDisplayedText] = useState("");
  const [isTypingComplete, setIsTypingComplete] = useState(false);
  const [setupDisplayedText, setSetupDisplayedText] = useState("");
  const [isSetupTypingComplete, setIsSetupTypingComplete] = useState(false);
  const [dotCount, setDotCount] = useState(0);
  const [workflowSearch, setWorkflowSearch] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);
  
  // Track if welcome experience was shown on initial load (persists even when messages appear)
  // This is captured on first render - if there were no real messages initially, we show welcome
  const welcomeShownOnLoadRef = useRef<boolean | null>(null);
  if (welcomeShownOnLoadRef.current === null) {
    welcomeShownOnLoadRef.current = !hasRealMessages;
  }
  
  // Use the selectedWorkflow prop to determine which workflow is currently selected
  const selectedWorkflowId = selectedWorkflow !== "none" ? selectedWorkflow : null;

  // Determine if we should show workflow cards and animation
  // Show animation unless we know for certain the session has already started running or user has interacted
  const isRunningOrBeyond = sessionPhase === "Running" || sessionPhase === "Completed" || sessionPhase === "Failed" || sessionPhase === "Stopped";
  const shouldShowAnimation = !userHasInteracted && !hasRealMessages && !isRunningOrBeyond;
  // Show workflow cards if welcome was shown on load (even if messages appear later) and session is not in terminal state
  const isTerminalPhase = sessionPhase === "Completed" || sessionPhase === "Failed" || sessionPhase === "Stopped";
  const shouldShowWorkflowCards = welcomeShownOnLoadRef.current && !isTerminalPhase;

  // Streaming text effect
  useEffect(() => {
    if (!shouldShowAnimation) {
      // Skip animation if session is already running or user has interacted
      setDisplayedText(WELCOME_MESSAGE);
      setIsTypingComplete(true);
      return;
    }

    let currentIndex = 0;
    let intervalId: ReturnType<typeof setInterval> | null = null;
    
    intervalId = setInterval(() => {
      if (currentIndex < WELCOME_MESSAGE.length) {
        setDisplayedText(WELCOME_MESSAGE.slice(0, currentIndex + 1));
        currentIndex++;
      } else {
        setIsTypingComplete(true);
        if (intervalId !== null) {
          clearInterval(intervalId);
          intervalId = null;
        }
      }
    }, 25); // 25ms per character

    return () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    };
  }, [shouldShowAnimation]);

  // Setup message typing effect (after workflow selected)
  useEffect(() => {
    if (!selectedWorkflowId) return;

    let currentIndex = 0;
    let intervalId: ReturnType<typeof setInterval> | null = null;
    
    intervalId = setInterval(() => {
      if (currentIndex < SETUP_MESSAGE.length) {
        setSetupDisplayedText(SETUP_MESSAGE.slice(0, currentIndex + 1));
        currentIndex++;
      } else {
        setIsSetupTypingComplete(true);
        if (intervalId !== null) {
          clearInterval(intervalId);
          intervalId = null;
        }
      }
    }, 25); // 25ms per character

    return () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    };
  }, [selectedWorkflowId]);

  // Animate dots after setup message completes (stop when real messages appear)
  useEffect(() => {
    if (!isSetupTypingComplete || hasRealMessages) return;

    let intervalId: ReturnType<typeof setInterval> | null = null;
    
    intervalId = setInterval(() => {
      setDotCount((prev) => (prev + 1) % 4); // Cycles 0, 1, 2, 3
    }, 500); // Change dot every 500ms

    return () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    };
  }, [isSetupTypingComplete, hasRealMessages]);

  const handleWorkflowSelect = (workflowId: string) => {
    onWorkflowSelect(workflowId);
    onUserInteraction();
  };

  // Filter out template workflows and only show enabled ones for the welcome cards
  const enabledWorkflows = ootbWorkflows
    .filter((w) => {
      const nameLower = (w.name || "").toLowerCase().trim();
      const idLower = (w.id || "").toLowerCase().trim();
      const isTemplate = nameLower.includes("template") || idLower.includes("template");
      return w.enabled && !isTemplate;
    })
    .sort((a, b) => {
      // Custom order: PRD workflows first, then the rest
      const aHasPRD = a.name.toLowerCase().includes("prd");
      const bHasPRD = b.name.toLowerCase().includes("prd");
      
      if (aHasPRD && !bHasPRD) return -1;
      if (!aHasPRD && bHasPRD) return 1;
      return 0; // Keep original order for items in the same category
    });


  // Filter workflows based on search query (for dropdown - includes all workflows)
  const filteredWorkflows = ootbWorkflows
    .filter((workflow) => {
      if (!workflowSearch) return true;
      const searchLower = workflowSearch.toLowerCase();
      return (
        workflow.name.toLowerCase().includes(searchLower) ||
        workflow.description.toLowerCase().includes(searchLower)
      );
    })
    .sort((a, b) => a.name.localeCompare(b.name)); // Sort alphabetically by display name

  // Filter for general chat based on search
  const showGeneralChat = !workflowSearch || 
    "general chat".includes(workflowSearch.toLowerCase()) ||
    "A general chat session with no structured workflow.".toLowerCase().includes(workflowSearch.toLowerCase());

  // Filter for custom workflow based on search
  const showCustomWorkflow = !workflowSearch ||
    "custom workflow".toLowerCase().includes(workflowSearch.toLowerCase()) ||
    "load a workflow from a custom git repository".toLowerCase().includes(workflowSearch.toLowerCase());

  return (
    <>
      {/* Only show welcome experience if it was shown on initial load */}
      {welcomeShownOnLoadRef.current && (
        <div className="space-y-4">
          {/* Static welcome message styled like a chat message */}
          <div className="mb-4 mt-6">
            <div className="flex space-x-3 items-start">
              {/* Avatar */}
              <div className="flex-shrink-0">
                <div className="w-8 h-8 rounded-full flex items-center justify-center bg-blue-600">
                  <span className="text-white text-xs font-semibold">AI</span>
                </div>
              </div>

              {/* Message Content */}
              <div className="flex-1 min-w-0">
                <div className="rounded-lg bg-card">
                  {/* Content */}
                  <p className="text-sm text-muted-foreground leading-relaxed whitespace-pre-wrap mb-[0.2rem]">
                    {shouldShowAnimation && !isTypingComplete ? (
                      <>
                        {displayedText.slice(0, -3)}
                        {displayedText.slice(-3).split('').map((char, idx) => (
                          <span key={displayedText.length - 3 + idx} className="animate-fade-in-char">
                            {char}
                          </span>
                        ))}
                      </>
                    ) : (
                      displayedText
                    )}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Workflow cards - show after typing completes (only for initial phases) */}
          {shouldShowWorkflowCards && isTypingComplete && enabledWorkflows.length > 0 && (
            <div className="pl-11 pr-4 space-y-2 animate-fade-in-up">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                {enabledWorkflows.map((workflow, index) => (
                  <Card
                    key={workflow.id}
                    title={workflow.description}
                    className={cn(
                      "py-2 cursor-pointer transition-all hover:shadow-md hover:border-primary/50",
                      selectedWorkflowId === workflow.id
                        ? "border-primary bg-primary/5"
                        : selectedWorkflowId !== null
                          ? "opacity-60 cursor-not-allowed bg-muted/30"
                          : ""
                    )}
                    style={{
                      animation: `fade-in-up 0.5s ease-out ${index * 0.1}s both`
                    }}
                    onClick={() => {
                      if (selectedWorkflowId === null) {
                        handleWorkflowSelect(workflow.id);
                      }
                    }}
                  >
                    <CardContent className="p-3 space-y-1">
                      <h3 className={cn(
                        "text-sm font-semibold",
                        selectedWorkflowId !== null && selectedWorkflowId !== workflow.id && "text-muted-foreground/60"
                      )}>
                        {workflow.name}
                      </h3>
                      <p className={cn(
                        "text-xs line-clamp-2",
                        selectedWorkflowId !== null && selectedWorkflowId !== workflow.id
                          ? "text-muted-foreground/40"
                          : "text-muted-foreground"
                      )}>
                        {workflow.description}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>

              {/* View all workflows button */}
              <div 
                className="mt-6 flex justify-start items-center gap-4"
                style={{
                  animation: `fade-in-up 0.5s ease-out ${enabledWorkflows.length * 0.1}s both`
                }}
              >
                <DropdownMenu onOpenChange={(open) => {
                  if (open) {
                    setWorkflowSearch("");
                    // Focus the search input after a brief delay to ensure it's rendered
                    setTimeout(() => searchInputRef.current?.focus(), 0);
                  }
                }}>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      className="text-sm text-primary hover:text-primary/80 hover:bg-transparent p-0 h-auto cursor-pointer"
                      disabled={selectedWorkflowId !== null}
                    >
                      View all workflows
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="center" className="w-[450px]">
                    {/* Search box */}
                    <div className="px-2 py-2 border-b sticky top-0 bg-popover z-10">
                      <div className="relative">
                        <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          ref={searchInputRef}
                          type="text"
                          placeholder="Search workflows..."
                          value={workflowSearch}
                          onChange={(e) => setWorkflowSearch(e.target.value)}
                          className="pl-8 h-9"
                          onKeyDown={(e) => {
                            // Prevent dropdown from closing on keyboard interaction
                            e.stopPropagation();
                          }}
                        />
                      </div>
                    </div>

                    {/* Workflow items */}
                    <div className="max-h-[400px] overflow-y-auto">
                      {showGeneralChat && (
                        <>
                          <DropdownMenuItem
                            onClick={() => handleWorkflowSelect("none")}
                            disabled={selectedWorkflowId !== null}
                          >
                            <div className="flex flex-col items-start gap-0.5 py-1 w-full">
                              <span>General chat</span>
                              <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                                A general chat session with no structured workflow.
                              </span>
                            </div>
                          </DropdownMenuItem>
                          {filteredWorkflows.length > 0 && <DropdownMenuSeparator />}
                        </>
                      )}
                      {filteredWorkflows.map((workflow) => (
                        <DropdownMenuItem
                          key={workflow.id}
                          onClick={() => workflow.enabled && handleWorkflowSelect(workflow.id)}
                          disabled={!workflow.enabled || selectedWorkflowId !== null}
                        >
                          <div className="flex flex-col items-start gap-0.5 py-1 w-full">
                            <span>{workflow.name}</span>
                            <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                              {workflow.description}
                            </span>
                          </div>
                        </DropdownMenuItem>
                      ))}
                      {(showGeneralChat || filteredWorkflows.length > 0) && showCustomWorkflow && (
                        <DropdownMenuSeparator />
                      )}
                      {showCustomWorkflow && (
                        <DropdownMenuItem
                          onClick={() => handleWorkflowSelect("custom")}
                          disabled={selectedWorkflowId !== null}
                        >
                          <div className="flex flex-col items-start gap-0.5 py-1 w-full">
                            <span>Custom workflow...</span>
                            <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                              Load a workflow from a custom Git repository
                            </span>
                          </div>
                        </DropdownMenuItem>
                      )}
                      {!showGeneralChat && filteredWorkflows.length === 0 && !showCustomWorkflow && (
                        <div className="px-2 py-6 text-center text-sm text-muted-foreground">
                          No workflows found
                        </div>
                      )}
                    </div>
                  </DropdownMenuContent>
                </DropdownMenu>
                
                {onLoadWorkflow && (
                  <Button
                    variant="ghost"
                    className="text-sm text-primary hover:text-primary/80 hover:bg-transparent p-0 h-auto cursor-pointer"
                    disabled={selectedWorkflowId !== null}
                    onClick={onLoadWorkflow}
                  >
                    Load workflow
                  </Button>
                )}
              </div>
            </div>
          )}

          {/* Setup message after workflow selection - only show if no real messages yet */}
          {selectedWorkflowId && !hasRealMessages && (
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
                <div className="rounded-lg bg-card">
                  {/* Content */}
                  <p className="text-sm text-muted-foreground leading-relaxed mb-[0.2rem]">
                      {!isSetupTypingComplete ? (
                        <>
                          {setupDisplayedText.slice(0, -3)}
                          {setupDisplayedText.slice(-3).split('').map((char, idx) => (
                            <span key={setupDisplayedText.length - 3 + idx} className="animate-fade-in-char">
                              {char}
                            </span>
                          ))}
                        </>
                      ) : (
                        <>
                          {setupDisplayedText}
                          {".".repeat(dotCount)}
                        </>
                      )}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}

