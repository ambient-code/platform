"use client";

import { Position, type NodeProps } from "@xyflow/react";
import { FileOutput } from "lucide-react";
import { BaseNode } from "./BaseNode";

type OutputNodeData = {
  path?: string;
  format?: string;
};

export function OutputNode({ data, selected }: NodeProps) {
  const nodeData = data as OutputNodeData;

  return (
    <BaseNode
      label="Output"
      color="yellow"
      icon={<FileOutput className="h-4 w-4" />}
      selected={!!selected}
      handles={[{ type: "target", position: Position.Top }]}
    >
      <div className="space-y-1 text-muted-foreground">
        {nodeData.path && (
          <p>
            <span className="font-medium text-foreground">Path:</span>{" "}
            {nodeData.path}
          </p>
        )}
        {nodeData.format && (
          <p>
            <span className="font-medium text-foreground">Format:</span>{" "}
            {nodeData.format}
          </p>
        )}
        {!nodeData.path && !nodeData.format && (
          <p className="italic">Click to configure output</p>
        )}
      </div>
    </BaseNode>
  );
}
