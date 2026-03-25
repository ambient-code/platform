"use client";

import { useState, useCallback } from "react";
import type { BackgroundTaskStatus } from "@/types/background-task";

export type FileTab = {
  path: string;
  name: string;
};

export type TaskTab = {
  taskId: string;
  name: string;
  status: BackgroundTaskStatus;
};

export type ActiveTab =
  | { type: "chat" }
  | { type: "file"; path: string }
  | { type: "task"; taskId: string };

export function useFileTabs() {
  const [openTabs, setOpenTabs] = useState<FileTab[]>([]);
  const [openTaskTabs, setOpenTaskTabs] = useState<TaskTab[]>([]);
  const [activeTab, setActiveTab] = useState<ActiveTab>({ type: "chat" });

  const openFile = useCallback((file: FileTab) => {
    setOpenTabs((prev) => {
      if (prev.some((t) => t.path === file.path)) return prev;
      return [...prev, file];
    });
    setActiveTab({ type: "file", path: file.path });
  }, []);

  const closeFile = useCallback(
    (path: string) => {
      setOpenTabs((prev) => prev.filter((t) => t.path !== path));
      setActiveTab((current) => {
        if (current.type === "file" && current.path === path) {
          return { type: "chat" };
        }
        return current;
      });
    },
    []
  );

  const switchToChat = useCallback(() => {
    setActiveTab({ type: "chat" });
  }, []);

  const switchToFile = useCallback((path: string) => {
    setActiveTab({ type: "file", path });
  }, []);

  const openTask = useCallback((tab: TaskTab) => {
    setOpenTaskTabs((prev) => {
      if (prev.some((t) => t.taskId === tab.taskId)) return prev;
      return [...prev, tab];
    });
    setActiveTab({ type: "task", taskId: tab.taskId });
  }, []);

  const closeTask = useCallback((taskId: string) => {
    setOpenTaskTabs((prev) => prev.filter((t) => t.taskId !== taskId));
    setActiveTab((current) => {
      if (current.type === "task" && current.taskId === taskId) {
        return { type: "chat" };
      }
      return current;
    });
  }, []);

  const switchToTask = useCallback((taskId: string) => {
    setActiveTab({ type: "task", taskId });
  }, []);

  const updateTaskStatus = useCallback((taskId: string, status: BackgroundTaskStatus) => {
    setOpenTaskTabs((prev) => {
      const idx = prev.findIndex((t) => t.taskId === taskId);
      if (idx === -1 || prev[idx].status === status) return prev;
      const next = [...prev];
      next[idx] = { ...prev[idx], status };
      return next;
    });
  }, []);

  return {
    openTabs,
    openTaskTabs,
    activeTab,
    openFile,
    closeFile,
    switchToChat,
    switchToFile,
    openTask,
    closeTask,
    switchToTask,
    updateTaskStatus,
  };
}
