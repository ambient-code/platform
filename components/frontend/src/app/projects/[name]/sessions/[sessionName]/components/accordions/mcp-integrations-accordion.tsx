'use client'

import { useState } from 'react'
import { Plug, Check, Loader2 } from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

type McpIntegrationsAccordionProps = {
  projectName: string
  sessionName: string
}

export function McpIntegrationsAccordion({
  projectName,
  sessionName,
}: McpIntegrationsAccordionProps) {
  const [googleConnected, setGoogleConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)

  const handleConnectGoogle = async () => {
    setConnecting(true)

    try {
      // Call backend to get OAuth URL
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/oauth/google/url`
      )

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to get OAuth URL')
      }

      const data = await response.json()
      const authUrl = data.url

      // Open OAuth flow in popup window
      const width = 600
      const height = 700
      const left = window.screen.width / 2 - width / 2
      const top = window.screen.height / 2 - height / 2

      const popup = window.open(
        authUrl,
        'Google OAuth',
        `width=${width},height=${height},left=${left},top=${top}`
      )

      // Poll for popup close (credentials will be stored server-side)
      const pollTimer = setInterval(() => {
        if (popup?.closed) {
          clearInterval(pollTimer)
          setConnecting(false)
          // TODO: Check if credentials were successfully stored
          setGoogleConnected(true)
        }
      }, 500)
    } catch (error) {
      console.error('Failed to initiate Google OAuth:', error)
      setConnecting(false)
    }
  }

  const handleDisconnectGoogle = () => {
    // TODO: Implement disconnect - remove credentials from session
    setGoogleConnected(false)
  }

  return (
    <AccordionItem value="mcp-integrations" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Plug className="h-4 w-4" />
          <span>MCP Integrations</span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-3">
          {/* Google Drive Integration */}
          <div className="flex items-center justify-between p-3 border rounded-lg bg-background/50">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-white border border-gray-200 rounded flex items-center justify-center flex-shrink-0">
                <svg className="w-6 h-6" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    fill="#4285F4"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="#34A853"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                  />
                  <path
                    fill="#EA4335"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  />
                </svg>
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <h4 className="font-medium text-sm">Google Drive</h4>
                  {googleConnected && (
                    <Badge variant="outline" className="text-xs bg-green-50 text-green-700 border-green-200">
                      <Check className="h-3 w-3 mr-1" />
                      Connected
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Access Drive files in this session
                </p>
              </div>
            </div>
            <div className="flex-shrink-0">
              {googleConnected ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleDisconnectGoogle}
                  className="text-xs h-8"
                >
                  Disconnect
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={handleConnectGoogle}
                  disabled={connecting}
                  className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8"
                >
                  {connecting ? (
                    <>
                      <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                      Connecting...
                    </>
                  ) : (
                    'Connect'
                  )}
                </Button>
              )}
            </div>
          </div>

          {/* Placeholder for future MCP integrations */}
          <p className="text-xs text-muted-foreground text-center py-2">
            More integrations coming soon...
          </p>
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}
