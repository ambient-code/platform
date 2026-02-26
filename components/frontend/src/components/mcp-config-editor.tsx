"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { McpServersTab } from "@/components/mcp-servers-tab";

type McpConfigEditorProps = {
  projectName: string;
};

export function McpConfigEditor({ projectName }: McpConfigEditorProps) {
  return (
    <Card className="flex-1">
      <CardHeader>
        <CardTitle>MCP Servers</CardTitle>
        <CardDescription>Configure Model Context Protocol servers for your sessions</CardDescription>
      </CardHeader>
      <CardContent>
        <McpServersTab projectName={projectName} />
      </CardContent>
    </Card>
  );
}
