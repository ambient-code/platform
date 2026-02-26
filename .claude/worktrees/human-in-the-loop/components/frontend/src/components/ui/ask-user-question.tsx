"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { HelpCircle, CheckCircle2, Send, ChevronRight } from "lucide-react";
import { formatTimestamp } from "@/lib/format-timestamp";
import type {
  ToolUseBlock,
  ToolResultBlock,
  AskUserQuestionItem,
  AskUserQuestionInput,
} from "@/types/agentic-session";

export type AskUserQuestionMessageProps = {
  toolUseBlock: ToolUseBlock;
  resultBlock?: ToolResultBlock;
  timestamp?: string;
  onSubmitAnswer?: (formattedAnswer: string) => void;
  isNewest?: boolean;
};

function parseQuestions(input: Record<string, unknown>): AskUserQuestionItem[] {
  const raw = input as unknown as AskUserQuestionInput;
  if (raw?.questions && Array.isArray(raw.questions)) {
    return raw.questions;
  }
  return [];
}

function hasResult(resultBlock?: ToolResultBlock): boolean {
  if (!resultBlock) return false;
  const content = resultBlock.content;
  if (!content) return false;
  if (typeof content === "string" && content.trim() === "") return false;
  return true;
}

export const AskUserQuestionMessage: React.FC<AskUserQuestionMessageProps> = ({
  toolUseBlock,
  resultBlock,
  timestamp,
  onSubmitAnswer,
  isNewest = false,
}) => {
  const questions = parseQuestions(toolUseBlock.input);
  const alreadyAnswered = hasResult(resultBlock);
  const formattedTime = formatTimestamp(timestamp);
  const isMultiQuestion = questions.length > 1;

  // Only interactive when newest message AND not already answered/submitted
  const [submitted, setSubmitted] = useState(false);
  const disabled = alreadyAnswered || submitted || !isNewest;

  // Active tab index (for multi-question tabbed view)
  const [activeTab, setActiveTab] = useState(0);

  // Selections: map from question text → selected label(s) or freeform string
  const [selections, setSelections] = useState<Record<string, string | string[]>>({});
  // Track which questions use freeform "Other" input
  const [usingOther, setUsingOther] = useState<Record<string, boolean>>({});
  // Freeform text inputs
  const [otherText, setOtherText] = useState<Record<string, string>>({});

  const handleSingleSelect = (questionText: string, label: string) => {
    if (disabled) return;
    setUsingOther((prev) => ({ ...prev, [questionText]: false }));
    setSelections((prev) => ({ ...prev, [questionText]: label }));
    // Auto-advance to next tab
    if (isMultiQuestion && activeTab < questions.length - 1) {
      setTimeout(() => setActiveTab((t) => t + 1), 250);
    }
  };

  const handleOtherToggle = (questionText: string) => {
    if (disabled) return;
    setUsingOther((prev) => ({ ...prev, [questionText]: true }));
    setSelections((prev) => ({ ...prev, [questionText]: otherText[questionText] || "" }));
  };

  const handleOtherTextChange = (questionText: string, text: string) => {
    if (disabled) return;
    setOtherText((prev) => ({ ...prev, [questionText]: text }));
    setSelections((prev) => ({ ...prev, [questionText]: text }));
  };

  const handleMultiSelect = (questionText: string, label: string, checked: boolean) => {
    if (disabled) return;
    setSelections((prev) => {
      const current = (prev[questionText] as string[]) || [];
      if (checked) return { ...prev, [questionText]: [...current, label] };
      return { ...prev, [questionText]: current.filter((l) => l !== label) };
    });
  };

  const isQuestionAnswered = (q: AskUserQuestionItem): boolean => {
    const sel = selections[q.question];
    if (!sel) return false;
    if (Array.isArray(sel)) return sel.length > 0;
    return sel.length > 0;
  };

  const allQuestionsAnswered = questions.every(isQuestionAnswered);

  const handleSubmit = () => {
    if (!onSubmitAnswer || !allQuestionsAnswered || disabled) return;

    // Build the structured response matching Claude SDK format
    const answers: Record<string, string> = {};
    for (const q of questions) {
      const sel = selections[q.question];
      answers[q.question] = Array.isArray(sel) ? sel.join(", ") : (sel as string);
    }

    // Send as JSON matching the AskUserQuestion response format
    const response = JSON.stringify({ questions, answers });
    onSubmitAnswer(response);
    setSubmitted(true);
  };

  if (questions.length === 0) return null;

  const currentQuestion = questions[activeTab] || questions[0];

  const renderQuestionOptions = (q: AskUserQuestionItem, qIdx: number) => {
    const isOther = usingOther[q.question];

    if (q.multiSelect) {
      return (
        <div className="space-y-1">
          {q.options.map((opt) => {
            const currentSel = (selections[q.question] as string[]) || [];
            const isSelected = currentSel.includes(opt.label);
            return (
              <label
                key={opt.label}
                className={cn(
                  "flex items-start gap-2 p-1.5 rounded cursor-pointer transition-colors",
                  isSelected ? "bg-accent" : "hover:bg-muted/50",
                  disabled && "cursor-default opacity-60"
                )}
              >
                <Checkbox
                  checked={isSelected}
                  onCheckedChange={(checked) =>
                    handleMultiSelect(q.question, opt.label, checked === true)
                  }
                  disabled={disabled}
                  className="mt-0.5"
                />
                <div className="min-w-0">
                  <span className="text-sm">{opt.label}</span>
                  {opt.description && (
                    <p className="text-xs text-muted-foreground leading-tight">{opt.description}</p>
                  )}
                </div>
              </label>
            );
          })}
        </div>
      );
    }

    // Single-select: Radio buttons + Other
    return (
      <div className="space-y-1">
        <RadioGroup
          value={isOther ? "" : ((selections[q.question] as string) || "")}
          onValueChange={(val) => handleSingleSelect(q.question, val)}
          disabled={disabled}
          className="gap-1"
        >
          {q.options.map((opt) => {
            const isSelected = !isOther && selections[q.question] === opt.label;
            return (
              <label
                key={opt.label}
                className={cn(
                  "flex items-start gap-2 p-1.5 rounded cursor-pointer transition-colors",
                  isSelected ? "bg-accent" : "hover:bg-muted/50",
                  disabled && "cursor-default opacity-60"
                )}
              >
                <RadioGroupItem value={opt.label} className="mt-0.5" />
                <div className="min-w-0">
                  <span className="text-sm">{opt.label}</span>
                  {opt.description && (
                    <p className="text-xs text-muted-foreground leading-tight">{opt.description}</p>
                  )}
                </div>
              </label>
            );
          })}
        </RadioGroup>

        {/* Other / freeform option */}
        <label
          className={cn(
            "flex items-start gap-2 p-1.5 rounded cursor-pointer transition-colors",
            isOther ? "bg-accent" : "hover:bg-muted/50",
            disabled && "cursor-default opacity-60"
          )}
          onClick={() => !disabled && handleOtherToggle(q.question)}
        >
          <div className={cn(
            "mt-0.5 aspect-square h-4 w-4 rounded-full border border-primary flex items-center justify-center flex-shrink-0",
            disabled && "opacity-50"
          )}>
            {isOther && <div className="h-2.5 w-2.5 rounded-full bg-current" />}
          </div>
          <div className="min-w-0 flex-1">
            <span className="text-sm">Other</span>
            {isOther && (
              <Input
                autoFocus
                placeholder="Type your answer..."
                value={otherText[q.question] || ""}
                onChange={(e) => handleOtherTextChange(q.question, e.target.value)}
                onClick={(e) => e.stopPropagation()}
                disabled={disabled}
                className="mt-1 h-7 text-sm"
              />
            )}
          </div>
        </label>
      </div>
    );
  };

  return (
    <div className="mb-3">
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <div className="flex-shrink-0">
          <div
            className={cn(
              "w-8 h-8 rounded-full flex items-center justify-center",
              disabled ? "bg-green-600" : "bg-amber-500"
            )}
          >
            {disabled ? (
              <CheckCircle2 className="w-4 h-4 text-white" />
            ) : (
              <HelpCircle className="w-4 h-4 text-white" />
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {formattedTime && (
            <div className="text-[10px] text-muted-foreground/60 mb-0.5">{formattedTime}</div>
          )}

          <div
            className={cn(
              "rounded-lg border-l-3 pl-3 pr-3 py-2.5",
              disabled
                ? "border-l-green-500 bg-green-50/30 dark:bg-green-950/10"
                : "border-l-amber-500 bg-amber-50/30 dark:bg-amber-950/10"
            )}
          >
            {/* Tab navigation for multi-question */}
            {isMultiQuestion && (
              <div className="flex gap-1 mb-2">
                {questions.map((q, idx) => {
                  const done = isQuestionAnswered(q);
                  const active = idx === activeTab;
                  return (
                    <button
                      key={idx}
                      onClick={() => setActiveTab(idx)}
                      className={cn(
                        "flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium transition-colors",
                        active
                          ? "bg-foreground/10 text-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-foreground/5",
                        done && !active && "text-green-600 dark:text-green-400"
                      )}
                    >
                      {done && <CheckCircle2 className="w-3 h-3" />}
                      {q.header || `Q${idx + 1}`}
                    </button>
                  );
                })}
              </div>
            )}

            {/* Question text */}
            <p className="text-sm text-foreground mb-2">{currentQuestion.question}</p>

            {/* Options */}
            {renderQuestionOptions(currentQuestion, activeTab)}

            {/* Footer */}
            {!disabled && (
              <div className="flex items-center justify-between mt-2 pt-1.5 border-t border-border/40">
                <div className="text-xs text-muted-foreground">
                  {isMultiQuestion
                    ? `${questions.filter(isQuestionAnswered).length}/${questions.length} answered`
                    : selections[currentQuestion.question]
                      ? ""
                      : "Select an option"}
                </div>

                <div className="flex items-center gap-1.5">
                  {isMultiQuestion && activeTab < questions.length - 1 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs gap-1 px-2"
                      onClick={() => setActiveTab((t) => t + 1)}
                    >
                      Next
                      <ChevronRight className="w-3 h-3" />
                    </Button>
                  )}

                  {(!isMultiQuestion || activeTab === questions.length - 1) && (
                    <Button
                      size="sm"
                      className="h-7 text-xs gap-1 px-3"
                      onClick={handleSubmit}
                      disabled={!allQuestionsAnswered}
                    >
                      <Send className="w-3 h-3" />
                      Submit
                    </Button>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

AskUserQuestionMessage.displayName = "AskUserQuestionMessage";
