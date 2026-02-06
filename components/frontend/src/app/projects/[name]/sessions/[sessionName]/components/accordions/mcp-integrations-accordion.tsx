'use client'

import { useState, useEffect } from 'react'
import type { ReactNode } from 'react'
import Link from 'next/link'
import { Plug, CheckCircle2, XCircle, AlertCircle, AlertTriangle } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { useMcpStatus } from '@/services/queries/use-mcp'
import { useProjectIntegrationStatus } from '@/services/queries/use-projects'
import { useIntegrationsStatus } from '@/services/queries/use-integrations'
import type { McpServer } from '@/services/api/sessions'

type McpIntegrationsAccordionProps = {
  projectName: string
  sessionName: string
}

export function McpIntegrationsAccordion({
  projectName,
  sessionName,
}: McpIntegrationsAccordionProps) {
  const [placeholderTimedOut, setPlaceholderTimedOut] = useState(false)

  // Fetch real MCP status from runner
  const { data: mcpStatus, isPending: mcpPending } = useMcpStatus(projectName, sessionName)
  const mcpServers = mcpStatus?.servers || []

  const { data: integrationStatus, isPending: integrationStatusPending } =
    useProjectIntegrationStatus(projectName)
  const githubConfigured = integrationStatus?.github ?? false

  const { data: integrationsStatus } = useIntegrationsStatus()
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false

  // Show skeleton cards until we have MCP servers or 2 min elapsed (backend returns empty when runner not ready)
  const showPlaceholders =
    mcpPending || (mcpServers.length === 0 && !placeholderTimedOut)

  useEffect(() => {
    if (mcpServers.length > 0) {
      setPlaceholderTimedOut(false)
      return
    }
    if (!mcpStatus) return
    const t = setTimeout(() => setPlaceholderTimedOut(true), 15 * 1000) // 15 seconds
    return () => clearTimeout(t)
  }, [mcpStatus, mcpServers.length])

  // Collect all MCP servers
  const allServers = [...mcpServers]

  // Ensure core integrations always appear (even if not in API response)
  if (!showPlaceholders) {
    // Webfetch - always available
    const hasWebfetch = allServers.some((s) => s.name === 'webfetch')
    if (!hasWebfetch) {
      allServers.push({
        name: 'webfetch',
        displayName: 'Webfetch',
        status: 'disconnected',
        authenticated: undefined,
        authMessage: 'Fetches web content for the session.',
      } as McpServer)
    }

    // Google Workspace - show as not configured if missing
    const hasGoogleWorkspace = allServers.some((s) => s.name === 'google-workspace')
    if (!hasGoogleWorkspace) {
      allServers.push({
        name: 'google-workspace',
        displayName: 'Google Workspace',
        status: 'disconnected',
        authenticated: false,
        authMessage: undefined,
      } as McpServer)
    }

    // Jira - workspace-level integration
    const hasJira = allServers.some((s) => s.name === 'mcp-atlassian')
    if (!hasJira) {
      allServers.push({
        name: 'mcp-atlassian',
        displayName: 'Jira',
        status: 'disconnected',
        authenticated: false,
        authMessage: undefined,
      } as McpServer)
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

  const renderGitHubCard = () =>
    integrationStatusPending ? (
      renderCardSkeleton()
    ) : (
    <div
      key="github"
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {githubConfigured ? (
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
                    <p>not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
          <h4 className="font-medium text-sm">GitHub</h4>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {githubConfigured ? (
            'MCP access to GitHub repositories.'
          ) : (
            <>
              Session started without GitHub MCP. Configure{' '}
              <Link href="/integrations" className="text-primary hover:underline">
                Integrations
              </Link>{' '}
              and start a new session.
            </>
          )}
        </p>
      </div>
    </div>
    )

  const renderGitLabCard = () =>
    integrationStatusPending ? (
      renderCardSkeleton()
    ) : (
    <div
      key="gitlab"
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {gitlabConfigured ? (
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
                    <p>not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
          <h4 className="font-medium text-sm">GitLab</h4>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {gitlabConfigured ? (
            'MCP access to GitLab repositories.'
          ) : (
            <>
              Session started without GitLab MCP. Configure{' '}
              <Link href="/integrations" className="text-primary hover:underline">
                Integrations
              </Link>{' '}
              and start a new session.
            </>
          )}
        </p>
      </div>
    </div>
    )

  const renderServerCard = (server: McpServer) => (
    <div
      key={server.name}
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {server.authenticated === false ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">{getStatusIcon(server)}</span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              getStatusIcon(server)
            )}
          </div>
          <h4 className="font-medium text-sm">{getDisplayName(server)}</h4>
{server.name === 'mcp-atlassian' && server.authenticated === true && (
                      <Badge variant="secondary" className="text-xs font-normal">
                        read only
                      </Badge>
                    )}
        </div>
        {getDescription(server) && (
          <p className="text-xs text-muted-foreground mt-0.5">
            {getDescription(server)}
          </p>
        )}
      </div>
      <div className="flex-shrink-0">
        {getRightContent(server)}
      </div>
    </div>
  )

  const getDisplayName = (server: McpServer) =>
    server.name === 'mcp-atlassian' ? 'Jira' : server.displayName

  const getDescription = (server: McpServer): ReactNode => {
    if (server.name === 'webfetch') return 'Fetches web content for the session.'
    if (server.name === 'mcp-atlassian') {
      if (server.authenticated === false) {
        return (
          <>
            Session started without Jira MCP. Configure{' '}
            <Link href="/integrations" className="text-primary hover:underline">
              Integrations
            </Link>{' '}
            and start a new session.
          </>
        )
      }
      return 'MCP access to Jira issues and projects.'
    }
    if (server.name === 'google-workspace') {
      if (server.authenticated === false) {
        return (
          <>
            Session started without Google Workspace MCP. Configure{' '}
            <Link href="/integrations" className="text-primary hover:underline">
              Integrations
            </Link>{' '}
            and start a new session.
          </>
        )
      }
      return 'MCP access to Google Drive files.'
    }
    return server.authMessage ?? null
  }

  const getStatusIcon = (server: McpServer) => {
    // If we have auth info, use that for the icon
    if (server.authenticated !== undefined) {
      if (server.authenticated === true) {
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      } else if (server.authenticated === null) {
        // Null = needs refresh/uncertain state
        return <AlertCircle className="h-4 w-4 text-amber-500" />
      } else {
        // False = not authenticated/not configured
        return <AlertTriangle className="h-4 w-4 text-amber-500" />
      }
    }

    // Fall back to status-based icons
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

  const getRightContent = (server: McpServer) => {
    // Webfetch: no badge
    if (server.name === 'webfetch') return null

    // Jira not authenticated: no link (description explains to configure and start new session)

    // Google Workspace not authenticated: no link (description explains to configure and start new session)

    // Jira connected: no badge
    if (server.name === 'mcp-atlassian' && server.authenticated === true) return null

    // Authenticated: show badge (with optional tooltip)
    if (server.authenticated === true) {
      const badge = (
        <Badge variant="outline" className="text-xs bg-green-50 text-green-700 border-green-200">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Authenticated
        </Badge>
      )
      if (server.authMessage) {
        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>{badge}</TooltipTrigger>
              <TooltipContent>
                <p>{server.authMessage}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )
      }
      return badge
    }

    // Other servers with auth status but not authenticated: no badge (only Jira/Google get links above)
    if (server.authenticated === false) return null

    // Fall back to status-based badges (for servers without auth info; webfetch already returns null)
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

  // Combine all integrations (GitHub + GitLab + all MCP servers)
  type IntegrationItem =
    | { type: 'github'; displayName: string }
    | { type: 'gitlab'; displayName: string }
    | { type: 'server'; displayName: string; server: McpServer }
  const allIntegrations: IntegrationItem[] = [
    { type: 'github' as const, displayName: 'GitHub' },
    { type: 'gitlab' as const, displayName: 'GitLab' },
    ...allServers.map((server) => ({ type: 'server' as const, displayName: getDisplayName(server), server })),
  ].sort((a, b) => a.displayName.localeCompare(b.displayName))

  return (
    <>
    <AccordionItem value="mcp-integrations" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Plug className="h-4 w-4" />
          <span>Integrations</span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {showPlaceholders ? (
            <>
              {renderCardSkeleton()}
              {renderCardSkeleton()}
            </>
          ) : (
            allIntegrations.map((item) => {
              if (item.type === 'github') {
                return <div key="github">{renderGitHubCard()}</div>
              } else if (item.type === 'gitlab') {
                return <div key="gitlab">{renderGitLabCard()}</div>
              } else {
                return renderServerCard(item.server)
              }
            })
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
    </>
  )
}
