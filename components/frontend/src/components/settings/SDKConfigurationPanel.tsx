import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import {
  Settings,
  Wrench,
  FileText,
  Server,
  CheckCircle,
  XCircle,
  Save,
  Eye,
  Code,
  AlertTriangle
} from 'lucide-react';

interface SDKConfiguration {
  model: string;
  maxTokens: number;
  temperature: number;
  permissionMode: 'acceptEdits' | 'prompt' | 'reject';
  allowedTools: string[];
  includePartialMessages: boolean;
  continueConversation: boolean;
  systemPrompt: string;
  mcpServers: Record<string, MCPServerConfig>;
}

interface MCPServerConfig {
  command: string;
  args: string[];
  env?: Record<string, string>;
  enabled: boolean;
}

const DEFAULT_CONFIG: SDKConfiguration = {
  model: 'claude-sonnet-4-5@20250929',
  maxTokens: 4096,
  temperature: 1.0,
  permissionMode: 'acceptEdits',
  allowedTools: ['Read', 'Write', 'Bash', 'Glob', 'Grep', 'Edit', 'MultiEdit', 'WebSearch'],
  includePartialMessages: true,
  continueConversation: true,
  systemPrompt: '',
  mcpServers: {},
};

const AVAILABLE_TOOLS = [
  'Read', 'Write', 'Bash', 'Glob', 'Grep', 'Edit', 'MultiEdit', 'WebSearch',
  'NotebookEdit', 'WebFetch'
];

const MODELS = [
  { value: 'claude-opus-4-5@20251101', label: 'Claude Opus 4.5 (Most capable)' },
  { value: 'claude-sonnet-4-5@20250929', label: 'Claude Sonnet 4.5 (Recommended)' },
  { value: 'claude-haiku-4-5@20251001', label: 'Claude Haiku 4.5 (Fast)' },
];

export function SDKConfigurationPanel() {
  const [config, setConfig] = useState<SDKConfiguration>(DEFAULT_CONFIG);
  const [saved, setSaved] = useState(false);
  const [previewMode, setPreviewMode] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [mcpTestResults, setMcpTestResults] = useState<Record<string, boolean>>({});

  useEffect(() => {
    loadConfiguration();
  }, []);

  const loadConfiguration = async () => {
    try {
      const response = await fetch('/api/sdk/configuration');
      if (response.ok) {
        const data = await response.json();
        setConfig(data);
      }
    } catch (error) {
      console.error('Failed to load SDK configuration:', error);
    }
  };

  const saveConfiguration = async () => {
    const errors = validateConfiguration();
    if (errors.length > 0) {
      setValidationErrors(errors);
      return;
    }

    try {
      const response = await fetch('/api/sdk/configuration', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });

      if (response.ok) {
        setSaved(true);
        setTimeout(() => setSaved(false), 3000);
      }
    } catch (error) {
      console.error('Failed to save SDK configuration:', error);
    }
  };

  const validateConfiguration = (): string[] => {
    const errors: string[] = [];

    if (config.maxTokens < 1 || config.maxTokens > 200000) {
      errors.push('Max tokens must be between 1 and 200,000');
    }

    if (config.temperature < 0 || config.temperature > 1) {
      errors.push('Temperature must be between 0 and 1');
    }

    if (config.allowedTools.length === 0) {
      errors.push('At least one tool must be enabled');
    }

    return errors;
  };

  const testMCPServer = async (serverName: string) => {
    try {
      const response = await fetch(`/api/sdk/mcp/test/${serverName}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config.mcpServers[serverName]),
      });

      const result = await response.json();
      setMcpTestResults(prev => ({ ...prev, [serverName]: result.connected }));
    } catch (error) {
      setMcpTestResults(prev => ({ ...prev, [serverName]: false }));
    }
  };

  const addMCPServer = () => {
    const name = prompt('Enter MCP server name:');
    if (!name) return;

    setConfig({
      ...config,
      mcpServers: {
        ...config.mcpServers,
        [name]: {
          command: '',
          args: [],
          enabled: true,
        },
      },
    });
  };

  const updateMCPServer = (name: string, updates: Partial<MCPServerConfig>) => {
    setConfig({
      ...config,
      mcpServers: {
        ...config.mcpServers,
        [name]: { ...config.mcpServers[name], ...updates },
      },
    });
  };

  const removeMCPServer = (name: string) => {
    const { [name]: _, ...rest } = config.mcpServers;
    setConfig({ ...config, mcpServers: rest });
  };

  const toggleTool = (tool: string) => {
    const tools = config.allowedTools.includes(tool)
      ? config.allowedTools.filter(t => t !== tool)
      : [...config.allowedTools, tool];
    setConfig({ ...config, allowedTools: tools });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold flex items-center gap-2">
            <Settings className="h-6 w-6" />
            Claude Agent SDK Configuration
          </h2>
          <p className="text-muted-foreground">
            Configure SDK options, tools, MCP servers, and system prompts
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => setPreviewMode(!previewMode)}
          >
            {previewMode ? <Settings className="h-4 w-4" /> : <Code className="h-4 w-4" />}
            {previewMode ? 'Form View' : 'JSON Preview'}
          </Button>
          <Button onClick={saveConfiguration} className="gap-2">
            <Save className="h-4 w-4" />
            Save Configuration
          </Button>
        </div>
      </div>

      {saved && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>
            Configuration saved successfully. Changes will apply to new sessions.
          </AlertDescription>
        </Alert>
      )}

      {validationErrors.length > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <ul className="list-disc list-inside">
              {validationErrors.map((error, i) => (
                <li key={i}>{error}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {previewMode ? (
        <Card>
          <CardHeader>
            <CardTitle>JSON Preview</CardTitle>
            <CardDescription>
              Raw configuration that will be passed to ClaudeAgentOptions
            </CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="bg-muted p-4 rounded-md overflow-auto text-sm">
              {JSON.stringify(config, null, 2)}
            </pre>
          </CardContent>
        </Card>
      ) : (
        <Tabs defaultValue="model" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="model">Model & Limits</TabsTrigger>
            <TabsTrigger value="tools">Tools</TabsTrigger>
            <TabsTrigger value="mcp">MCP Servers</TabsTrigger>
            <TabsTrigger value="prompts">System Prompts</TabsTrigger>
          </TabsList>

          <TabsContent value="model" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Model Configuration</CardTitle>
                <CardDescription>
                  Select the Claude model and generation parameters
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="model">Model</Label>
                  <Select
                    value={config.model}
                    onValueChange={(value) => setConfig({ ...config, model: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {MODELS.map(model => (
                        <SelectItem key={model.value} value={model.value}>
                          {model.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="maxTokens">Max Tokens</Label>
                  <Input
                    id="maxTokens"
                    type="number"
                    min={1}
                    max={200000}
                    value={config.maxTokens}
                    onChange={(e) => setConfig({ ...config, maxTokens: parseInt(e.target.value) })}
                  />
                  <p className="text-sm text-muted-foreground">
                    Maximum tokens to generate (1-200,000)
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="temperature">Temperature</Label>
                  <Input
                    id="temperature"
                    type="number"
                    step={0.1}
                    min={0}
                    max={1}
                    value={config.temperature}
                    onChange={(e) => setConfig({ ...config, temperature: parseFloat(e.target.value) })}
                  />
                  <p className="text-sm text-muted-foreground">
                    Sampling temperature (0-1). Higher values = more creative.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="permissionMode">Permission Mode</Label>
                  <Select
                    value={config.permissionMode}
                    onValueChange={(value: any) => setConfig({ ...config, permissionMode: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="acceptEdits">Accept Edits (Auto-approve)</SelectItem>
                      <SelectItem value="prompt">Prompt (Ask user)</SelectItem>
                      <SelectItem value="reject">Reject (Read-only)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>Streaming (Partial Messages)</Label>
                    <p className="text-sm text-muted-foreground">
                      Enable real-time streaming of responses
                    </p>
                  </div>
                  <Switch
                    checked={config.includePartialMessages}
                    onCheckedChange={(checked) =>
                      setConfig({ ...config, includePartialMessages: checked })
                    }
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>Continue Conversation</Label>
                    <p className="text-sm text-muted-foreground">
                      Resume from disk state between runs
                    </p>
                  </div>
                  <Switch
                    checked={config.continueConversation}
                    onCheckedChange={(checked) =>
                      setConfig({ ...config, continueConversation: checked })
                    }
                  />
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="tools" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Allowed Tools</CardTitle>
                <CardDescription>
                  Select which tools Claude can use during execution
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4">
                  {AVAILABLE_TOOLS.map(tool => (
                    <div key={tool} className="flex items-center justify-between p-3 border rounded-md">
                      <div>
                        <Label className="font-medium">{tool}</Label>
                      </div>
                      <Switch
                        checked={config.allowedTools.includes(tool)}
                        onCheckedChange={() => toggleTool(tool)}
                      />
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="mcp" className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>MCP Servers</CardTitle>
                    <CardDescription>
                      Configure Model Context Protocol integrations
                    </CardDescription>
                  </div>
                  <Button onClick={addMCPServer} variant="outline" size="sm">
                    Add Server
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {Object.keys(config.mcpServers).length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    No MCP servers configured. Click "Add Server" to add one.
                  </div>
                ) : (
                  Object.entries(config.mcpServers).map(([name, server]) => (
                    <Card key={name}>
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <CardTitle className="text-lg">{name}</CardTitle>
                            {mcpTestResults[name] !== undefined && (
                              <Badge variant={mcpTestResults[name] ? 'default' : 'destructive'}>
                                {mcpTestResults[name] ? (
                                  <><CheckCircle className="h-3 w-3 mr-1" />Connected</>
                                ) : (
                                  <><XCircle className="h-3 w-3 mr-1" />Failed</>
                                )}
                              </Badge>
                            )}
                          </div>
                          <div className="flex gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => testMCPServer(name)}
                            >
                              Test
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => removeMCPServer(name)}
                            >
                              Remove
                            </Button>
                          </div>
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
                        <div className="flex items-center justify-between">
                          <Label>Enabled</Label>
                          <Switch
                            checked={server.enabled}
                            onCheckedChange={(checked) =>
                              updateMCPServer(name, { enabled: checked })
                            }
                          />
                        </div>

                        <div className="space-y-2">
                          <Label>Command</Label>
                          <Input
                            value={server.command}
                            onChange={(e) =>
                              updateMCPServer(name, { command: e.target.value })
                            }
                            placeholder="mcp-server-command"
                          />
                        </div>

                        <div className="space-y-2">
                          <Label>Arguments (JSON array)</Label>
                          <Input
                            value={JSON.stringify(server.args)}
                            onChange={(e) => {
                              try {
                                const args = JSON.parse(e.target.value);
                                updateMCPServer(name, { args });
                              } catch {}
                            }}
                            placeholder='["--option", "value"]'
                          />
                        </div>

                        <div className="space-y-2">
                          <Label>Environment Variables (JSON object)</Label>
                          <Input
                            value={JSON.stringify(server.env || {})}
                            onChange={(e) => {
                              try {
                                const env = JSON.parse(e.target.value);
                                updateMCPServer(name, { env });
                              } catch {}
                            }}
                            placeholder='{"KEY": "value"}'
                          />
                        </div>
                      </CardContent>
                    </Card>
                  ))
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="prompts" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>System Prompt</CardTitle>
                <CardDescription>
                  Custom instructions injected into every SDK session
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Textarea
                  value={config.systemPrompt}
                  onChange={(e) => setConfig({ ...config, systemPrompt: e.target.value })}
                  placeholder="Add custom system instructions here..."
                  className="min-h-[200px] font-mono"
                />
                <p className="text-sm text-muted-foreground mt-2">
                  This will be merged with workspace context and workflow instructions.
                  Leave empty to use default workspace prompt.
                </p>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}
