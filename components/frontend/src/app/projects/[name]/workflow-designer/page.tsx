"use client";

import { useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { ReactFlowProvider } from "@xyflow/react";
import { ArrowLeft, Download, FileJson, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { Canvas, initialNodes, initialEdges } from "@/components/workflow-designer/Canvas";
import { NodePanel } from "@/components/workflow-designer/NodePanel";
import { DnDProvider } from "@/components/workflow-designer/contexts/DnDContext";
import { exportToAmbientJson } from "@/components/workflow-designer/workflow-export";
import type { Node, Edge } from "@xyflow/react";

export default function WorkflowDesignerPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  const [workflowName, setWorkflowName] = useState("Untitled Workflow");
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState(false);

  const nodesRef = useRef<Node[]>(initialNodes);
  const edgesRef = useRef<Edge[]>(initialEdges);

  function handleNodesChange(nodes: Node[]) {
    nodesRef.current = nodes;
  }

  function handleEdgesChange(edges: Edge[]) {
    edgesRef.current = edges;
  }

  function downloadJson(data: unknown, filename: string) {
    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  function handleSave() {
    setSaving(true);
    const flowData = {
      name: workflowName,
      nodes: nodesRef.current,
      edges: edgesRef.current,
    };
    downloadJson(flowData, `${workflowName.toLowerCase().replace(/\s+/g, "-")}.json`);
    setSaving(false);
  }

  function handleExport() {
    setExporting(true);
    const config = exportToAmbientJson(nodesRef.current, edgesRef.current);
    config.name = workflowName;
    downloadJson(config, "ambient.json");
    setExporting(false);
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      {/* Top bar */}
      <div className="border-b bg-card px-4 py-3 shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => router.push(`/projects/${projectName}`)}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <Breadcrumbs
              items={[
                { label: "Workspaces", href: "/projects" },
                { label: projectName, href: `/projects/${projectName}` },
                { label: "Workflow Designer" },
              ]}
            />
          </div>

          <div className="flex items-center gap-2">
            <Input
              value={workflowName}
              onChange={(e) => setWorkflowName(e.target.value)}
              className="w-52 h-8 text-sm"
            />
            <Button
              variant="outline"
              size="sm"
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Download className="mr-1.5 h-3.5 w-3.5" />
              )}
              Save
            </Button>
            <Button
              size="sm"
              onClick={handleExport}
              disabled={exporting}
            >
              {exporting ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
              ) : (
                <FileJson className="mr-1.5 h-3.5 w-3.5" />
              )}
              Export
            </Button>
          </div>
        </div>
      </div>

      {/* Designer area */}
      <div className="flex flex-1 min-h-0">
        <ReactFlowProvider>
          <DnDProvider>
            <aside className="border-r bg-card p-3 overflow-y-auto shrink-0">
              <NodePanel />
            </aside>
            <main className="flex-1 min-w-0">
              <Canvas
                onNodesChange={handleNodesChange}
                onEdgesChange={handleEdgesChange}
              />
            </main>
          </DnDProvider>
        </ReactFlowProvider>
      </div>
    </div>
  );
}
