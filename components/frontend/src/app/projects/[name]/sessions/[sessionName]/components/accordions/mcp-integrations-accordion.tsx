'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Plug, Link2, CheckCircle2, XCircle, AlertCircle, AlertTriangle, Info, Check, X } from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { useMcpStatus } from '@/services/queries/use-mcp'
import { useIntegrationsStatus } from '@/services/queries/use-integrations'
import type { McpServer, McpTool } from '@/services/api/sessions'

// ─── MCP Servers Accordion ───────────────────────────────────────────────────

type McpServersAccordionProps = {
  projectName: string
  sessionName: string
  sessionPhase?: string
}

export function McpServersAccordion({
  projectName,
  sessionName,
  sessionPhase,
}: McpServersAccordionProps) {
  const [placeholderTimedOut, setPlaceholderTimedOut] = useState(false)

  // Only fetch MCP status once the session is actually running (runner pod ready)
  const isRunning = sessionPhase === 'Running'
  const { data: mcpStatus, isPending: mcpPending } = useMcpStatus(projectName, sessionName, isRunning)
  const mcpServers = mcpStatus?.servers || []

  const showPlaceholders =
    !isRunning || mcpPending || (mcpServers.length === 0 && !placeholderTimedOut)

  useEffect(() => {
    if (mcpServers.length > 0) {
      setPlaceholderTimedOut(false)
      return
    }
    if (!isRunning || !mcpStatus) return
    const t = setTimeout(() => setPlaceholderTimedOut(true), 15 * 1000)
    return () => clearTimeout(t)
  }, [mcpStatus, mcpServers.length, isRunning])

  const getStatusIcon = (server: McpServer) => {
    switch (server.status) {
      case 'configured':
      case 'connected':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'disconnected':
      default:
        return <AlertCircle className="h-4 w-4 text-gray-400" />
    }
  }

  const getStatusBadge = (server: McpServer) => {
    switch (server.status) {
      case 'configured':
        return (
          <Badge variant="outline" className="text-xs bg-blue-50 text-blue-700 border-blue-200">
            Configured
          </Badge>
        )
      case 'connected':
        return (
          <Badge variant="outline" className="text-xs bg-green-50 text-green-700 border-green-200">
            Connected
          </Badge>
        )
      case 'error':
        return (
          <Badge variant="outline" className="text-xs bg-red-50 text-red-700 border-red-200">
            Error
          </Badge>
        )
      case 'disconnected':
      default:
        return (
          <Badge variant="outline" className="text-xs bg-gray-50 text-gray-700 border-gray-200">
            Disconnected
          </Badge>
        )
    }
  }

  const renderCardSkeleton = () => (
    <div
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      aria-hidden
    >
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full flex-shrink-0" />
          <Skeleton className="h-4 w-20" />
        </div>
        <Skeleton className="h-3 w-full max-w-[240px]" />
      </div>
    </div>
  )

  const renderAnnotationBadge = (key: string, value: boolean) => (
    <Badge
      key={key}
      variant="outline"
      className={`text-[10px] px-1.5 py-0 font-normal gap-0.5 ${
        value
          ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-950/30 dark:text-green-400 dark:border-green-800'
          : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-950/30 dark:text-red-400 dark:border-red-800'
      }`}
    >
      {value ? <Check className="h-2.5 w-2.5" /> : <X className="h-2.5 w-2.5" />}
      {key}
    </Badge>
  )

  const renderToolRow = (tool: McpTool) => {
    const annotations = Object.entries(tool.annotations).filter(
      ([, v]) => typeof v === 'boolean'
    )
    return (
      <div key={tool.name} className="flex items-center justify-between gap-3 px-3 py-2">
        <code className="text-xs truncate">{tool.name}</code>
        {annotations.length > 0 && (
          <div className="flex items-center gap-1 flex-shrink-0">
            {annotations.map(([k, v]) => renderAnnotationBadge(k, v as boolean))}
          </div>
        )}
      </div>
    )
  }

  const renderServerCard = (server: McpServer) => {
    const tools = server.tools ?? []
    const toolCount = tools.length

    return (
      <div
        key={server.name}
        className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <div className="flex-shrink-0">
              {getStatusIcon(server)}
            </div>
            <h4 className="font-medium text-sm">{server.displayName}</h4>
          </div>
          <div className="flex items-center gap-2 mt-1 ml-6">
            {server.version && (
              <span className="text-[10px] text-muted-foreground">v{server.version}</span>
            )}
            {toolCount > 0 && (
              <Popover>
                <PopoverTrigger asChild>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
                  >
                    <Info className="h-3 w-3" />
                    <span>{toolCount} {toolCount === 1 ? 'tool' : 'tools'}</span>
                  </button>
                </PopoverTrigger>
                <PopoverContent
                  align="start"
                  className="w-80 p-0"
                >
                  <div className="px-3 py-2.5 border-b bg-muted/30">
                    <p className="text-xs font-medium">
                      {server.displayName} — {toolCount} {toolCount === 1 ? 'tool' : 'tools'}
                    </p>
                  </div>
                  <div className="max-h-72 overflow-y-auto">
                    {tools.map((tool) => renderToolRow(tool))}
                  </div>
                </PopoverContent>
              </Popover>
            )}
          </div>
        </div>
        <div className="flex-shrink-0">
          {getStatusBadge(server)}
        </div>
      </div>
    )
  }

  return (
    <AccordionItem value="mcp-servers" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Plug className="h-4 w-4" />
          <span>MCP Servers</span>
          {!showPlaceholders && mcpServers.length > 0 && (
            <Badge variant="outline" className="text-[10px] px-2 py-0.5">
              {mcpServers.length}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {showPlaceholders ? (
            <>
              {renderCardSkeleton()}
              {renderCardSkeleton()}
            </>
          ) : mcpServers.length > 0 ? (
            mcpServers.map((server) => renderServerCard(server))
          ) : (
            <p className="text-xs text-muted-foreground py-2">
              No MCP servers available for this session.
            </p>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

// ─── Integrations Accordion ──────────────────────────────────────────────────

export function IntegrationsAccordion() {
  const { data: integrationsStatus, isPending } = useIntegrationsStatus()

  const githubConfigured = integrationsStatus?.github?.active != null
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false
  const jiraConfigured = integrationsStatus?.jira?.connected ?? false
  const googleConfigured = integrationsStatus?.google?.connected ?? false

  const integrations = [
    {
      key: 'github',
      name: 'GitHub',
      configured: githubConfigured,
      configuredMessage: 'Authenticated. Git push and repository access enabled.',
    },
    {
      key: 'gitlab',
      name: 'GitLab',
      configured: gitlabConfigured,
      configuredMessage: 'Authenticated. Git push and repository access enabled.',
    },
    {
      key: 'google',
      name: 'Google Workspace',
      configured: googleConfigured,
      configuredMessage: 'Authenticated. Drive, Calendar, and Gmail access enabled.',
    },
    {
      key: 'jira',
      name: 'Jira',
      configured: jiraConfigured,
      configuredMessage: 'Authenticated. Issue and project access enabled.',
    },
  ].sort((a, b) => a.name.localeCompare(b.name))

  const configuredCount = integrations.filter((i) => i.configured).length

  const renderCardSkeleton = () => (
    <div
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      aria-hidden
    >
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full flex-shrink-0" />
          <Skeleton className="h-4 w-20" />
        </div>
        <Skeleton className="h-3 w-full max-w-[240px]" />
      </div>
    </div>
  )

  const renderIntegrationCard = (integration: (typeof integrations)[number]) => (
    <div
      key={integration.key}
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {integration.configured ? (
              <CheckCircle2 className="h-4 w-4 text-green-600" />
            ) : (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
          <h4 className="font-medium text-sm">{integration.name}</h4>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {integration.configured ? (
            integration.configuredMessage
          ) : (
            <>
              Not connected.{' '}
              <Link href="/integrations" className="text-primary hover:underline">
                Set up
              </Link>{' '}
              to enable {integration.name} access.
            </>
          )}
        </p>
      </div>
    </div>
  )

  return (
    <AccordionItem value="integrations" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Link2 className="h-4 w-4" />
          <span>Integrations</span>
          {!isPending && (
            <Badge variant="outline" className="text-[10px] px-2 py-0.5">
              {configuredCount}/{integrations.length}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {isPending ? (
            <>
              {renderCardSkeleton()}
              {renderCardSkeleton()}
              {renderCardSkeleton()}
            </>
          ) : (
            integrations.map((integration) => renderIntegrationCard(integration))
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

// ─── Legacy export (renders both) ────────────────────────────────────────────

type McpIntegrationsAccordionProps = {
  projectName: string
  sessionName: string
}

/** @deprecated Use McpServersAccordion + IntegrationsAccordion separately */
export function McpIntegrationsAccordion({
  projectName,
  sessionName,
}: McpIntegrationsAccordionProps) {
  return (
    <>
      <McpServersAccordion projectName={projectName} sessionName={sessionName} />
      <IntegrationsAccordion />
    </>
  )
}
