"use client";

import { Position, type NodeProps } from "@xyflow/react";
import { MessageSquare } from "lucide-react";
import { BaseNode } from "./BaseNode";

type PromptNodeData = {
  prompt?: string;
  model?: string;
};

export function PromptNode({ data, selected }: NodeProps) {
  const nodeData = data as PromptNodeData;

  return (
    <BaseNode
      label="Prompt"
      color="blue"
      icon={<MessageSquare className="h-4 w-4" />}
      selected={!!selected}
      handles={[
        { type: "target", position: Position.Top },
        { type: "source", position: Position.Bottom },
      ]}
    >
      <div className="space-y-1 text-muted-foreground">
        {nodeData.model && (
          <p>
            <span className="font-medium text-foreground">Model:</span>{" "}
            {nodeData.model}
          </p>
        )}
        {nodeData.prompt && (
          <p className="line-clamp-2">
            <span className="font-medium text-foreground">Prompt:</span>{" "}
            {nodeData.prompt}
          </p>
        )}
        {!nodeData.prompt && !nodeData.model && (
          <p className="italic">Click to configure prompt</p>
        )}
      </div>
    </BaseNode>
  );
}
