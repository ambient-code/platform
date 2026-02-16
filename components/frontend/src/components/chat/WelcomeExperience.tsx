"use client";

import React from "react";
import { Bug, Sparkles, FileSearch, GitPullRequest } from "lucide-react";
import { Button } from "@/components/ui/button";

export type WelcomePrompt = {
  id: string;
  icon: React.ElementType;
  label: string;
  description: string;
  prompt: string;
};

const DEFAULT_PROMPTS: WelcomePrompt[] = [
  {
    id: "fix-bug",
    icon: Bug,
    label: "Fix a bug",
    description: "Debug and fix issues in your code",
    prompt: "Help me fix a bug in ",
  },
  {
    id: "add-feature",
    icon: Sparkles,
    label: "Add a feature",
    description: "Implement new functionality",
    prompt: "I want to add a feature that ",
  },
  {
    id: "understand-code",
    icon: FileSearch,
    label: "Understand code",
    description: "Explain how something works",
    prompt: "Explain how ",
  },
  {
    id: "review-pr",
    icon: GitPullRequest,
    label: "Review a PR",
    description: "Get feedback on code changes",
    prompt: "Review the PR at ",
  },
];

export type WelcomeExperienceProps = {
  onSelectPrompt: (prompt: string) => void;
  visible?: boolean;
};

export const WelcomeExperience: React.FC<WelcomeExperienceProps> = ({
  onSelectPrompt,
  visible = true,
}) => {
  if (!visible) return null;

  return (
    <div className="flex flex-col items-center justify-center py-8 px-4">
      {/* Header */}
      <div className="text-center mb-8">
        <h2 className="text-xl font-semibold mb-2">What would you like to work on?</h2>
        <p className="text-sm text-muted-foreground max-w-md">
          Choose a starting point below, or type your own message in the chat box.
        </p>
      </div>

      {/* Prompt Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full max-w-xl">
        {DEFAULT_PROMPTS.map((prompt) => {
          const Icon = prompt.icon;

          return (
            <Button
              key={prompt.id}
              variant="outline"
              className="h-auto p-4 flex flex-col items-start gap-2 text-left hover:border-primary hover:bg-primary/5 transition-colors"
              onClick={() => onSelectPrompt(prompt.prompt)}
            >
              <div className="flex items-center gap-2">
                <div className="p-1.5 rounded-md bg-primary/10">
                  <Icon className="h-4 w-4 text-primary" />
                </div>
                <span className="font-medium">{prompt.label}</span>
              </div>
              <p className="text-xs text-muted-foreground">{prompt.description}</p>
            </Button>
          );
        })}
      </div>

      {/* Hint */}
      <p className="text-xs text-muted-foreground mt-6">
        Type <kbd className="px-1 py-0.5 bg-muted rounded text-[10px]">@</kbd> to mention an agent or{" "}
        <kbd className="px-1 py-0.5 bg-muted rounded text-[10px]">/</kbd> for commands
      </p>
    </div>
  );
};

export default WelcomeExperience;
