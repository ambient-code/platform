"use client";

import { type DragEvent, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  useNodesState,
  useEdgesState,
  useReactFlow,
  type Connection,
  type Node,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { nodeTypes } from "./nodes";
import { edgeTypes } from "./edges";
import { useDnD } from "./contexts/DnDContext";

const initialNodes: Node[] = [
  {
    id: "start-1",
    type: "start-node",
    position: { x: 250, y: 50 },
    data: {},
  },
];

const initialEdges: Edge[] = [];

type CanvasProps = {
  onNodesChange?: (nodes: Node[]) => void;
  onEdgesChange?: (edges: Edge[]) => void;
};

export function Canvas({ onNodesChange, onEdgesChange }: CanvasProps) {
  const [nodes, setNodes, handleNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, handleEdgesChange] = useEdgesState(initialEdges);
  const { screenToFlowPosition } = useReactFlow();
  const { type } = useDnD();

  // Report state changes to parent after ReactFlow applies them
  useEffect(() => {
    onNodesChange?.(nodes);
  }, [nodes, onNodesChange]);

  useEffect(() => {
    onEdgesChange?.(edges);
  }, [edges, onEdgesChange]);

  function handleConnect(connection: Connection) {
    if (connection.source === connection.target) return;
    setEdges((eds) => addEdge(connection, eds));
  }

  function handleDragOver(event: DragEvent) {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault();
    if (!type) return;

    const position = screenToFlowPosition({
      x: event.clientX,
      y: event.clientY,
    });

    const newNode: Node = {
      id: `${type}-${Date.now()}`,
      type,
      position,
      data: {},
    };

    setNodes((nds) => [...nds, newNode]);
  }

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        onConnect={handleConnect}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        snapToGrid
        snapGrid={[16, 16]}
        fitView
        defaultEdgeOptions={{
          type: "default",
        }}
      >
        <Background gap={16} size={1} />
        <Controls showInteractive={false} />
        <MiniMap
          nodeStrokeWidth={3}
          className="rounded-lg border"
          nodeColor={(node) => {
            switch (node.type) {
              case "start-node":
                return "#10b981";
              case "prompt-node":
                return "#3b82f6";
              case "tool-node":
                return "#f97316";
              case "review-node":
                return "#a855f7";
              case "output-node":
                return "#eab308";
              default:
                return "#64748b";
            }
          }}
        />
      </ReactFlow>
    </div>
  );
}

export { initialNodes, initialEdges };
