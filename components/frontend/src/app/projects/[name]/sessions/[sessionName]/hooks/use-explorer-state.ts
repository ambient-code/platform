"use client";

import { useCallback } from "react";
import { useLocalStorage } from "@/hooks/use-local-storage";

type ExplorerTab = "files" | "context";

export function useExplorerState() {
  const [visible, setVisible] = useLocalStorage("session-explorer-visible", false);
  const [activeTab, setActiveTab] = useLocalStorage<ExplorerTab>("session-explorer-tab", "files");

  const open = useCallback((tab?: ExplorerTab) => {
    setVisible(true);
    if (tab) setActiveTab(tab);
  }, [setVisible, setActiveTab]);

  const close = useCallback(() => {
    setVisible(false);
  }, [setVisible]);

  const toggle = useCallback(() => {
    setVisible((prev) => !prev);
  }, [setVisible]);

  return { visible, activeTab, setActiveTab, open, close, toggle };
}
