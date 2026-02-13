"use client";

import { Position, type NodeProps } from "@xyflow/react";
import { Play } from "lucide-react";
import { BaseNode } from "./BaseNode";

export function StartNode({ selected }: NodeProps) {
  return (
    <BaseNode
      label="Start"
      color="emerald"
      icon={<Play className="h-4 w-4" />}
      selected={!!selected}
      handles={[{ type: "source", position: Position.Bottom }]}
    />
  );
}
