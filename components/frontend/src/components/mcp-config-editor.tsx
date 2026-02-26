"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { McpServersTab } from "@/components/mcp-servers-tab";
import { HttpToolsTab } from "@/components/http-tools-tab";

type McpConfigEditorProps = {
  projectName: string;
};

export function McpConfigEditor({ projectName }: McpConfigEditorProps) {
  return (
    <Card className="flex-1">
      <CardHeader>
        <CardTitle>MCP Servers & HTTP Tools</CardTitle>
        <CardDescription>Configure Model Context Protocol servers and custom HTTP tools for your sessions</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="mcp-servers">
          <TabsList>
            <TabsTrigger value="mcp-servers">MCP Servers</TabsTrigger>
            <TabsTrigger value="http-tools">HTTP Tools</TabsTrigger>
          </TabsList>
          <TabsContent value="mcp-servers" className="mt-4">
            <McpServersTab projectName={projectName} />
          </TabsContent>
          <TabsContent value="http-tools" className="mt-4">
            <HttpToolsTab projectName={projectName} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
