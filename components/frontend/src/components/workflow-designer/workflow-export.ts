import type { Node, Edge } from "@xyflow/react";

type AmbientWorkflowConfig = {
  name: string;
  description: string;
  systemPrompt: string;
  startupPrompt: string;
  results: Record<string, string>;
};

function getOrderedNodes(nodes: Node[], edges: Edge[]): Node[] {
  const adjacency = new Map<string, string[]>();
  for (const edge of edges) {
    const existing = adjacency.get(edge.source) ?? [];
    existing.push(edge.target);
    adjacency.set(edge.source, existing);
  }

  const startNode = nodes.find((n) => n.type === "start-node");
  if (!startNode) return nodes;

  const ordered: Node[] = [];
  const visited = new Set<string>();
  const queue = [startNode.id];

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (visited.has(current)) continue;
    visited.add(current);

    const node = nodes.find((n) => n.id === current);
    if (node) ordered.push(node);

    const neighbors = adjacency.get(current) ?? [];
    for (const neighbor of neighbors) {
      if (!visited.has(neighbor)) {
        queue.push(neighbor);
      }
    }
  }

  return ordered;
}

export function exportToAmbientJson(
  nodes: Node[],
  edges: Edge[]
): AmbientWorkflowConfig {
  const ordered = getOrderedNodes(nodes, edges);

  const promptParts: string[] = [];
  const results: Record<string, string> = {};

  for (const node of ordered) {
    const data = node.data as Record<string, unknown>;

    switch (node.type) {
      case "prompt-node": {
        const prompt = (data.prompt as string) ?? "";
        if (prompt) promptParts.push(prompt);
        break;
      }
      case "tool-node": {
        const server = (data.serverName as string) ?? "";
        const tool = (data.toolName as string) ?? "";
        if (server || tool) {
          promptParts.push(`Use tool ${tool} from server ${server}.`);
        }
        break;
      }
      case "review-node": {
        const conditions = (data.conditions as string[]) ?? [];
        if (conditions.length > 0) {
          promptParts.push(
            `Review gate: verify ${conditions.join(", ")}.`
          );
        }
        break;
      }
      case "output-node": {
        const path = (data.path as string) ?? "";
        const format = (data.format as string) ?? "";
        // ambient.json results format: { "Label": "glob/pattern" }
        // Use format as the label (or derive from path), path as the glob pattern
        const label = format || path.split("/").pop() || "Output";
        if (path) {
          results[label] = path;
        }
        break;
      }
    }
  }

  return {
    name: "Untitled Workflow",
    description: "",
    systemPrompt: promptParts.join("\n\n"),
    startupPrompt: promptParts[0] ?? "",
    results,
  };
}
