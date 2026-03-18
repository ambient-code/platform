"use client";

import { useState, useCallback } from "react";

export type FileTab = {
  path: string;
  name: string;
};

export type ActiveTab = { type: "chat" } | { type: "file"; path: string };

export function useFileTabs() {
  const [openTabs, setOpenTabs] = useState<FileTab[]>([]);
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

  return {
    openTabs,
    activeTab,
    openFile,
    closeFile,
    switchToChat,
    switchToFile,
  };
}
