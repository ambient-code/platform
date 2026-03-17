"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MessageSquare, X } from "lucide-react";
import type { FileTab, ActiveTab } from "../hooks/use-file-tabs";

type ContentTabsProps = {
  openTabs: FileTab[];
  activeTab: ActiveTab;
  onSwitchToChat: () => void;
  onSwitchToFile: (path: string) => void;
  onCloseFile: (path: string) => void;
  rightActions?: React.ReactNode;
};

export function ContentTabs({
  openTabs,
  activeTab,
  onSwitchToChat,
  onSwitchToFile,
  onCloseFile,
  rightActions,
}: ContentTabsProps) {
  const isChatActive = activeTab.type === "chat";

  return (
    <div className="flex items-center border-b bg-muted/30 px-3 h-10">
      <div className="flex items-center gap-0.5 flex-1 overflow-x-auto">
        {/* Chat tab — always present, not closable */}
        <button
          type="button"
          onClick={onSwitchToChat}
          className={cn(
            "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-sm text-sm font-medium transition-colors whitespace-nowrap",
            isChatActive
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground hover:bg-accent"
          )}
        >
          <MessageSquare className="w-3.5 h-3.5" />
          Chat
        </button>

        {/* File tabs — closable */}
        {openTabs.map((tab) => {
          const isActive =
            activeTab.type === "file" && activeTab.path === tab.path;
          return (
            <div
              key={tab.path}
              className={cn(
                "inline-flex items-center gap-1 rounded-sm text-sm transition-colors whitespace-nowrap group",
                isActive
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent"
              )}
            >
              <button
                type="button"
                onClick={() => onSwitchToFile(tab.path)}
                className="pl-3 py-1.5 font-medium"
              >
                {tab.name}
              </button>
              <Button
                variant="ghost"
                size="icon"
                className="h-5 w-5 mr-1 opacity-0 group-hover:opacity-100 transition-opacity"
                onClick={(e) => {
                  e.stopPropagation();
                  onCloseFile(tab.path);
                }}
              >
                <X className="w-3 h-3" />
                <span className="sr-only">Close {tab.name}</span>
              </Button>
            </div>
          );
        })}
      </div>

      {/* Right-side actions (settings, explorer toggle) */}
      {rightActions && (
        <div className="flex items-center gap-1 ml-auto pl-2 flex-shrink-0">
          {rightActions}
        </div>
      )}
    </div>
  );
}
