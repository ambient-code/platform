"use client";

import { Position, type NodeProps } from "@xyflow/react";
import { ShieldCheck } from "lucide-react";
import { BaseNode } from "./BaseNode";

type ReviewNodeData = {
  conditions?: string[];
};

export function ReviewNode({ data, selected }: NodeProps) {
  const nodeData = data as ReviewNodeData;
  const conditions = nodeData.conditions ?? [];

  return (
    <BaseNode
      label="Review Gate"
      color="purple"
      icon={<ShieldCheck className="h-4 w-4" />}
      selected={!!selected}
      handles={[
        { type: "target", position: Position.Top },
        { type: "source", position: Position.Bottom },
      ]}
    >
      <div className="text-muted-foreground">
        {conditions.length > 0 ? (
          <ul className="list-disc pl-3 space-y-0.5">
            {conditions.slice(0, 3).map((c, i) => (
              <li key={i} className="truncate">
                {c}
              </li>
            ))}
            {conditions.length > 3 && (
              <li className="italic">+{conditions.length - 3} more</li>
            )}
          </ul>
        ) : (
          <p className="italic">Click to configure conditions</p>
        )}
      </div>
    </BaseNode>
  );
}
