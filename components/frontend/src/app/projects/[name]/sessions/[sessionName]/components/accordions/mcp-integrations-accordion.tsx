'use client'

import { Plug, CheckCircle2, XCircle, AlertCircle } from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { useMcpStatus } from '@/services/queries/use-mcp'

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
  const getStatusIcon = (status: 'connected' | 'disconnected' | 'error') => {
    switch (status) {
      case 'connected':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'disconnected':
      default:
        return <AlertCircle className="h-4 w-4 text-gray-400" />
    }
  }

  const getStatusBadge = (status: 'connected' | 'disconnected' | 'error') => {
    switch (status) {
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
                    {getStatusIcon(server.status)}
                  </div>
                  <div className="flex-1">
                    <h4 className="font-medium text-sm">{server.displayName}</h4>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {server.name}
                    </p>
                  </div>
                </div>
                <div className="flex-shrink-0">
                  {getStatusBadge(server.status)}
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
