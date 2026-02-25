"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { HelpCircle, CheckCircle2, Send } from "lucide-react";
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
};

function parseQuestions(input: Record<string, unknown>): AskUserQuestionItem[] {
  const raw = input as unknown as AskUserQuestionInput;
  if (raw?.questions && Array.isArray(raw.questions)) {
    return raw.questions;
  }
  return [];
}

function isAnswered(resultBlock?: ToolResultBlock): boolean {
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
}) => {
  const questions = parseQuestions(toolUseBlock.input);
  const answered = isAnswered(resultBlock);
  const formattedTime = formatTimestamp(timestamp);

  // State: map from question text to selected label(s)
  const [selections, setSelections] = useState<Record<string, string | string[]>>({});

  const handleSingleSelect = (questionText: string, label: string) => {
    if (answered) return;
    setSelections((prev) => ({ ...prev, [questionText]: label }));
  };

  const handleMultiSelect = (questionText: string, label: string, checked: boolean) => {
    if (answered) return;
    setSelections((prev) => {
      const current = (prev[questionText] as string[]) || [];
      if (checked) {
        return { ...prev, [questionText]: [...current, label] };
      }
      return { ...prev, [questionText]: current.filter((l) => l !== label) };
    });
  };

  const allQuestionsAnswered = questions.every((q) => {
    const sel = selections[q.question];
    if (!sel) return false;
    if (Array.isArray(sel)) return sel.length > 0;
    return sel.length > 0;
  });

  const handleSubmit = () => {
    if (!onSubmitAnswer || !allQuestionsAnswered) return;

    // Format as a readable answer message
    const parts = questions.map((q) => {
      const sel = selections[q.question];
      const answer = Array.isArray(sel) ? sel.join(", ") : sel;
      if (questions.length === 1) return answer;
      return `${q.header || q.question}: ${answer}`;
    });

    onSubmitAnswer(parts.join("\n"));
  };

  if (questions.length === 0) {
    return null;
  }

  return (
    <div className="mb-4">
      <div className="flex items-start space-x-3">
        {/* Avatar */}
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-amber-500">
            <HelpCircle className="w-4 h-4 text-white" />
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {formattedTime && (
            <div className="text-[10px] text-muted-foreground/60 mb-1">
              {formattedTime}
            </div>
          )}

          <Card className={cn(
            "border-l-4",
            answered ? "border-l-green-500 bg-green-50/50 dark:bg-green-950/20" : "border-l-amber-500 bg-amber-50/50 dark:bg-amber-950/20"
          )}>
            <CardContent className="p-4 space-y-4">
              {/* Header */}
              <div className="flex items-center gap-2">
                {answered ? (
                  <CheckCircle2 className="w-4 h-4 text-green-600" />
                ) : (
                  <HelpCircle className="w-4 h-4 text-amber-600" />
                )}
                <span className={cn(
                  "text-sm font-medium",
                  answered ? "text-green-700 dark:text-green-400" : "text-amber-700 dark:text-amber-400"
                )}>
                  {answered ? "Question Answered" : "Input Needed"}
                </span>
              </div>

              {/* Questions */}
              {questions.map((q, qIdx) => (
                <div key={qIdx} className="space-y-2">
                  {q.header && (
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                      {q.header}
                    </div>
                  )}
                  <p className="text-sm text-foreground">{q.question}</p>

                  {/* Options */}
                  {q.multiSelect ? (
                    /* Multi-select: Checkboxes */
                    <div className="space-y-2 pl-1">
                      {q.options.map((opt) => {
                        const currentSel = (selections[q.question] as string[]) || [];
                        const isSelected = currentSel.includes(opt.label);

                        return (
                          <div key={opt.label} className="flex items-start space-x-2">
                            <Checkbox
                              id={`q${qIdx}-${opt.label}`}
                              checked={isSelected}
                              onCheckedChange={(checked) =>
                                handleMultiSelect(q.question, opt.label, checked === true)
                              }
                              disabled={answered}
                            />
                            <div className="grid gap-0.5 leading-none">
                              <Label
                                htmlFor={`q${qIdx}-${opt.label}`}
                                className="text-sm font-medium cursor-pointer"
                              >
                                {opt.label}
                              </Label>
                              {opt.description && (
                                <p className="text-xs text-muted-foreground">
                                  {opt.description}
                                </p>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    /* Single-select: Buttons */
                    <div className="flex flex-wrap gap-2">
                      {q.options.map((opt) => {
                        const isSelected = selections[q.question] === opt.label;

                        return (
                          <Button
                            key={opt.label}
                            variant={isSelected ? "default" : "outline"}
                            size="sm"
                            className={cn(
                              "h-auto py-1.5 px-3",
                              answered && !isSelected && "opacity-50"
                            )}
                            onClick={() => handleSingleSelect(q.question, opt.label)}
                            disabled={answered}
                            title={opt.description}
                          >
                            {opt.label}
                          </Button>
                        );
                      })}
                    </div>
                  )}

                  {/* Show descriptions for single-select options below the buttons */}
                  {!q.multiSelect && q.options.some((o) => o.description) && (
                    <div className="space-y-1 pl-1">
                      {q.options.map((opt) => {
                        if (!opt.description) return null;
                        const isSelected = selections[q.question] === opt.label;
                        return (
                          <p
                            key={opt.label}
                            className={cn(
                              "text-xs text-muted-foreground",
                              isSelected && "text-foreground font-medium"
                            )}
                          >
                            <span className="font-medium">{opt.label}:</span>{" "}
                            {opt.description}
                          </p>
                        );
                      })}
                    </div>
                  )}
                </div>
              ))}

              {/* Submit button */}
              {!answered && onSubmitAnswer && (
                <div className="flex justify-end pt-1">
                  <Button
                    size="sm"
                    onClick={handleSubmit}
                    disabled={!allQuestionsAnswered}
                    className="gap-1.5"
                  >
                    <Send className="w-3 h-3" />
                    Submit
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

AskUserQuestionMessage.displayName = "AskUserQuestionMessage";
