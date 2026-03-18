"use client";

import { useState, useRef, useCallback, useMemo } from "react";
import type { WorkflowConfig } from "../lib/types";

type UseWorkflowSelectionProps = {
  selectedWorkflow: string;
  ootbWorkflows: WorkflowConfig[];
  workflowActivating: boolean;
  onWorkflowChange: (value: string) => void;
};

export function useWorkflowSelection({
  selectedWorkflow,
  ootbWorkflows,
  workflowActivating,
  onWorkflowChange,
}: UseWorkflowSelectionProps) {
  const [search, setSearch] = useState("");
  const [popoverOpen, setPopoverOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const filteredWorkflows = useMemo(() =>
    ootbWorkflows
      .filter((workflow) => {
        if (!search) return true;
        const lower = search.toLowerCase();
        return (
          workflow.name.toLowerCase().includes(lower) ||
          (workflow.description ?? "").toLowerCase().includes(lower)
        );
      })
      .sort((a, b) => a.name.localeCompare(b.name)),
    [ootbWorkflows, search]
  );

  const showGeneralChat = useMemo(() =>
    !search ||
    "general chat".includes(search.toLowerCase()) ||
    "a general chat session with no structured workflow."
      .toLowerCase()
      .includes(search.toLowerCase()),
    [search]
  );

  const showCustomWorkflow = useMemo(() =>
    !search ||
    "custom workflow".toLowerCase().includes(search.toLowerCase()) ||
    "load a workflow from a custom git repository"
      .toLowerCase()
      .includes(search.toLowerCase()),
    [search]
  );

  const selectedLabel = useMemo((): string => {
    if (selectedWorkflow === "none") return "No workflow";
    if (selectedWorkflow === "custom") return "Custom workflow";
    const wf = ootbWorkflows.find((w) => w.id === selectedWorkflow);
    return wf?.name || "No workflow";
  }, [selectedWorkflow, ootbWorkflows]);

  const handleSelect = useCallback(
    (value: string) => {
      onWorkflowChange(value);
      setPopoverOpen(false);
    },
    [onWorkflowChange]
  );

  const handleOpenChange = useCallback((open: boolean) => {
    setPopoverOpen(open);
    if (open) {
      setSearch("");
      setTimeout(() => searchInputRef.current?.focus(), 0);
    }
  }, []);

  return {
    search,
    setSearch,
    popoverOpen,
    searchInputRef,
    filteredWorkflows,
    showGeneralChat,
    showCustomWorkflow,
    selectedLabel,
    isActivating: workflowActivating,
    handleSelect,
    handleOpenChange,
  };
}
