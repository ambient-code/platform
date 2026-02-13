import type { NodeTypes } from "@xyflow/react";
import { StartNode } from "./StartNode";
import { PromptNode } from "./PromptNode";
import { ToolNode } from "./ToolNode";
import { ReviewNode } from "./ReviewNode";
import { OutputNode } from "./OutputNode";

export const nodeTypes: NodeTypes = {
  "start-node": StartNode,
  "prompt-node": PromptNode,
  "tool-node": ToolNode,
  "review-node": ReviewNode,
  "output-node": OutputNode,
};
