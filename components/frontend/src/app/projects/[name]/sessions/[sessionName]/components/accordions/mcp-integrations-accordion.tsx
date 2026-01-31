'use client'

import { Plug, CheckCircle2, XCircle, AlertCircle, KeyRound, KeyRoundIcon } from 'lucide-react'
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
import { useMcpStatus } from '@/services/queries/use-mcp'
import type { McpServer } from '@/services/api/sessions'

type McpIntegrationsAccordionProps = {
  projectName: string
  sessionName: string
}

export function McpIntegrationsAccordion({
  projectName,
  sessionName,
}: McpIntegrationsAccordionProps) {
  // Fetch real MCP status from runner
  const { data: mcpStatus } = useMcpStatus(projectName, sessionName)
  const mcpServers = mcpStatus?.servers || []

  const getStatusIcon = (server: McpServer) => {
    // If we have auth info, use that for the icon
    if (server.authenticated !== undefined) {
      if (server.authenticated === true) {
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      } else if (server.authenticated === null) {
        // Null = needs refresh/uncertain state
        return <AlertCircle className="h-4 w-4 text-amber-500" />
      } else {
        // False = not authenticated
        return <KeyRound className="h-4 w-4 text-amber-500" />
      }
    }

    // Fall back to status-based icons
    switch (server.status) {
      case 'configured':
      case 'connected':
        return <CheckCircle2 className="h-4 w-4 text-blue-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'disconnected':
      default:
        return <AlertCircle className="h-4 w-4 text-gray-400" />
    }
  }

  const getAuthBadge = (server: McpServer) => {
    // If auth info is available, show auth status
    if (server.authenticated !== undefined) {
      if (server.authenticated === true) {
        return (
          <Badge variant="outline" className="text-xs bg-green-50 text-green-700 border-green-200">
            <KeyRoundIcon className="h-3 w-3 mr-1" />
            Authenticated
          </Badge>
        )
      } else if (server.authenticated === null) {
        // Null = needs refresh/uncertain state
        return (
          <Badge variant="outline" className="text-xs bg-amber-50 text-amber-700 border-amber-200">
            <AlertCircle className="h-3 w-3 mr-1" />
            Needs Refresh
          </Badge>
        )
      } else {
        return (
          <Badge variant="outline" className="text-xs bg-amber-50 text-amber-700 border-amber-200">
            <KeyRound className="h-3 w-3 mr-1" />
            Not Authenticated
          </Badge>
        )
      }
    }
    
    // Fall back to status-based badges
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

  return (
    <AccordionItem value="mcp-integrations" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Plug className="h-4 w-4" />
          <span>MCP Server Status</span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {mcpServers.length > 0 ? (
            mcpServers.map((server) => (
              <div
                key={server.name}
                className="flex items-center justify-between p-3 border rounded-lg bg-background/50"
              >
                <div className="flex items-center gap-3">
                  <div className="flex-shrink-0">
                    {getStatusIcon(server)}
                  </div>
                  <div className="flex-1">
                    <h4 className="font-medium text-sm">{server.displayName}</h4>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {server.authMessage || server.name}
                    </p>
                  </div>
                </div>
                <div className="flex-shrink-0">
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        {getAuthBadge(server)}
                      </TooltipTrigger>
                      {server.authMessage && (
                        <TooltipContent>
                          <p>{server.authMessage}</p>
                        </TooltipContent>
                      )}
                    </Tooltip>
                  </TooltipProvider>
                </div>
              </div>
            ))
          ) : (
            <div className="text-center py-4">
              <p className="text-xs text-muted-foreground">
                No MCP servers configured for this session
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                Configure MCP servers in your workflow or project settings
              </p>
            </div>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}
