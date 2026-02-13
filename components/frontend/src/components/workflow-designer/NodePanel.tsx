"use client";

import type { DragEvent } from "react";
import {
  Play,
  MessageSquare,
  Wrench,
  ShieldCheck,
  FileOutput,
  GripVertical,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { useDnD } from "./contexts/DnDContext";

type PaletteItem = {
  nodeType: string;
  label: string;
  icon: typeof Play;
  color: string;
  description: string;
};

const paletteItems: PaletteItem[] = [
  {
    nodeType: "start-node",
    label: "Start",
    icon: Play,
    color: "text-emerald-400",
    description: "Entry point",
  },
  {
    nodeType: "prompt-node",
    label: "Prompt",
    icon: MessageSquare,
    color: "text-blue-400",
    description: "Claude prompt step",
  },
  {
    nodeType: "tool-node",
    label: "MCP Tool",
    icon: Wrench,
    color: "text-orange-400",
    description: "Tool invocation",
  },
  {
    nodeType: "review-node",
    label: "Review Gate",
    icon: ShieldCheck,
    color: "text-purple-400",
    description: "Quality check",
  },
  {
    nodeType: "output-node",
    label: "Output",
    icon: FileOutput,
    color: "text-yellow-400",
    description: "Artifact output",
  },
];

export function NodePanel() {
  const { setType } = useDnD();

  function handleDragStart(event: DragEvent, nodeType: string) {
    event.dataTransfer.setData("application/reactflow", nodeType);
    event.dataTransfer.effectAllowed = "move";
    setType(nodeType);
  }

  return (
    <Card className="w-56 shrink-0 h-fit">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">Nodes</CardTitle>
      </CardHeader>
      <CardContent className="space-y-1.5 px-3 pb-3">
        {paletteItems.map((item) => {
          const Icon = item.icon;
          return (
            <div
              key={item.nodeType}
              draggable
              onDragStart={(e) => handleDragStart(e, item.nodeType)}
              className={cn(
                "flex items-center gap-2 rounded-md border px-2.5 py-2",
                "cursor-grab select-none transition-colors",
                "hover:bg-accent active:cursor-grabbing"
              )}
            >
              <GripVertical className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <Icon className={cn("h-4 w-4 shrink-0", item.color)} />
              <div className="min-w-0">
                <p className="text-sm font-medium leading-tight truncate">
                  {item.label}
                </p>
                <p className="text-[10px] text-muted-foreground leading-tight truncate">
                  {item.description}
                </p>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
