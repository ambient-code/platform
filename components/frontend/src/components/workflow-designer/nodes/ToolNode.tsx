"use client";

import { Position, type NodeProps } from "@xyflow/react";
import { Wrench } from "lucide-react";
import { BaseNode } from "./BaseNode";

type ToolNodeData = {
  toolName?: string;
  serverName?: string;
};

export function ToolNode({ data, selected }: NodeProps) {
  const nodeData = data as ToolNodeData;

  return (
    <BaseNode
      label="MCP Tool"
      color="orange"
      icon={<Wrench className="h-4 w-4" />}
      selected={!!selected}
      handles={[
        { type: "target", position: Position.Top },
        { type: "source", position: Position.Bottom },
      ]}
    >
      <div className="space-y-1 text-muted-foreground">
        {nodeData.serverName && (
          <p>
            <span className="font-medium text-foreground">Server:</span>{" "}
            {nodeData.serverName}
          </p>
        )}
        {nodeData.toolName && (
          <p>
            <span className="font-medium text-foreground">Tool:</span>{" "}
            {nodeData.toolName}
          </p>
        )}
        {!nodeData.toolName && !nodeData.serverName && (
          <p className="italic">Click to configure tool</p>
        )}
      </div>
    </BaseNode>
  );
}
